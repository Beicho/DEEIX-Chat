package security

import (
	"context"
	"errors"
	"sync"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

type memoryProofStore struct {
	mu         sync.Mutex
	challenges map[string]StoredPoWChallenge
	used       map[string]time.Time
	nonces     map[string]time.Time
	keys       map[string]domainsecurity.BrowserKey
}

func newMemoryProofStore() *memoryProofStore {
	return &memoryProofStore{
		challenges: make(map[string]StoredPoWChallenge),
		used:       make(map[string]time.Time),
		nonces:     make(map[string]time.Time),
		keys:       make(map[string]domainsecurity.BrowserKey),
	}
}

func (s *memoryProofStore) StorePoWChallenge(_ context.Context, item StoredPoWChallenge, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[item.Challenge] = item
	return nil
}

func (s *memoryProofStore) GetPoWChallenge(_ context.Context, challenge string) (*StoredPoWChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.challenges[challenge]
	if !ok {
		return nil, ErrPoWChallengeNotFound
	}
	return &item, nil
}

func (s *memoryProofStore) UsePoWChallenge(_ context.Context, challenge string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.used[challenge]; ok {
		return false, nil
	}
	s.used[challenge] = time.Now().Add(ttl)
	return true, nil
}

func (s *memoryProofStore) UseRequestNonce(_ context.Context, sessionID string, nonce string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := sessionID + ":" + nonce
	if _, ok := s.nonces[key]; ok {
		return false, nil
	}
	s.nonces[key] = time.Now().Add(ttl)
	return true, nil
}

func (s *memoryProofStore) StoreBrowserKey(_ context.Context, key domainsecurity.BrowserKey, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[browserKeyMapKey(key.UserID, key.SessionID, key.KeyID)] = key
	return nil
}

func (s *memoryProofStore) GetBrowserKey(_ context.Context, userID uint, sessionID string, keyID string) (*domainsecurity.BrowserKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.keys[browserKeyMapKey(userID, sessionID, keyID)]
	if !ok {
		return nil, ErrBrowserKeyNotFound
	}
	if item.RevokedAt != nil {
		return nil, ErrBrowserKeyRevoked
	}
	return &item, nil
}

func browserKeyMapKey(userID uint, sessionID string, keyID string) string {
	return string(rune(userID)) + ":" + sessionID + ":" + keyID
}

var errMemoryStoreNotImplemented = errors.New("memory proof store method not implemented")
