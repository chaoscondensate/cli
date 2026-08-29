// Package rfc3161 implements the bounded Forecast Ledger RFC 3161 profile.
// Dependency-specific types do not cross this package boundary.
package rfc3161

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"time"

	tspclient "github.com/notaryproject/tspclient-go"
)

const (
	HashAlgorithm        = "sha256"
	MaxTargetBytes       = 1 << 20
	MaxRequestBytes      = 64 << 10
	MaxResponseBytes     = 1 << 20
	MaxCABundleBytes     = 1 << 20
	MaxCertificates      = 64
	MaxSignerInfos       = 8
	MaxExtensions        = 32
	defaultNonceByteSize = 20
)

var (
	oidSHA256           = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidExtendedKeyUsage = asn1.ObjectIdentifier{2, 5, 29, 37}
)

// Reason is a stable safe RFC 3161 failure category.
type Reason string

const (
	ReasonLimit             Reason = "rfc3161.limit"
	ReasonRequestMalformed  Reason = "rfc3161.request_malformed"
	ReasonRequestProfile    Reason = "rfc3161.request_profile"
	ReasonTargetMismatch    Reason = "rfc3161.target_mismatch"
	ReasonResponseMalformed Reason = "rfc3161.response_malformed"
	ReasonResponseRejected  Reason = "rfc3161.response_rejected"
	ReasonBindingMismatch   Reason = "rfc3161.binding_mismatch"
	ReasonTokenMalformed    Reason = "rfc3161.token_malformed"
	ReasonAlgorithm         Reason = "rfc3161.algorithm_unsupported"
	ReasonTrustBundle       Reason = "rfc3161.trust_bundle"
	ReasonSignature         Reason = "rfc3161.signature"
	ReasonCertificate       Reason = "rfc3161.certificate"
	ReasonMetadata          Reason = "rfc3161.metadata"
	ReasonEntropy           Reason = "rfc3161.entropy"
)

// Error deliberately contains no request, response, certificate, target, or
// credential bytes. The underlying parser error is not exposed to callers.
type Error struct {
	Reason  Reason
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return "RFC 3161 error"
	}
	return e.Message
}

func failure(reason Reason, message string) error {
	return &Error{Reason: reason, Message: message}
}

// Limits defines application-owned byte and collection limits.
type Limits struct {
	TargetBytes   int
	RequestBytes  int
	ResponseBytes int
	CABundleBytes int
	Certificates  int
	SignerInfos   int
	Extensions    int
}

func DefaultLimits() Limits {
	return Limits{
		TargetBytes: MaxTargetBytes, RequestBytes: MaxRequestBytes,
		ResponseBytes: MaxResponseBytes, CABundleBytes: MaxCABundleBytes,
		Certificates: MaxCertificates, SignerInfos: MaxSignerInfos,
		Extensions: MaxExtensions,
	}
}

func (l Limits) normalized() Limits {
	d := DefaultLimits()
	if l.TargetBytes > 0 {
		d.TargetBytes = l.TargetBytes
	}
	if l.RequestBytes > 0 {
		d.RequestBytes = l.RequestBytes
	}
	if l.ResponseBytes > 0 {
		d.ResponseBytes = l.ResponseBytes
	}
	if l.CABundleBytes > 0 {
		d.CABundleBytes = l.CABundleBytes
	}
	if l.Certificates > 0 {
		d.Certificates = l.Certificates
	}
	if l.SignerInfos > 0 {
		d.SignerInfos = l.SignerInfos
	}
	if l.Extensions > 0 {
		d.Extensions = l.Extensions
	}
	return d
}

// Metadata is the safe verified projection retained in a Forecast Ledger.
type Metadata struct {
	HashAlgorithm     string    `json:"hash_algorithm"`
	GenTime           time.Time `json:"gen_time"`
	PolicyOID         string    `json:"policy_oid"`
	SerialNumber      string    `json:"serial_number"`
	SignerSubject     string    `json:"signer_subject"`
	SignerFingerprint string    `json:"signer_fingerprint_sha256"`
	CABundleSHA256    string    `json:"ca_bundle_sha256"`
}

