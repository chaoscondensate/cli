package forecastcrypto

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/chaoscondensate/cli/internal/canonical"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	SealScheme        = "forecast-seal/v1"
	KeyFileSchema     = "forecast-key/v1"
	EncryptionProfile = "chacha20-poly1305"
)

// EntropySource is satisfied by the service CSPRNG. Production callers must
// use the operating-system-backed implementation; tests can inject vectors.
type EntropySource interface {
	ReadFull(context.Context, []byte) error
}

// PrivateBundle is the exact six-field mirror disclosed by a later reveal.
// Public note, supersession, key hint, and ledger metadata never enter it.
type PrivateBundle struct {
	ForecastedAt ledger.Timestamp     `json:"forecasted_at"`
	RecordedAt   ledger.Timestamp     `json:"recorded_at"`
	Value        ledger.ForecastValue `json:"value"`
	Rationale    string               `json:"rationale"`
	KeyFactors   []string             `json:"key_factors"`
	Comment      string               `json:"comment"`
}

type SealResult struct {
	Commitment ledger.SealedCommitment
	KeyFile    []byte
}

type OpenResult struct {
	Bundle PrivateBundle
	KeyHex ledger.Hex32
}

// Seal creates the published forecast-seal/v1 commitment. It returns the
// exact protected key-file bytes separately so callers can durably create the
// key before committing any public ledger mutation.
func Seal(ctx context.Context, questionID, forecastID ledger.Slug, bundle PrivateBundle, keyHint string, entropy EntropySource) (SealResult, error) {
	var result SealResult
	if entropy == nil {
		return result, fmt.Errorf("entropy source is not configured")
	}
	salt := make([]byte, 32)
	key := make([]byte, chacha20poly1305.KeySize)
	nonce := make([]byte, chacha20poly1305.NonceSize)
	defer clear(key)
	for _, destination := range [][]byte{salt, key, nonce} {
		if err := entropy.ReadFull(ctx, destination); err != nil {
			clear(salt)
			clear(nonce)
			return result, fmt.Errorf("read sealing entropy: %w", err)
		}
	}

	plaintext, err := canonicalJSON(struct {
		Schema     string        `json:"schema"`
		QuestionID ledger.Slug   `json:"question_id"`
		ForecastID ledger.Slug   `json:"forecast_id"`
		Salt       string        `json:"salt"`
		Bundle     PrivateBundle `json:"bundle"`
	}{SealScheme, questionID, forecastID, hex.EncodeToString(salt), bundle})
	clear(salt)
	if err != nil {
		return result, fmt.Errorf("canonicalize sealed forecast: %w", err)
	}
	digest := sha256.Sum256(plaintext)
	digestHex := hex.EncodeToString(digest[:])
	aad, err := canonical.Marshal(map[string]any{
		"scheme": SealScheme, "question_id": string(questionID), "forecast_id": string(forecastID),
		"commitment_sha256": digestHex,
	})
	if err != nil {
		return result, fmt.Errorf("canonicalize seal associated data: %w", err)
	}
	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return result, fmt.Errorf("initialize sealing cipher: %w", err)
	}
	ciphertext := cipher.Seal(nil, nonce, plaintext, aad)
	keyFile, err := EncodeKeyFile(questionID, forecastID, key)
	if err != nil {
		return result, err
	}
	result.Commitment = ledger.SealedCommitment{
		Scheme:         SealScheme,
		CommitmentHash: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(digestHex)},
		Encryption: ledger.Encryption{
			Algorithm:  EncryptionProfile,
			Nonce:      ledger.Base64Nonce12(base64.StdEncoding.EncodeToString(nonce)),
			Ciphertext: ledger.Base64Ciphertext(base64.StdEncoding.EncodeToString(ciphertext)),
		},
		KeyHint: keyHint,
	}
	result.KeyFile = keyFile
	return result, nil
}

func EncodeKeyFile(questionID, forecastID ledger.Slug, key []byte) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key must contain exactly 32 bytes")
	}
	encoded, err := canonical.Marshal(map[string]any{
		"schema": KeyFileSchema, "question_id": string(questionID), "forecast_id": string(forecastID),
		"key_hex": hex.EncodeToString(key),
	})
	if err != nil {
		return nil, fmt.Errorf("canonicalize key file: %w", err)
	}
	return append(encoded, '\n'), nil
}

type KeyFile struct {
	Schema     string `json:"schema"`
	QuestionID string `json:"question_id"`
	ForecastID string `json:"forecast_id"`
	KeyHex     string `json:"key_hex"`
}

