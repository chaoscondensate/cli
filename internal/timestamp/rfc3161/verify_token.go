package rfc3161

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha1" // #nosec G505 -- RFC 2634 ESSCertID uses SHA-1 only as a certificate identifier.
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"time"

	tspclient "github.com/notaryproject/tspclient-go"
)

var (
	oidContentType          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidSigningTime          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
	oidSigningCertificate   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 12}
	oidSigningCertificateV2 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47}
	oidTSTInfo              = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}

	oidRSAEncryption = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidRSAPSS        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	oidSHA256RSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	oidSHA384RSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	oidSHA512RSA     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}
	oidECDSASHA256   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidECDSASHA384   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	oidECDSASHA512   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}
)

// These private structures are the bounded subset of RFC 2634 and RFC 5035
// needed to bind the single CMS signer used by the Forecast Ledger profile.
type signingCertificateV1 struct {
	Certificates []essCertIDV1
	Policies     asn1.RawValue `asn1:"optional"`
}

type essCertIDV1 struct {
	CertHash     []byte
	IssuerSerial essIssuerSerial `asn1:"optional"`
}

type signingCertificateV2 struct {
	Certificates []essCertIDV2
	Policies     asn1.RawValue `asn1:"optional"`
}

type essCertIDV2 struct {
	HashAlgorithm pkix.AlgorithmIdentifier `asn1:"optional"`
	CertHash      []byte
	IssuerSerial  essIssuerSerial `asn1:"optional"`
}

type essIssuerSerial struct {
	IssuerName   essGeneralNames
	SerialNumber *big.Int
}

type essGeneralNames struct {
	Name asn1.RawValue `asn1:"optional,tag:4"`
}

func validateResponseBinding(resp *tspclient.Response, req *tspclient.Request) (*tspclient.SignedToken, *tspclient.TSTInfo, error) {
	if resp == nil || req == nil {
		return nil, nil, failure(ReasonResponseMalformed, "timestamp response or request is missing")
	}
	token, err := resp.SignedToken()
	if err != nil {
		return nil, nil, failure(ReasonResponseRejected, "timestamp response status or signed token is invalid")
	}
	info, err := token.Info()
	if err != nil || info.Version != 1 || info.GenTime.IsZero() || info.GenTime.Location() != time.UTC {
		return nil, nil, failure(ReasonTokenMalformed, "timestamp token metadata is malformed")
	}
	if req.ReqPolicy != nil && !req.ReqPolicy.Equal(info.Policy) {
		return nil, nil, failure(ReasonPolicyMismatch, "timestamp response policy does not match the request")
	}
	if !info.MessageImprint.Equal(req.MessageImprint) {
		return nil, nil, failure(ReasonImprintMismatch, "timestamp response message imprint does not match the request")
	}
	if req.Nonce != nil && (info.Nonce == nil || info.Nonce.Cmp(req.Nonce) != 0) {
		return nil, nil, failure(ReasonNonceMismatch, "timestamp response nonce does not match the request")
	}
	if req.CertReq && len(token.Certificates) == 0 {
		return nil, nil, failure(ReasonCertificate, "timestamp response omitted the requested signer certificate")
	}
	if !req.CertReq && len(token.Certificates) != 0 {
		return nil, nil, failure(ReasonResponseMalformed, "timestamp response included unrequested certificates")
	}
	return token, info, nil
}

