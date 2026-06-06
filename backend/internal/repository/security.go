package repository

import (
	"context"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

// FingerprintRepository defines persistence for device fingerprints and links.
type FingerprintRepository interface {
	UpsertDeviceFingerprint(ctx context.Context, item *domainsecurity.DeviceFingerprint) error
	FindByFingerprintID(ctx context.Context, fingerprintID string) ([]domainsecurity.DeviceFingerprint, error)
	UpsertAssociation(ctx context.Context, item *domainsecurity.FingerprintAssociation) error
	ListAssociations(ctx context.Context, offset int, limit int) ([]domainsecurity.FingerprintAssociation, int64, error)
	FindAssociationsByUserID(ctx context.Context, userID uint) ([]domainsecurity.FingerprintAssociation, error)
}
