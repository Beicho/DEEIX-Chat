package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

type RequestProof = domainsecurity.RequestProof

type PoWVerifier interface {
	VerifyProof(ctx context.Context, userID uint, action string, proof *PoWProof) error
}

type RequestProofServiceOptions struct {
	Store                ProofStore
	TimestampSkew        time.Duration
	RequestNonceTTL      time.Duration
	KeyTTL               time.Duration
	PoWVerifier          PoWVerifier
	RequirePoW           bool
	ProtectedProofPrefix string
}

type RequestProofService struct {
	store       ProofStore
	skew        time.Duration
	nonceTTL    time.Duration
	keyTTL      time.Duration
	powVerifier PoWVerifier
	requirePoW  bool
	prefix      string
}

type RequestProofVerifyInput struct {
	UserID    uint
	SessionID string
	Method    string
	Path      string
	Query     string
	Action    string
	Body      []byte
	Proof     *RequestProof
	Now       time.Time
}

func NewRequestProofService(options RequestProofServiceOptions) *RequestProofService {
	skew := options.TimestampSkew
	if skew <= 0 {
		skew = time.Minute
	}
	nonceTTL := options.RequestNonceTTL
	if nonceTTL <= 0 {
		nonceTTL = 2 * time.Minute
	}
	keyTTL := options.KeyTTL
	if keyTTL <= 0 {
		keyTTL = 7 * 24 * time.Hour
	}
	prefix := strings.TrimSpace(options.ProtectedProofPrefix)
	if prefix == "" {
		prefix = "DEEIX-PROOF-v1"
	}
	return &RequestProofService{
		store:       options.Store,
		skew:        skew,
		nonceTTL:    nonceTTL,
		keyTTL:      keyTTL,
		powVerifier: options.PoWVerifier,
		requirePoW:  options.RequirePoW,
		prefix:      prefix,
	}
}

func (s *RequestProofService) BootstrapBrowserKey(ctx context.Context, userID uint, sessionID string, keyID string, publicKeyJWK string) error {
	if s == nil || s.store == nil {
		return ErrPoWStoreUnavailable
	}
	if _, err := parseP256PublicJWK(publicKeyJWK); err != nil {
		return ErrBrowserKeyInvalid
	}
	if strings.TrimSpace(keyID) == "" {
		keyID = BrowserKeyID(publicKeyJWK)
	}
	now := time.Now().UTC()
	return s.store.StoreBrowserKey(ctx, domainsecurity.BrowserKey{
		UserID:       userID,
		SessionID:    strings.TrimSpace(sessionID),
		KeyID:        strings.TrimSpace(keyID),
		PublicKeyJWK: strings.TrimSpace(publicKeyJWK),
		CreatedAt:    now,
		LastUsedAt:   now,
	}, s.keyTTL)
}

func BrowserKeyID(publicKeyJWK string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(publicKeyJWK)))
	return hex.EncodeToString(sum[:16])
}

func (s *RequestProofService) VerifyRequestProof(ctx context.Context, input RequestProofVerifyInput) error {
	if s == nil || s.store == nil {
		return ErrPoWStoreUnavailable
	}
	proof := input.Proof
	if proof == nil {
		return ErrRequestProofMissing
	}
	if strings.TrimSpace(proof.SessionID) == "" || strings.TrimSpace(proof.SessionID) != strings.TrimSpace(input.SessionID) {
		return ErrRequestProofSession
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	requestTime := time.UnixMilli(proof.Timestamp).UTC()
	if requestTime.Before(now.Add(-s.skew)) || requestTime.After(now.Add(s.skew)) {
		return ErrRequestProofTimestamp
	}
	actualBodyHash := base64.RawURLEncoding.EncodeToString(sha256Bytes(input.Body))
	if len(input.Body) == 0 && strings.TrimSpace(proof.BodyHash) == "" {
		actualBodyHash = ""
	}
	if strings.TrimSpace(proof.BodyHash) != actualBodyHash {
		return ErrRequestProofBodyHashMismatch
	}
	key, err := s.store.GetBrowserKey(ctx, input.UserID, proof.SessionID, proof.KeyID)
	if err != nil {
		return err
	}
	publicKey, err := parseP256PublicJWK(key.PublicKeyJWK)
	if err != nil {
		return ErrBrowserKeyInvalid
	}
	canonical := BuildCanonicalRequestProof(s.prefix, input.Method, input.Path, input.Query, proof.Timestamp, proof.Nonce, proof.BodyHash, proof.SessionID, proof.KeyID)
	if !verifyP1363Signature(publicKey, []byte(canonical), proof.Signature) {
		return ErrRequestProofSignature
	}
	if s.requirePoW || proof.PoWProof != nil {
		if s.powVerifier == nil || proof.PoWProof == nil {
			return ErrPoWInvalidProof
		}
		if err = s.powVerifier.VerifyProof(ctx, input.UserID, input.Action, proof.PoWProof); err != nil {
			return err
		}
	}
	used, err := s.store.UseRequestNonce(ctx, proof.SessionID, proof.Nonce, s.nonceTTL)
	if err != nil {
		return err
	}
	if !used {
		return ErrRequestProofNonceReplay
	}
	return nil
}

func BuildCanonicalRequestProof(prefix string, method string, path string, query string, timestamp int64, nonce string, bodyHash string, sessionID string, keyID string) string {
	return strings.Join([]string{
		strings.TrimSpace(prefix),
		strings.ToUpper(strings.TrimSpace(method)),
		strings.TrimSpace(path),
		strings.TrimPrefix(strings.TrimSpace(query), "?"),
		strconv.FormatInt(timestamp, 10),
		strings.TrimSpace(nonce),
		strings.TrimSpace(bodyHash),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(keyID),
	}, "\n")
}

type p256JWK struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func parseP256PublicJWK(payload string) (*ecdsa.PublicKey, error) {
	var jwk p256JWK
	if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &jwk); err != nil {
		return nil, err
	}
	if jwk.KTY != "EC" || jwk.CRV != "P-256" || jwk.X == "" || jwk.Y == "" {
		return nil, ErrBrowserKeyInvalid
	}
	xRaw, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, err
	}
	yRaw, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, err
	}
	x := new(big.Int).SetBytes(xRaw)
	y := new(big.Int).SetBytes(yRaw)
	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, ErrBrowserKeyInvalid
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func verifyP1363Signature(publicKey *ecdsa.PublicKey, payload []byte, signatureB64 string) bool {
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(signatureB64))
	if err != nil || len(signature) != 64 {
		return false
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	digest := sha256.Sum256(payload)
	return ecdsa.Verify(publicKey, digest[:], r, s)
}

func sha256Bytes(body []byte) []byte {
	sum := sha256.Sum256(body)
	return sum[:]
}
