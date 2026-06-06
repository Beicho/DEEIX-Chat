package security

import (
	"context"
	"sort"
	"sync"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

type memoryFingerprintRepo struct {
	mu           sync.Mutex
	fingerprints map[string]domainsecurity.DeviceFingerprint
	associations map[string]domainsecurity.FingerprintAssociation
	nextID       uint
}

func newMemoryFingerprintRepo() *memoryFingerprintRepo {
	return &memoryFingerprintRepo{
		fingerprints: make(map[string]domainsecurity.DeviceFingerprint),
		associations: make(map[string]domainsecurity.FingerprintAssociation),
		nextID:       1,
	}
}

func (r *memoryFingerprintRepo) UpsertDeviceFingerprint(_ context.Context, item *domainsecurity.DeviceFingerprint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return nil
	}
	key := fingerprintMapKey(item.UserID, item.FingerprintID)
	now := time.Now().UTC()
	existing, ok := r.fingerprints[key]
	if ok {
		item.ID = existing.ID
		item.FirstSeenAt = existing.FirstSeenAt
		item.SeenCount = existing.SeenCount + 1
	} else {
		item.ID = r.nextID
		r.nextID++
		item.FirstSeenAt = now
		item.SeenCount = 1
	}
	item.LastSeenAt = now
	r.fingerprints[key] = *item
	return nil
}

func (r *memoryFingerprintRepo) FindByFingerprintID(_ context.Context, fingerprintID string) ([]domainsecurity.DeviceFingerprint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domainsecurity.DeviceFingerprint, 0)
	for _, item := range r.fingerprints {
		if item.FingerprintID == fingerprintID {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].UserID < result[j].UserID
	})
	return result, nil
}

func (r *memoryFingerprintRepo) UpsertAssociation(_ context.Context, item *domainsecurity.FingerprintAssociation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item == nil {
		return nil
	}
	existing, ok := r.associations[item.FingerprintID]
	if ok {
		item.ID = existing.ID
	} else {
		item.ID = r.nextID
		r.nextID++
	}
	item.DetectedAt = time.Now().UTC()
	r.associations[item.FingerprintID] = *item
	return nil
}

func (r *memoryFingerprintRepo) ListAssociations(_ context.Context, offset int, limit int) ([]domainsecurity.FingerprintAssociation, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domainsecurity.FingerprintAssociation, 0, len(r.associations))
	for _, item := range r.associations {
		items = append(items, item)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].DetectedAt.After(items[j].DetectedAt)
	})
	total := int64(len(items))
	if offset >= len(items) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(items) || limit <= 0 {
		end = len(items)
	}
	return items[offset:end], total, nil
}

func (r *memoryFingerprintRepo) FindAssociationsByUserID(_ context.Context, userID uint) ([]domainsecurity.FingerprintAssociation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domainsecurity.FingerprintAssociation, 0)
	for _, item := range r.associations {
		for _, associatedUserID := range item.UserIDs {
			if associatedUserID == userID {
				result = append(result, item)
				break
			}
		}
	}
	return result, nil
}

func fingerprintMapKey(userID uint, fingerprintID string) string {
	return string(rune(userID)) + ":" + fingerprintID
}