func DecodeKeyFile(data []byte, questionID, forecastID ledger.Slug) (KeyFile, error) {
	var result KeyFile
	if len(data) == 0 || data[len(data)-1] != '\n' || bytes.Contains(data[:len(data)-1], []byte{'\n'}) {
		return result, fmt.Errorf("key file must end in exactly one LF")
	}
	parsed, err := document.ParseJSON(bytes.NewReader(data[:len(data)-1]), document.Limits{MaxBytes: 4096, MaxDepth: 8, MaxNodes: 16, MaxScalarBytes: 256})
	if err != nil {
		return result, fmt.Errorf("key file is not valid JSON: %w", err)
	}
	canonicalBytes, err := canonical.Marshal(parsed.Root.Any())
	if err != nil || !bytes.Equal(canonicalBytes, data[:len(data)-1]) {
		return result, fmt.Errorf("key file is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(data[:len(data)-1]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return KeyFile{}, fmt.Errorf("key file shape is invalid: %w", err)
	}
	decoded, err := hex.DecodeString(result.KeyHex)
	if result.Schema != KeyFileSchema || result.QuestionID != string(questionID) || result.ForecastID != string(forecastID) || err != nil || len(decoded) != chacha20poly1305.KeySize || result.KeyHex != hex.EncodeToString(decoded) {
		clear(decoded)
		return KeyFile{}, fmt.Errorf("key file binding is invalid")
	}
	clear(decoded)
	return result, nil
}

// Open authenticates every forecast-seal/v1 binding and returns the private
// six-field bundle only after the ciphertext, commitment, protocol, IDs, salt,
// and canonical plaintext have all been checked.
func Open(keyFileBytes []byte, questionID, forecastID ledger.Slug, commitment ledger.SealedCommitment) (OpenResult, error) {
	var result OpenResult
	keyFile, err := DecodeKeyFile(keyFileBytes, questionID, forecastID)
	if err != nil {
		return result, fmt.Errorf("verify key file: %w", err)
	}
	if commitment.Scheme != SealScheme || commitment.CommitmentHash.Algorithm != "sha-256" || commitment.Encryption.Algorithm != EncryptionProfile {
		return result, fmt.Errorf("sealed commitment protocol identifiers are invalid")
	}
	key, err := hex.DecodeString(keyFile.KeyHex)
	if err != nil || len(key) != chacha20poly1305.KeySize {
		clear(key)
		return result, fmt.Errorf("key file contains an invalid key")
	}
	defer clear(key)
	nonce, err := base64.StdEncoding.Strict().DecodeString(string(commitment.Encryption.Nonce))
	if err != nil || len(nonce) != chacha20poly1305.NonceSize {
		clear(nonce)
		return result, fmt.Errorf("sealed commitment nonce is invalid")
	}
	defer clear(nonce)
	ciphertext, err := base64.StdEncoding.Strict().DecodeString(string(commitment.Encryption.Ciphertext))
	if err != nil || len(ciphertext) < chacha20poly1305.Overhead || int64(len(ciphertext)) > document.DefaultLimits.MaxBytes {
		clear(ciphertext)
		return result, fmt.Errorf("sealed commitment ciphertext is invalid")
	}
	defer clear(ciphertext)
	aad, err := canonical.Marshal(map[string]any{
		"scheme": SealScheme, "question_id": string(questionID), "forecast_id": string(forecastID),
		"commitment_sha256": string(commitment.CommitmentHash.Value),
	})
	if err != nil {
		return result, fmt.Errorf("canonicalize seal associated data: %w", err)
	}
	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return result, fmt.Errorf("initialize sealing cipher: %w", err)
	}
	plaintext, err := cipher.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return result, fmt.Errorf("sealed forecast authentication failed")
	}
	defer clear(plaintext)
	digest := sha256.Sum256(plaintext)
	if hex.EncodeToString(digest[:]) != string(commitment.CommitmentHash.Value) {
		return result, fmt.Errorf("sealed forecast commitment digest does not match")
	}
	parsed, err := document.ParseJSON(bytes.NewReader(plaintext), document.DefaultLimits)
	if err != nil {
		return result, fmt.Errorf("sealed plaintext is invalid: %w", err)
	}
	canonicalBytes, err := canonical.Marshal(parsed.Root.Any())
	if err != nil || !bytes.Equal(canonicalBytes, plaintext) {
		return result, fmt.Errorf("sealed plaintext is not canonical")
	}
	var sealed struct {
		Schema     string        `json:"schema"`
		QuestionID ledger.Slug   `json:"question_id"`
		ForecastID ledger.Slug   `json:"forecast_id"`
		Salt       string        `json:"salt"`
		Bundle     PrivateBundle `json:"bundle"`
	}
	decoder := json.NewDecoder(bytes.NewReader(plaintext))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sealed); err != nil {
		return result, fmt.Errorf("sealed plaintext shape is invalid: %w", err)
	}
	reencoded, err := canonicalJSON(sealed)
	if err != nil || !bytes.Equal(reencoded, plaintext) {
		return result, fmt.Errorf("sealed plaintext is not the exact closed forecast-seal/v1 object")
	}
	salt, saltErr := hex.DecodeString(sealed.Salt)
	if sealed.Schema != SealScheme || sealed.QuestionID != questionID || sealed.ForecastID != forecastID || saltErr != nil || len(salt) != 32 || sealed.Salt != hex.EncodeToString(salt) {
		clear(salt)
		return result, fmt.Errorf("sealed plaintext binding is invalid")
	}
	clear(salt)
	result.Bundle = sealed.Bundle
	result.KeyHex = ledger.Hex32(keyFile.KeyHex)
	return result, nil
}

func canonicalJSON(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	parsed, err := document.ParseJSON(bytes.NewReader(encoded), document.DefaultLimits)
	if err != nil {
		return nil, err
	}
	return canonical.Marshal(parsed.Root.Any())
}