// RequestInfo is a safe parsed request projection.
type RequestInfo struct {
	HashAlgorithm string `json:"hash_algorithm"`
	HasNonce      bool   `json:"has_nonce"`
	RequestsCert  bool   `json:"requests_certificate"`
}

// CreateRequest builds the exact supported request profile. Entropy must be a
// CSPRNG in production and is injectable only below adapters for tests.
func CreateRequest(target []byte, entropy io.Reader, limits Limits) ([]byte, RequestInfo, error) {
	limits = limits.normalized()
	if len(target) == 0 || len(target) > limits.TargetBytes {
		return nil, RequestInfo{}, failure(ReasonLimit, "timestamp target exceeds the supported size")
	}
	if entropy == nil {
		return nil, RequestInfo{}, failure(ReasonEntropy, "timestamp request entropy is unavailable")
	}
	nonceBytes := make([]byte, defaultNonceByteSize)
	if _, err := io.ReadFull(entropy, nonceBytes); err != nil {
		return nil, RequestInfo{}, failure(ReasonEntropy, "timestamp request nonce generation failed")
	}
	// Match the dependency's reviewed 159-bit positive nonce profile.
	nonceBytes[0] &= 0x7f
	nonce := new(big.Int).SetBytes(nonceBytes)
	if nonce.Sign() == 0 {
		nonce.SetInt64(1)
	}
	digest := sha256.Sum256(target)
	req := &tspclient.Request{
		Version: 1,
		MessageImprint: tspclient.MessageImprint{
			HashAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm:  oidSHA256,
				Parameters: asn1.RawValue{Tag: asn1.TagNull, FullBytes: []byte{asn1.TagNull, 0}},
			},
			HashedMessage: digest[:],
		},
		Nonce:   nonce,
		CertReq: true,
	}
	encoded, err := req.MarshalBinary()
	if err != nil || len(encoded) > limits.RequestBytes {
		return nil, RequestInfo{}, failure(ReasonRequestMalformed, "timestamp request could not be encoded within limits")
	}
	return encoded, RequestInfo{HashAlgorithm: HashAlgorithm, HasNonce: true, RequestsCert: true}, nil
}

// ParseRequest parses and enforces the supported request profile and target
// binding. It rejects trailing bytes that the dependency API otherwise ignores.
func ParseRequest(data, target []byte, limits Limits) (RequestInfo, error) {
	_, info, err := parseRequest(data, target, limits)
	return info, err
}

func parseRequest(data, target []byte, limits Limits) (*tspclient.Request, RequestInfo, error) {
	limits = limits.normalized()
	if len(data) == 0 || len(data) > limits.RequestBytes || len(target) == 0 || len(target) > limits.TargetBytes {
		return nil, RequestInfo{}, failure(ReasonLimit, "timestamp request or target exceeds the supported size")
	}
	var req tspclient.Request
	rest, err := asn1.Unmarshal(data, &req)
	if err != nil || len(rest) != 0 {
		return nil, RequestInfo{}, failure(ReasonRequestMalformed, "timestamp request is malformed or has trailing data")
	}
	if err := req.Validate(); err != nil {
		return nil, RequestInfo{}, failure(ReasonRequestMalformed, "timestamp request is not valid")
	}
	if !req.MessageImprint.HashAlgorithm.Algorithm.Equal(oidSHA256) || !req.CertReq || req.Nonce == nil || req.Nonce.Sign() <= 0 || req.Nonce.BitLen() > 159 || len(req.Extensions) != 0 {
		return nil, RequestInfo{}, failure(ReasonRequestProfile, "timestamp request is outside the supported SHA-256 nonce and certificate profile")
	}
	digest := sha256.Sum256(target)
	if !bytes.Equal(req.MessageImprint.HashedMessage, digest[:]) {
		return nil, RequestInfo{}, failure(ReasonTargetMismatch, "timestamp request does not bind the selected target")
	}
	return &req, RequestInfo{HashAlgorithm: HashAlgorithm, HasNonce: true, RequestsCert: true}, nil
}

