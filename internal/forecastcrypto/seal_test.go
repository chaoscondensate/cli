package forecastcrypto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"reflect"
	"testing"

	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

type vectorEntropy struct{ reader io.Reader }

func (source vectorEntropy) ReadFull(_ context.Context, destination []byte) error {
	_, err := io.ReadFull(source.reader, destination)
	return err
}

func TestSealMatchesPinnedUpstreamVector(t *testing.T) {
	data, err := fs.ReadFile(contractschema.Conformance(), "forecast-seal-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		QuestionID string        `json:"question_id"`
		ForecastID string        `json:"forecast_id"`
		Bundle     PrivateBundle `json:"bundle"`
		Material   struct {
			SaltHex  string `json:"salt_hex"`
			KeyHex   string `json:"key_hex"`
			NonceHex string `json:"nonce_hex"`
		} `json:"material"`
		Expected struct {
			CanonicalPlaintext string `json:"canonical_plaintext"`
			Commitment         struct {
				Scheme         string            `json:"scheme"`
				CommitmentHash ledger.Digest     `json:"commitment_hash"`
				Encryption     ledger.Encryption `json:"encryption"`
				KeyHint        string            `json:"key_hint"`
			} `json:"commitment"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	material := append(decodeHex(t, vector.Material.SaltHex), decodeHex(t, vector.Material.KeyHex)...)
	material = append(material, decodeHex(t, vector.Material.NonceHex)...)
	sealed, err := Seal(context.Background(), ledger.Slug(vector.QuestionID), ledger.Slug(vector.ForecastID), vector.Bundle, vector.Expected.Commitment.KeyHint, vectorEntropy{reader: bytes.NewReader(material)})
	if err != nil {
		t.Fatal(err)
	}
	wantCommitment := ledger.SealedCommitment{
		Scheme: vector.Expected.Commitment.Scheme, CommitmentHash: vector.Expected.Commitment.CommitmentHash,
		Encryption: vector.Expected.Commitment.Encryption, KeyHint: vector.Expected.Commitment.KeyHint,
	}
	if !reflect.DeepEqual(sealed.Commitment, wantCommitment) {
		t.Fatalf("commitment = %#v, want %#v", sealed.Commitment, wantCommitment)
	}
	wantKeyFile := "{\"forecast_id\":\"" + vector.ForecastID + "\",\"key_hex\":\"" + vector.Material.KeyHex + "\",\"question_id\":\"" + vector.QuestionID + "\",\"schema\":\"forecast-key/v1\"}\n"
	if string(sealed.KeyFile) != wantKeyFile {
		t.Fatalf("key file = %q, want %q", sealed.KeyFile, wantKeyFile)
	}
	revealed := ledger.RevealedCommitment{
		Scheme: sealed.Commitment.Scheme, CommitmentHash: sealed.Commitment.CommitmentHash,
		Encryption: sealed.Commitment.Encryption, KeyHint: sealed.Commitment.KeyHint,
		RevealedKey: ledger.Hex32(vector.Material.KeyHex),
	}
	payload, err := Reveal(ledger.Slug(vector.QuestionID), ledger.Slug(vector.ForecastID), revealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload.Plaintext) != vector.Expected.CanonicalPlaintext {
		t.Fatalf("plaintext = %s", payload.Plaintext)
	}
	if _, err := DecodeKeyFile(sealed.KeyFile, ledger.Slug(vector.QuestionID), ledger.Slug(vector.ForecastID)); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(sealed.KeyFile, ledger.Slug(vector.QuestionID), ledger.Slug(vector.ForecastID), sealed.Commitment)
	if err != nil || !reflect.DeepEqual(opened.Bundle, vector.Bundle) || string(opened.KeyHex) != vector.Material.KeyHex {
		t.Fatalf("opened = %#v, error = %v", opened, err)
	}
}

func TestDecodeKeyFileRejectsNonCanonicalAndWrongBinding(t *testing.T) {
	key := bytes.Repeat([]byte{0xab}, 32)
	data, err := EncodeKeyFile("q-one", "f-one", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeKeyFile(data, "q-two", "f-one"); err == nil {
		t.Fatal("wrong question binding accepted")
	}
	nonCanonical := []byte("{\"schema\":\"forecast-key/v1\",\"question_id\":\"q-one\",\"forecast_id\":\"f-one\",\"key_hex\":\"" + hex.EncodeToString(key) + "\"}\n")
	if _, err := DecodeKeyFile(nonCanonical, "q-one", "f-one"); err == nil {
		t.Fatal("non-canonical key file accepted")
	}
}

func TestOpenRejectsWrongKeyAndTamperedCiphertext(t *testing.T) {
	bundle := PrivateBundle{ForecastedAt: "2026-01-01T00:00:00Z", RecordedAt: "2026-01-01T00:01:00Z", Value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}}, Rationale: "private", KeyFactors: []string{}, Comment: "private"}
	sealed, err := Seal(context.Background(), "q-one", "f-one", bundle, "forecast-key:f-one", vectorEntropy{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 76))})
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := EncodeKeyFile("q-one", "f-one", bytes.Repeat([]byte{0x24}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(wrong, "q-one", "f-one", sealed.Commitment); err == nil {
		t.Fatal("wrong key authenticated")
	}
	tampered := sealed.Commitment
	ciphertext, err := base64.StdEncoding.DecodeString(string(tampered.Encryption.Ciphertext))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 1
	tampered.Encryption.Ciphertext = ledger.Base64Ciphertext(base64.StdEncoding.EncodeToString(ciphertext))
	if _, err := Open(sealed.KeyFile, "q-one", "f-one", tampered); err == nil {
		t.Fatal("tampered ciphertext authenticated")
	}
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func FuzzDecodeKeyFile(f *testing.F) {
	valid, err := EncodeKeyFile("q-one", "f-one", bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte(`{"schema":"forecast-key/v1"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeKeyFile(data, "q-one", "f-one")
	})
}

func FuzzOpenSealedForecast(f *testing.F) {
	bundle := PrivateBundle{ForecastedAt: "2026-01-01T00:00:00Z", RecordedAt: "2026-01-01T00:01:00Z", Value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}}, Rationale: "private", KeyFactors: []string{}, Comment: "private"}
	sealed, err := Seal(context.Background(), "q-one", "f-one", bundle, "forecast-key:f-one", vectorEntropy{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 76))})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(sealed.KeyFile, string(sealed.Commitment.Encryption.Ciphertext))
	f.Add([]byte("not-a-key"), "not-base64")
	f.Fuzz(func(t *testing.T, keyFile []byte, ciphertext string) {
		commitment := sealed.Commitment
		commitment.Encryption.Ciphertext = ledger.Base64Ciphertext(ciphertext)
		_, _ = Open(keyFile, "q-one", "f-one", commitment)
	})
}
