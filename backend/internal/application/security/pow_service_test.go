package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPoWServiceVerifiesProofAndRejectsReplay(t *testing.T) {
	store := newMemoryProofStore()
	service := NewPoWService(PoWServiceOptions{
		Store:               store,
		BaseDifficulty:      map[string]int{"send_message": 2},
		DefaultDifficulty:   2,
		MaxDifficulty:       8,
		ChallengeTTL:        time.Minute,
		UsedChallengeTTL:    2 * time.Minute,
		ChallengeByteLength: 16,
	})

	challenge, err := service.GenerateChallenge(context.Background(), 42, "send_message")
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	if challenge.Action != "send_message" {
		t.Fatalf("expected action send_message, got %q", challenge.Action)
	}
	if challenge.Difficulty != 2 {
		t.Fatalf("expected difficulty 2, got %d", challenge.Difficulty)
	}

	nonce, hash := solveTestPoW(challenge.Challenge, challenge.Difficulty)
	proof := &PoWProof{
		Challenge:  challenge.Challenge,
		Nonce:      nonce,
		Hash:       hash,
		Difficulty: challenge.Difficulty,
	}

	if err = service.VerifyProof(context.Background(), 42, "send_message", proof); err != nil {
		t.Fatalf("verify proof: %v", err)
	}
	if err = service.VerifyProof(context.Background(), 42, "send_message", proof); !errors.Is(err, ErrPoWChallengeReplay) {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestPoWServiceRejectsWrongHash(t *testing.T) {
	store := newMemoryProofStore()
	service := NewPoWService(PoWServiceOptions{
		Store:             store,
		DefaultDifficulty: 2,
		MaxDifficulty:     8,
		ChallengeTTL:      time.Minute,
		UsedChallengeTTL:  2 * time.Minute,
	})

	challenge, err := service.GenerateChallenge(context.Background(), 42, "send_message")
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	proof := &PoWProof{
		Challenge:  challenge.Challenge,
		Nonce:      "1",
		Hash:       strings.Repeat("0", 64),
		Difficulty: challenge.Difficulty,
	}

	if err = service.VerifyProof(context.Background(), 42, "send_message", proof); !errors.Is(err, ErrPoWInvalidProof) {
		t.Fatalf("expected invalid proof error, got %v", err)
	}
}

func solveTestPoW(challenge string, difficulty int) (string, string) {
	target := strings.Repeat("0", difficulty)
	for nonce := int64(0); ; nonce++ {
		nonceText := strconv.FormatInt(nonce, 10)
		sum := sha256.Sum256([]byte(challenge + nonceText))
		hash := hex.EncodeToString(sum[:])
		if strings.HasPrefix(hash, target) {
			return nonceText, hash
		}
	}
}