// Verify checks the complete local target/request/response/trust chain.
func Verify(ctx context.Context, target, request, response, caBundle []byte, limits Limits) (Metadata, error) {
	limits = limits.normalized()
	req, _, err := parseRequest(request, target, limits)
	if err != nil {
		return Metadata{}, err
	}
	resp, err := parseResponse(response, limits)
	if err != nil {
		return Metadata{}, err
	}
	if err := resp.Validate(req); err != nil {
		return Metadata{}, failure(ReasonBindingMismatch, "timestamp response does not match the request")
	}
	token, err := resp.SignedToken()
	if err != nil {
		return Metadata{}, failure(ReasonTokenMalformed, "timestamp response does not contain a supported signed token")
	}
	if len(token.Certificates) == 0 || len(token.Certificates) > limits.Certificates || len(token.SignerInfos) != 1 || len(token.SignerInfos) > limits.SignerInfos {
		return Metadata{}, failure(ReasonLimit, "timestamp token certificate or signer count is outside limits")
	}
	for _, signerInfo := range token.SignerInfos {
		if !signerInfo.DigestAlgorithm.Algorithm.Equal(oidSHA256) {
			return Metadata{}, failure(ReasonAlgorithm, "timestamp token signer uses an unsupported digest algorithm")
		}
	}
	info, err := token.Info()
	if err != nil {
		return Metadata{}, failure(ReasonTokenMalformed, "timestamp token metadata is malformed")
	}
	if !info.MessageImprint.HashAlgorithm.Algorithm.Equal(oidSHA256) || len(info.Extensions) > limits.Extensions {
		return Metadata{}, failure(ReasonAlgorithm, "timestamp token uses an unsupported algorithm or extension profile")
	}
	if info.SerialNumber == nil || info.SerialNumber.Sign() < 0 || len(info.Policy) < 2 || info.GenTime.IsZero() || info.GenTime.Location() != time.UTC {
		return Metadata{}, failure(ReasonMetadata, "timestamp token metadata is incomplete or unsupported")
	}
	if _, err := info.Validate(target); err != nil {
		return Metadata{}, failure(ReasonTargetMismatch, "timestamp token does not bind the selected target")
	}
	roots, bundleDigest, err := parseCABundle(caBundle, limits)
	if err != nil {
		return Metadata{}, err
	}
	chain, err := token.Verify(ctx, x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: info.GenTime,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	})
	if err != nil || len(chain) == 0 {
		return Metadata{}, failure(ReasonSignature, "timestamp signature or certificate chain verification failed")
	}
	signer := chain[0]
	if !hasCriticalTimestampingEKU(signer) {
		return Metadata{}, failure(ReasonCertificate, "timestamp signer certificate lacks the required critical timestamping EKU")
	}
	for _, certificate := range chain {
		if weakSignature(certificate.SignatureAlgorithm) {
			return Metadata{}, failure(ReasonAlgorithm, "timestamp certificate chain uses an unsupported weak signature algorithm")
		}
	}
	if !strongPublicKey(signer) {
		return Metadata{}, failure(ReasonAlgorithm, "timestamp signer public key is outside the supported strength profile")
	}
	fingerprint := sha256.Sum256(signer.Raw)
	return Metadata{
		HashAlgorithm:     HashAlgorithm,
		GenTime:           info.GenTime,
		PolicyOID:         info.Policy.String(),
		SerialNumber:      info.SerialNumber.String(),
		SignerSubject:     signer.Subject.String(),
		SignerFingerprint: hex.EncodeToString(fingerprint[:]),
		CABundleSHA256:    bundleDigest,
	}, nil
}

func parseResponse(data []byte, limits Limits) (*tspclient.Response, error) {
	if len(data) == 0 || len(data) > limits.ResponseBytes {
		return nil, failure(ReasonLimit, "timestamp response exceeds the supported size")
	}
	var resp tspclient.Response
	rest, err := asn1.Unmarshal(data, &resp)
	if err != nil || len(rest) != 0 {
		return nil, failure(ReasonResponseMalformed, "timestamp response is malformed or has trailing data")
	}
	if _, err := resp.SignedToken(); err != nil {
		return nil, failure(ReasonResponseRejected, "timestamp response status is not successful or its token is malformed")
	}
	return &resp, nil
}

