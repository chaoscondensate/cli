package forecastcrypto

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/chaoscondensate/forecast-ledger/internal/canonical"
	"github.com/chaoscondensate/forecast-ledger/internal/document"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	"golang.org/x/crypto/chacha20poly1305"
)

var ErrRevealVerification = errors.New("revealed forecast verification failed")

type RevealedPayload struct {
	Schema     string
	QuestionID string
	ForecastID string
	Bundle     map[string]any
	Plaintext  []byte
}

// Reveal authenticates, decrypts, binds, hashes, parses, and canonical-checks
// a revealed forecast. Errors intentionally do not include secret material.
func Reveal(questionID, forecastID ledger.Slug, commitment ledger.RevealedCommitment) (*RevealedPayload, error) {
	key, err := hex.DecodeString(string(commitment.RevealedKey))
	if err != nil || len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("%w: invalid revealed key", ErrRevealVerification)
	}
	nonce, err := base64.StdEncoding.Strict().DecodeString(string(commitment.Encryption.Nonce))
	if err != nil || len(nonce) != chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("%w: invalid nonce", ErrRevealVerification)
	}
	ciphertext, err := base64.StdEncoding.Strict().DecodeString(string(commitment.Encryption.Ciphertext))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext", ErrRevealVerification)
	}
	aad, err := canonical.Marshal(map[string]any{
		"scheme":            "forecast-seal/v1",
		"question_id":       string(questionID),
		"forecast_id":       string(forecastID),
		"commitment_sha256": string(commitment.CommitmentHash.Value),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: invalid associated data", ErrRevealVerification)
	}
	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("%w: initialize cipher", ErrRevealVerification)
	}
	plaintext, err := cipher.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrRevealVerification)
	}
	digest := sha256.Sum256(plaintext)
	expected, err := hex.DecodeString(string(commitment.CommitmentHash.Value))
	if err != nil || len(expected) != sha256.Size || subtle.ConstantTimeCompare(digest[:], expected) != 1 {
		return nil, fmt.Errorf("%w: commitment digest mismatch", ErrRevealVerification)
	}

	parsed, err := document.ParseJSON(bytes.NewReader(plaintext), document.DefaultLimits)
	if err != nil {
		return nil, fmt.Errorf("%w: plaintext is not valid profile JSON", ErrRevealVerification)
	}
	value, ok := parsed.Root.Any().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: plaintext root is not an object", ErrRevealVerification)
	}
	canonicalBytes, err := canonical.Marshal(value)
	if err != nil || !bytes.Equal(canonicalBytes, plaintext) {
		return nil, fmt.Errorf("%w: plaintext is not canonical", ErrRevealVerification)
	}
	schemaName, _ := value["schema"].(string)
	payloadQuestionID, _ := value["question_id"].(string)
	payloadForecastID, _ := value["forecast_id"].(string)
	bundle, bundleOK := value["bundle"].(map[string]any)
	if schemaName != "forecast-seal/v1" || payloadQuestionID != string(questionID) || payloadForecastID != string(forecastID) || !bundleOK {
		return nil, fmt.Errorf("%w: payload binding mismatch", ErrRevealVerification)
	}
	if strings.TrimSpace(commitment.Scheme) != "forecast-seal/v1" {
		return nil, fmt.Errorf("%w: unsupported scheme", ErrRevealVerification)
	}
	return &RevealedPayload{
		Schema:     schemaName,
		QuestionID: payloadQuestionID,
		ForecastID: payloadForecastID,
		Bundle:     bundle,
		Plaintext:  bytes.Clone(plaintext),
	}, nil
}
