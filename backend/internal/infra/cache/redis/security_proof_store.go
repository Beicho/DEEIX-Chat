package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
	"github.com/go-redis/redis/v8"
)

type securityProofStore struct {
	client *redis.Client
}

func NewSecurityProofStore(client *redis.Client) *securityProofStore {
	if client == nil {
		return nil
	}
	return &securityProofStore{client: client}
}

func (s *securityProofStore) StorePoWChallenge(ctx context.Context, item domainsecurity.StoredPoWChallenge, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return domainsecurity.ErrPoWStoreUnavailable
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, powChallengeKey(item.Challenge), payload, ttl).Err()
}

func (s *securityProofStore) GetPoWChallenge(ctx context.Context, challenge string) (*domainsecurity.StoredPoWChallenge, error) {
	if s == nil || s.client == nil {
		return nil, domainsecurity.ErrPoWStoreUnavailable
	}
	payload, err := s.client.Get(ctx, powChallengeKey(challenge)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domainsecurity.ErrPoWChallengeNotFound
		}
		return nil, err
	}
	var item domainsecurity.StoredPoWChallenge
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *securityProofStore) UsePoWChallenge(ctx context.Context, challenge string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, domainsecurity.ErrPoWStoreUnavailable
	}
	return s.client.SetNX(ctx, powUsedKey(challenge), "1", ttl).Result()
}

func (s *securityProofStore) UseRequestNonce(ctx context.Context, sessionID string, nonce string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return false, domainsecurity.ErrPoWStoreUnavailable
	}
	return s.client.SetNX(ctx, requestNonceKey(sessionID, nonce), "1", ttl).Result()
}

func (s *securityProofStore) StoreBrowserKey(ctx context.Context, key domainsecurity.BrowserKey, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return domainsecurity.ErrPoWStoreUnavailable
	}
	payload, err := json.Marshal(key)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, browserKeyKey(key.UserID, key.SessionID, key.KeyID), payload, ttl).Err()
}

func (s *securityProofStore) GetBrowserKey(ctx context.Context, userID uint, sessionID string, keyID string) (*domainsecurity.BrowserKey, error) {
	if s == nil || s.client == nil {
		return nil, domainsecurity.ErrPoWStoreUnavailable
	}
	payload, err := s.client.Get(ctx, browserKeyKey(userID, sessionID, keyID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domainsecurity.ErrBrowserKeyNotFound
		}
		return nil, err
	}
	var item domainsecurity.BrowserKey
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	if item.RevokedAt != nil {
		return nil, domainsecurity.ErrBrowserKeyRevoked
	}
	return &item, nil
}

func powChallengeKey(challenge string) string {
	return "deeix:pow:challenge:" + strings.TrimSpace(challenge)
}

func powUsedKey(challenge string) string {
	return "deeix:pow:used:" + strings.TrimSpace(challenge)
}

func requestNonceKey(sessionID string, nonce string) string {
	return fmt.Sprintf("deeix:proof:nonce:%s:%s", strings.TrimSpace(sessionID), strings.TrimSpace(nonce))
}

func browserKeyKey(userID uint, sessionID string, keyID string) string {
	return fmt.Sprintf("deeix:proof:key:%d:%s:%s", userID, strings.TrimSpace(sessionID), strings.TrimSpace(keyID))
}