// ParseResponse performs bounded structural and successful-status parsing
// without claiming signature, trust-chain, request, or target verification.
func ParseResponse(data []byte, limits Limits) error {
	_, err := parseResponse(data, limits.normalized())
	return err
}

func parseCABundle(data []byte, limits Limits) (*x509.CertPool, string, error) {
	if len(data) == 0 || len(data) > limits.CABundleBytes {
		return nil, "", failure(ReasonLimit, "timestamp CA bundle exceeds the supported size")
	}
	pool := x509.NewCertPool()
	rest := data
	count := 0
	for len(bytes.TrimSpace(rest)) > 0 {
		block, remaining := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, "", failure(ReasonTrustBundle, "timestamp CA bundle must contain only PEM certificates")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, "", failure(ReasonTrustBundle, "timestamp CA bundle contains an invalid certificate")
		}
		count++
		if count > limits.Certificates {
			return nil, "", failure(ReasonLimit, "timestamp CA bundle contains too many certificates")
		}
		pool.AddCert(cert)
		rest = remaining
	}
	if count == 0 {
		return nil, "", failure(ReasonTrustBundle, "timestamp CA bundle contains no certificates")
	}
	digest := sha256.Sum256(data)
	return pool, hex.EncodeToString(digest[:]), nil
}

// ValidateCABundle performs bounded PEM and certificate parsing without using
// system roots or making a network request.
func ValidateCABundle(data []byte, limits Limits) error {
	_, _, err := parseCABundle(data, limits.normalized())
	return err
}

func weakSignature(algorithm x509.SignatureAlgorithm) bool {
	switch algorithm {
	case x509.MD2WithRSA, x509.MD5WithRSA, x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1, x509.UnknownSignatureAlgorithm:
		return true
	default:
		return false
	}
}

func strongPublicKey(certificate *x509.Certificate) bool {
	if certificate == nil {
		return false
	}
	switch key := certificate.PublicKey.(type) {
	case *rsa.PublicKey:
		return key.N != nil && key.N.BitLen() >= 2048
	case *ecdsa.PublicKey:
		return key.Curve != nil && key.Curve.Params() != nil && key.Curve.Params().BitSize >= 256
	case ed25519.PublicKey:
		return len(key) == ed25519.PublicKeySize
	default:
		return false
	}
}

func hasCriticalTimestampingEKU(certificate *x509.Certificate) bool {
	if certificate == nil || len(certificate.ExtKeyUsage) != 1 || certificate.ExtKeyUsage[0] != x509.ExtKeyUsageTimeStamping || len(certificate.UnknownExtKeyUsage) != 0 {
		return false
	}
	for _, extension := range certificate.Extensions {
		if extension.Id.Equal(oidExtendedKeyUsage) {
			return extension.Critical
		}
	}
	return false
}

// MetadataMatches compares parsed verified metadata with one ledger entry.
func MetadataMatches(metadata Metadata, genTime, policyOID, serialNumber, hashAlgorithm string) error {
	parsed, err := time.Parse(time.RFC3339Nano, genTime)
	if err != nil || !parsed.Equal(metadata.GenTime) || policyOID != metadata.PolicyOID || serialNumber != metadata.SerialNumber || hashAlgorithm != metadata.HashAlgorithm {
		return failure(ReasonMetadata, "stored timestamp metadata does not match the verified response")
	}
	return nil
}

// SafeReason returns the stable reason without exposing parser details.
func SafeReason(err error) Reason {
	if typed, ok := err.(*Error); ok {
		return typed.Reason
	}
	return ReasonMetadata
}

func (m Metadata) String() string {
	return fmt.Sprintf("RFC 3161 timestamp at %s under policy %s", m.GenTime.Format(time.RFC3339Nano), m.PolicyOID)
}