func verifySignedToken(ctx context.Context, token *tspclient.SignedToken, opts x509.VerifyOptions, limits Limits) ([]*x509.Certificate, error) {
	if token == nil || len(token.SignerInfos) != 1 || len(token.Certificates) == 0 {
		return nil, failure(ReasonTokenMalformed, "timestamp token must contain exactly one signer and its certificate")
	}
	if err := ctx.Err(); err != nil {
		return nil, failure(ReasonSignature, "timestamp signature verification was interrupted")
	}
	signerInfo := token.SignerInfos[0]
	if signerInfo.Version != 1 || signerInfo.SignerIdentifier.SerialNumber == nil {
		return nil, failure(ReasonTokenMalformed, "timestamp signer identifier is unsupported")
	}
	if len(signerInfo.SignedAttributes) == 0 || len(signerInfo.SignedAttributes) > limits.Extensions {
		return nil, failure(ReasonLimit, "timestamp signed-attribute count is outside limits")
	}

	var signer *x509.Certificate
	for _, candidate := range token.Certificates {
		if candidate != nil && candidate.SerialNumber.Cmp(signerInfo.SignerIdentifier.SerialNumber) == 0 && bytes.Equal(candidate.RawIssuer, signerInfo.SignerIdentifier.Issuer.FullBytes) {
			if signer != nil {
				return nil, failure(ReasonSignerBinding, "timestamp signer identifier is ambiguous")
			}
			signer = candidate
		}
	}
	if signer == nil {
		return nil, failure(ReasonSignerBinding, "timestamp signer certificate is not present")
	}

	var contentTypeValues, messageDigestValues, signingTimeValues, v1Values, v2Values [][]byte
	for _, attribute := range signerInfo.SignedAttributes {
		switch {
		case attribute.Type.Equal(oidContentType):
			contentTypeValues = append(contentTypeValues, attribute.Values.Bytes)
		case attribute.Type.Equal(oidMessageDigest):
			messageDigestValues = append(messageDigestValues, attribute.Values.Bytes)
		case attribute.Type.Equal(oidSigningTime):
			signingTimeValues = append(signingTimeValues, attribute.Values.Bytes)
		case attribute.Type.Equal(oidSigningCertificate):
			v1Values = append(v1Values, attribute.Values.Bytes)
		case attribute.Type.Equal(oidSigningCertificateV2):
			v2Values = append(v2Values, attribute.Values.Bytes)
		}
	}
	if len(contentTypeValues) != 1 || len(messageDigestValues) != 1 {
		return nil, failure(ReasonTokenMalformed, "timestamp token has missing or duplicate CMS content attributes")
	}
	if len(signingTimeValues) > 1 {
		return nil, failure(ReasonTokenMalformed, "timestamp token has duplicate CMS signing-time attributes")
	}
	if len(v1Values) > 1 || len(v2Values) > 1 || len(v1Values)+len(v2Values) == 0 {
		return nil, failure(ReasonSignerBinding, "timestamp token has missing or duplicate ESS signer attributes")
	}

	var contentType asn1.ObjectIdentifier
	if err := unmarshalSingleAttribute(contentTypeValues[0], &contentType); err != nil || !contentType.Equal(oidTSTInfo) || !token.ContentType.Equal(contentType) {
		return nil, failure(ReasonTokenMalformed, "timestamp CMS content type is malformed or inconsistent")
	}
	var expectedDigest []byte
	if err := unmarshalSingleAttribute(messageDigestValues[0], &expectedDigest); err != nil {
		return nil, failure(ReasonTokenMalformed, "timestamp CMS message digest is malformed")
	}
	var signingTime *time.Time
	if len(signingTimeValues) == 1 {
		var value time.Time
		if err := unmarshalSingleAttribute(signingTimeValues[0], &value); err != nil {
			return nil, failure(ReasonTokenMalformed, "timestamp CMS signing time is malformed")
		}
		signingTime = &value
	}
	digest, signatureAlgorithm, err := strongSignerAlgorithms(signerInfo.DigestAlgorithm.Algorithm, signerInfo.SignatureAlgorithm.Algorithm)
	if err != nil {
		return nil, err
	}
	actualDigest := digestBytes(digest, token.Content)
	if !bytes.Equal(expectedDigest, actualDigest) {
		return nil, failure(ReasonSignature, "timestamp CMS message digest does not match its content")
	}

	if len(v1Values) == 1 {
		var binding signingCertificateV1
		if err := unmarshalSingleAttribute(v1Values[0], &binding); err != nil || len(binding.Certificates) == 0 || len(binding.Certificates) > limits.Certificates {
			return nil, failure(ReasonSignerBinding, "timestamp ESS v1 signer binding is malformed or ambiguous")
		}
		for index, identifier := range binding.Certificates {
			certificate := selectESSCertificate(identifier.IssuerSerial, token.Certificates, signer, index)
			if certificate == nil {
				return nil, failure(ReasonSignerBinding, "timestamp ESS v1 certificate identifier is missing or ambiguous")
			}
			hash := sha1.Sum(certificate.Raw) // #nosec G401 -- RFC 2634 mandates this certificate identifier.
			if !bytes.Equal(identifier.CertHash, hash[:]) || index == 0 && !certificate.Equal(signer) {
				return nil, failure(ReasonSignerBinding, "timestamp ESS v1 signer binding does not match the CMS signer")
			}
		}
	}
	if len(v2Values) == 1 {
		var binding signingCertificateV2
		if err := unmarshalSingleAttribute(v2Values[0], &binding); err != nil || len(binding.Certificates) == 0 || len(binding.Certificates) > limits.Certificates {
			return nil, failure(ReasonSignerBinding, "timestamp ESS v2 signer binding is malformed or ambiguous")
		}
		for index, identifier := range binding.Certificates {
			certificate := selectESSCertificate(identifier.IssuerSerial, token.Certificates, signer, index)
			if certificate == nil {
				return nil, failure(ReasonSignerBinding, "timestamp ESS v2 certificate identifier is missing or ambiguous")
			}
			hashOID := identifier.HashAlgorithm.Algorithm
			if len(hashOID) == 0 {
				hashOID = oidSHA256
			}
			certDigest, ok := strongDigest(hashOID, certificate.Raw)
			if !ok {
				return nil, failure(ReasonAlgorithm, "timestamp ESS v2 certificate digest algorithm is unsupported")
			}
			if !bytes.Equal(identifier.CertHash, certDigest) || index == 0 && !certificate.Equal(signer) {
				return nil, failure(ReasonSignerBinding, "timestamp ESS v2 signer binding does not match the CMS signer")
			}
		}
	}

	signedAttributes, marshalErr := asn1.MarshalWithParams(signerInfo.SignedAttributes, "set")
	if marshalErr != nil {
		return nil, failure(ReasonTokenMalformed, "timestamp signed attributes cannot be encoded canonically")
	}
	if checkErr := signer.CheckSignature(signatureAlgorithm, signedAttributes, signerInfo.Signature); checkErr != nil {
		return nil, failure(ReasonSignature, "timestamp CMS signature is invalid")
	}

	intermediates := x509.NewCertPool()
	for _, certificate := range token.Certificates {
		if certificate != nil && !certificate.Equal(signer) {
			intermediates.AddCert(certificate)
		}
	}
	opts.Intermediates = intermediates
	chains, verifyErr := signer.Verify(opts)
	if verifyErr != nil || len(chains) == 0 {
		return nil, failure(ReasonTrustBundle, "timestamp signer does not chain to the retained trust bundle")
	}
	if signingTime != nil {
		for _, certificate := range chains[0] {
			if signingTime.Before(certificate.NotBefore) || signingTime.After(certificate.NotAfter) {
				return nil, failure(ReasonCertificate, "timestamp CMS signing time is outside the retained certificate chain validity")
			}
		}
	}
	return chains[0], nil
}

