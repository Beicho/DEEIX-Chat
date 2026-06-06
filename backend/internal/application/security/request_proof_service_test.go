package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestRequestProofServiceVerifiesWebCryptoStyleSignature(t *testing.T) {
	store := newMemoryProofStore()
	service := NewRequestProofService(RequestProofServiceOptions{
		Store:                store,
		TimestampSkew:        time.Minute,
		RequestNonceTTL:      2 * time.Minute,
		PoWVerifier:          nil,
		RequirePoW:           false,
		ProtectedProofPrefix: "DEEIX-PROOF-v1",
	})
	privateKey, publicJWK := generateTestP256Key(t)
	now := time.UnixMilli(1700000000000)
	body := []byte(`{"content":"hello"}`)
	bodyHash := base64.RawURLEncoding.EncodeToString(sha256BytesTest(body))

	if err := service.BootstrapBrowserKey(context.Background(), 7, "session-1", "key-1", publicJWK); err != nil {
		t.Fatalf("bootstrap browser key: %v", err)
	}

	proof := &PoWRequestProof{
		RequestProof: RequestProof{
			KeyID:     "key-1",
			SessionID: "session-1",
			Timestamp: now.UnixMilli(),
			Nonce:     "nonce-1",
			BodyHash:  bodyHash,
		},
		Method: "POST",
		Path:   "/api/v1/conversations/abc/messages/stream",
		Query:  "",
	}
	canonical := BuildCanonicalRequestProof("DEEIX-PROOF-v1", proof.Method, proof.Path, proof.Query, proof.Timestamp, proof.Nonce, proof.BodyHash, proof.SessionID, proof.KeyID)
	proof.Signature = signP1363(t, privateKey, []byte(canonical))

	err := service.VerifyRequestProof(context.Background(), RequestProofVerifyInput{
		UserID:    7,
		SessionID: "session-1",
		Method:    proof.Method,
		Path:      proof.Path,
		Query:     proof.Query,
		Body:      body,
		Proof:     &proof.RequestProof,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("verify request proof: %v", err)
	}
}

func TestRequestProofServiceRejectsTamperedBodyAndReplayNonce(t *testing.T) {
	store := newMemoryProofStore()
	service := NewRequestProofService(RequestProofServiceOptions{
		Store:                store,
		TimestampSkew:        time.Minute,
		RequestNonceTTL:      2 * time.Minute,
		ProtectedProofPrefix: "DEEIX-PROOF-v1",
	})
	privateKey, publicJWK := generateTestP256Key(t)
	now := time.UnixMilli(1700000000000)
	body := []byte(`{"content":"hello"}`)
	bodyHash := base64.RawURLEncoding.EncodeToString(sha256BytesTest(body))

	if err := service.BootstrapBrowserKey(context.Background(), 7, "session-1", "key-1", publicJWK); err != nil {
		t.Fatalf("bootstrap browser key: %v", err)
	}

	proof := RequestProof{
		KeyID:     "key-1",
		SessionID: "session-1",
		Timestamp: now.UnixMilli(),
		Nonce:     "nonce-1",
		BodyHash:  bodyHash,
	}
	canonical := BuildCanonicalRequestProof("DEEIX-PROOF-v1", "POST", "/api/v1/conversations/abc/messages", "", proof.Timestamp, proof.Nonce, proof.BodyHash, proof.SessionID, proof.KeyID)
	proof.Signature = signP1363(t, privateKey, []byte(canonical))

	tamperedErr := service.VerifyRequestProof(context.Background(), RequestProofVerifyInput{
		UserID:    7,
		SessionID: "session-1",
		Method:    "POST",
		Path:      "/api/v1/conversations/abc/messages",
		Body:      []byte(`{"content":"tampered"}`),
		Proof:     &proof,
		Now:       now,
	})
	if !errors.Is(tamperedErr, ErrRequestProofBodyHashMismatch) {
		t.Fatalf("expected body hash mismatch, got %v", tamperedErr)
	}

	if err := service.VerifyRequestProof(context.Background(), RequestProofVerifyInput{
		UserID:    7,
		SessionID: "session-1",
		Method:    "POST",
		Path:      "/api/v1/conversations/abc/messages",
		Body:      body,
		Proof:     &proof,
		Now:       now,
	}); err != nil {
		t.Fatalf("first verify request proof: %v", err)
	}
	if err := service.VerifyRequestProof(context.Background(), RequestProofVerifyInput{
		UserID:    7,
		SessionID: "session-1",
		Method:    "POST",
		Path:      "/api/v1/conversations/abc/messages",
		Body:      body,
		Proof:     &proof,
		Now:       now,
	}); !errors.Is(err, ErrRequestProofNonceReplay) {
		t.Fatalf("expected nonce replay, got %v", err)
	}
}

type PoWRequestProof struct {
	RequestProof
	Method string
	Path   string
	Query  string
}

func generateTestP256Key(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	jwk := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes()),
	}
	payload, err := json.Marshal(jwk)
	if err != nil {
		t.Fatalf("marshal jwk: %v", err)
	}
	return privateKey, string(payload)
}

func signP1363(t *testing.T, privateKey *ecdsa.PrivateKey, payload []byte) string {
	t.Helper()
	digest := sha256.Sum256(payload)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	signature := append(padBigInt(r, 32), padBigInt(s, 32)...)
	return base64.RawURLEncoding.EncodeToString(signature)
}

func padBigInt(value *big.Int, size int) []byte {
	raw := value.Bytes()
	if len(raw) >= size {
		return raw[len(raw)-size:]
	}
	padded := make([]byte, size)
	copy(padded[size-len(raw):], raw)
	return padded
}

func sha256BytesTest(body []byte) []byte {
	sum := sha256.Sum256(body)
	return sum[:]
}

func assertNoNewline(t *testing.T, value string) {
	t.Helper()
	if strings.Contains(value, "\r") {
		t.Fatal("unexpected CR in test data")
	}
}
