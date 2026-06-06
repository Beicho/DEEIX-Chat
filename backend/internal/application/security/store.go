package security

import (
	"context"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

type StoredPoWChallenge struct {
	Challenge  string
	UserID     uint
	Action     string
	Difficulty int
	ExpiresAt  time.Time
}

// ProofStore hides Redis/Postgres details from proof verification logic.
type ProofStore interface {
	StorePoWChallenge(ctx context.Context, item StoredPoWChallenge, ttl time.Duration) error
	GetPoWChallenge(ctx context.Context, challenge string) (*StoredPoWChallenge, error)
	UsePoWChallenge(ctx context.Context, challenge string, ttl time.Duration) (bool, error)
	UseRequestNonce(ctx context.Context, sessionID string, nonce string, ttl time.Duration) (bool, error)
	StoreBrowserKey(ctx context.Context, key domainsecurity.BrowserKey, ttl time.Duration) error
	GetBrowserKey(ctx context.Context, userID uint, sessionID string, keyID string) (*domainsecurity.BrowserKey, error)
}