func unmarshalSingleAttribute(data []byte, out any) error {
	rest, err := asn1.Unmarshal(data, out)
	if err != nil || len(rest) != 0 {
		return asn1.SyntaxError{Msg: "attribute contains malformed or trailing data"}
	}
	return nil
}

func issuerSerialMatches(identifier essIssuerSerial, signer *x509.Certificate) bool {
	if identifier.SerialNumber == nil {
		return identifier.IssuerName.Name.FullBytes == nil && identifier.IssuerName.Name.Bytes == nil
	}
	if signer == nil || identifier.SerialNumber.Cmp(signer.SerialNumber) != 0 || len(identifier.IssuerName.Name.Bytes) == 0 {
		return false
	}
	var issuer asn1.RawValue
	rest, err := asn1.Unmarshal(identifier.IssuerName.Name.Bytes, &issuer)
	return err == nil && len(rest) == 0 && bytes.Equal(issuer.FullBytes, signer.RawIssuer)
}

func selectESSCertificate(identifier essIssuerSerial, certificates []*x509.Certificate, signer *x509.Certificate, index int) *x509.Certificate {
	if identifier.SerialNumber == nil {
		if index == 0 && issuerSerialMatches(identifier, signer) {
			return signer
		}
		return nil
	}
	var selected *x509.Certificate
	for _, candidate := range certificates {
		if candidate == nil || !issuerSerialMatches(identifier, candidate) {
			continue
		}
		if selected != nil {
			return nil
		}
		selected = candidate
	}
	return selected
}

