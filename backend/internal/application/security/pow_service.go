package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

type PoWChallenge = domainsecurity.PoWChallenge
type PoWProof = domainsecurity.PoWProof

// PoWServiceOptions configures proof-of-work challenge generation.
type PoWServiceOptions struct {
	Store               ProofStore
	BaseDifficulty      map[string]int
	DefaultDifficulty   int
	MaxDifficulty       int
	ChallengeTTL        time.Duration
	UsedChallengeTTL    time.Duration
	ChallengeByteLength int
	RiskResolver        UserRiskResolver
}

// UserRiskResolver lets PoW difficulty respond to fingerprint risk.
type UserRiskResolver interface {
	GetUserRiskLevel(ctx context.Context, userID uint) (string, error)
}

// PoWService issues and verifies short-lived SHA-256 leading-zero challenges.
type PoWService struct {
	store               ProofStore
	baseDifficulty      map[string]int
	defaultDifficulty   int
	maxDifficulty       int
	challengeTTL        time.Duration
	usedChallengeTTL    time.Duration
	challengeByteLength int
	riskResolver        UserRiskResolver
}

func NewPoWService(options PoWServiceOptions) *PoWService {
	defaultDifficulty := options.DefaultDifficulty
	if defaultDifficulty <= 0 {
		defaultDifficulty = 5
	}
	maxDifficulty := options.MaxDifficulty
	if maxDifficulty <= 0 {
		maxDifficulty = 10
	}
	challengeTTL := options.ChallengeTTL
	if challengeTTL <= 0 {
		challengeTTL = time.Minute
	}
	usedChallengeTTL := options.UsedChallengeTTL
	if usedChallengeTTL <= 0 {
		usedChallengeTTL = 2 * time.Minute
	}
	challengeByteLength := options.ChallengeByteLength
	if challengeByteLength <= 0 {
		challengeByteLength = 32
	}
	return &PoWService{
		store:               options.Store,
		baseDifficulty:      options.BaseDifficulty,
		defaultDifficulty:   defaultDifficulty,
		maxDifficulty:       maxDifficulty,
		challengeTTL:        challengeTTL,
		usedChallengeTTL:    usedChallengeTTL,
		challengeByteLength: challengeByteLength,
		riskResolver:        options.RiskResolver,
	}
}

func (s *PoWService) GenerateChallenge(ctx context.Context, userID uint, action string) (*PoWChallenge, error) {
	if s == nil || s.store == nil {
		return nil, ErrPoWStoreUnavailable
	}
	difficulty := s.dynamicDifficulty(ctx, userID, action)
	randomBytes := make([]byte, s.challengeByteLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate pow challenge: %w", err)
	}
	now := time.Now().UTC()
	challenge := hex.EncodeToString(randomBytes)
	item := StoredPoWChallenge{
		Challenge:  challenge,
		UserID:     userID,
		Action:     strings.TrimSpace(action),
		Difficulty: difficulty,
		ExpiresAt:  now.Add(s.challengeTTL),
	}
	if err := s.store.StorePoWChallenge(ctx, item, s.challengeTTL); err != nil {
		return nil, err
	}
	return &PoWChallenge{
		Challenge:  item.Challenge,
		Difficulty: item.Difficulty,
		Action:     item.Action,
		ExpiresAt:  item.ExpiresAt,
	}, nil
}

func (s *PoWService) VerifyProof(ctx context.Context, userID uint, action string, proof *PoWProof) error {
	if s == nil || s.store == nil {
		return ErrPoWStoreUnavailable
	}
	if proof == nil || strings.TrimSpace(proof.Challenge) == "" || strings.TrimSpace(proof.Nonce) == "" || strings.TrimSpace(proof.Hash) == "" {
		return ErrPoWInvalidProof
	}
	item, err := s.store.GetPoWChallenge(ctx, strings.TrimSpace(proof.Challenge))
	if err != nil {
		return err
	}
	if item.UserID != userID || strings.TrimSpace(item.Action) != strings.TrimSpace(action) {
		return ErrPoWActionMismatch
	}
	if time.Now().UTC().After(item.ExpiresAt) {
		return ErrPoWChallengeExpired
	}
	if proof.Difficulty != item.Difficulty || proof.Difficulty <= 0 || proof.Difficulty > s.maxDifficulty {
		return ErrPoWInvalidDifficulty
	}
	expected := hashPoW(strings.TrimSpace(proof.Challenge), strings.TrimSpace(proof.Nonce))
	if !strings.EqualFold(expected, strings.TrimSpace(proof.Hash)) {
		return ErrPoWInvalidProof
	}
	if !strings.HasPrefix(expected, strings.Repeat("0", proof.Difficulty)) {
		return ErrPoWInvalidProof
	}
	used, err := s.store.UsePoWChallenge(ctx, item.Challenge, s.usedChallengeTTL)
	if err != nil {
		return err
	}
	if !used {
		return ErrPoWChallengeReplay
	}
	return nil
}

func (s *PoWService) GetDynamicDifficulty(_ uint, action string) int {
	difficulty := s.defaultDifficulty
	if s != nil && s.baseDifficulty != nil {
		if value := s.baseDifficulty[strings.TrimSpace(action)]; value > 0 {
			difficulty = value
		}
	}
	if s != nil && s.maxDifficulty > 0 && difficulty > s.maxDifficulty {
		return s.maxDifficulty
	}
	if difficulty <= 0 {
		return 1
	}
	return difficulty
}

func (s *PoWService) dynamicDifficulty(ctx context.Context, userID uint, action string) int {
	difficulty := s.GetDynamicDifficulty(userID, action)
	if s == nil || s.riskResolver == nil || userID == 0 {
		return difficulty
	}
	level, err := s.riskResolver.GetUserRiskLevel(ctx, userID)
	if err != nil {
		return difficulty
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high":
		difficulty += 2
	case "medium":
		difficulty++
	}
	if s.maxDifficulty > 0 && difficulty > s.maxDifficulty {
		return s.maxDifficulty
	}
	return difficulty
}

func hashPoW(challenge string, nonce string) string {
	sum := sha256.Sum256([]byte(challenge + nonce))
	return hex.EncodeToString(sum[:])
}

func SolvePoWForTest(challenge string, difficulty int) (string, string) {
	target := strings.Repeat("0", difficulty)
	for nonce := int64(0); ; nonce++ {
		nonceText := strconv.FormatInt(nonce, 10)
		hash := hashPoW(challenge, nonceText)
		if strings.HasPrefix(hash, target) {
			return nonceText, hash
		}
	}
}