func strongSignerAlgorithms(digestOID, signatureOID asn1.ObjectIdentifier) (crypto.Hash, x509.SignatureAlgorithm, error) {
	var digest crypto.Hash
	switch {
	case digestOID.Equal(oidSHA256):
		digest = crypto.SHA256
	case digestOID.Equal(oidSHA384):
		digest = crypto.SHA384
	case digestOID.Equal(oidSHA512):
		digest = crypto.SHA512
	default:
		return 0, x509.UnknownSignatureAlgorithm, failure(ReasonAlgorithm, "timestamp token signer uses an unsupported digest algorithm")
	}

	algorithm := x509.UnknownSignatureAlgorithm
	switch {
	case signatureOID.Equal(oidRSAEncryption):
		algorithm = mapRSAAlgorithm(digest, false)
	case signatureOID.Equal(oidRSAPSS):
		algorithm = mapRSAAlgorithm(digest, true)
	case signatureOID.Equal(oidSHA256RSA) && digest == crypto.SHA256:
		algorithm = x509.SHA256WithRSA
	case signatureOID.Equal(oidSHA384RSA) && digest == crypto.SHA384:
		algorithm = x509.SHA384WithRSA
	case signatureOID.Equal(oidSHA512RSA) && digest == crypto.SHA512:
		algorithm = x509.SHA512WithRSA
	case signatureOID.Equal(oidECDSASHA256) && digest == crypto.SHA256:
		algorithm = x509.ECDSAWithSHA256
	case signatureOID.Equal(oidECDSASHA384) && digest == crypto.SHA384:
		algorithm = x509.ECDSAWithSHA384
	case signatureOID.Equal(oidECDSASHA512) && digest == crypto.SHA512:
		algorithm = x509.ECDSAWithSHA512
	}
	if algorithm == x509.UnknownSignatureAlgorithm {
		return 0, algorithm, failure(ReasonAlgorithm, "timestamp token signature and digest algorithms are unsupported or inconsistent")
	}
	return digest, algorithm, nil
}

func mapRSAAlgorithm(digest crypto.Hash, pss bool) x509.SignatureAlgorithm {
	if pss {
		switch digest {
		case crypto.SHA256:
			return x509.SHA256WithRSAPSS
		case crypto.SHA384:
			return x509.SHA384WithRSAPSS
		case crypto.SHA512:
			return x509.SHA512WithRSAPSS
		}
	}
	switch digest {
	case crypto.SHA256:
		return x509.SHA256WithRSA
	case crypto.SHA384:
		return x509.SHA384WithRSA
	case crypto.SHA512:
		return x509.SHA512WithRSA
	default:
		return x509.UnknownSignatureAlgorithm
	}
}

func digestBytes(hash crypto.Hash, data []byte) []byte {
	switch hash {
	case crypto.SHA256:
		sum := sha256.Sum256(data)
		return sum[:]
	case crypto.SHA384:
		sum := sha512.Sum384(data)
		return sum[:]
	case crypto.SHA512:
		sum := sha512.Sum512(data)
		return sum[:]
	default:
		return nil
	}
}

func strongDigest(oid asn1.ObjectIdentifier, data []byte) ([]byte, bool) {
	switch {
	case oid.Equal(oidSHA256):
		sum := sha256.Sum256(data)
		return sum[:], true
	case oid.Equal(oidSHA384):
		sum := sha512.Sum384(data)
		return sum[:], true
	case oid.Equal(oidSHA512):
		sum := sha512.Sum512(data)
		return sum[:], true
	default:
		return nil, false
	}
}
