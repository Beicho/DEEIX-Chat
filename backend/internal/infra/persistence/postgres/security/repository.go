package security

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repo implements security-related Postgres persistence.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates security repository.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	return err
}

// UpsertDeviceFingerprint inserts or updates a user fingerprint observation.
func (r *Repo) UpsertDeviceFingerprint(ctx context.Context, item *domainsecurity.DeviceFingerprint) error {
	if r == nil || r.db == nil || item == nil {
		return nil
	}
	now := time.Now().UTC()
	if item.FirstSeenAt.IsZero() {
		item.FirstSeenAt = now
	}
	item.LastSeenAt = now
	if item.SeenCount <= 0 {
		item.SeenCount = 1
	}
	row := toModelDeviceFingerprint(item)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "fingerprint_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"screen_resolution":    keepExistingWhenExcludedStringEmpty("screen_resolution"),
			"color_depth":          keepExistingWhenExcludedIntZero("color_depth"),
			"pixel_ratio":          keepExistingWhenExcludedFloatZero("pixel_ratio"),
			"hardware_concurrency": keepExistingWhenExcludedIntZero("hardware_concurrency"),
			"device_memory":        keepExistingWhenExcludedIntZero("device_memory"),
			"max_touch_points":     keepExistingWhenExcludedIntZero("max_touch_points"),
			"user_agent":           keepExistingWhenExcludedStringEmpty("user_agent"),
			"language":             keepExistingWhenExcludedStringEmpty("language"),
			"timezone":             keepExistingWhenExcludedStringEmpty("timezone"),
			"platform":             keepExistingWhenExcludedStringEmpty("platform"),
			"canvas_hash":          keepExistingWhenExcludedStringEmpty("canvas_hash"),
			"web_gl_vendor":        keepExistingWhenExcludedStringEmpty("web_gl_vendor"),
			"web_gl_renderer":      keepExistingWhenExcludedStringEmpty("web_gl_renderer"),
			"fonts_hash":           keepExistingWhenExcludedStringEmpty("fonts_hash"),
			"audio_hash":           keepExistingWhenExcludedStringEmpty("audio_hash"),
			"ip_address":           keepExistingWhenExcludedStringEmpty("ip_address"),
			"tls_fingerprint":      keepExistingWhenExcludedStringEmpty("tls_fingerprint"),
			"last_seen_at":         row.LastSeenAt,
			"seen_count":           gorm.Expr("device_fingerprints.seen_count + 1"),
			"updated_at":           now,
		}),
	}).Create(row).Error
	return translateError(err)
}

// FindByFingerprintID lists all user observations for a fingerprint.
func (r *Repo) FindByFingerprintID(ctx context.Context, fingerprintID string) ([]domainsecurity.DeviceFingerprint, error) {
	rows := make([]model.DeviceFingerprint, 0)
	err := r.db.WithContext(ctx).
		Where("fingerprint_id = ?", fingerprintID).
		Order("user_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, translateError(err)
	}
	return toDomainDeviceFingerprints(rows), nil
}

// UpsertAssociation stores the latest association for a fingerprint.
func (r *Repo) UpsertAssociation(ctx context.Context, item *domainsecurity.FingerprintAssociation) error {
	if r == nil || r.db == nil || item == nil {
		return nil
	}
	sort.Slice(item.UserIDs, func(i int, j int) bool {
		return item.UserIDs[i] < item.UserIDs[j]
	})
	userIDsJSON, err := json.Marshal(item.UserIDs)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if item.DetectedAt.IsZero() {
		item.DetectedAt = now
	}
	row := &model.FingerprintAssociation{
		FingerprintID:   item.FingerprintID,
		UserIDsJSON:     string(userIDsJSON),
		ConfidenceScore: item.ConfidenceScore,
		RiskLevel:       item.RiskLevel,
		DetectedAt:      item.DetectedAt,
		IgnoredAt:       item.IgnoredAt,
		Reason:          item.Reason,
	}
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fingerprint_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"user_ids_json":    row.UserIDsJSON,
			"confidence_score": row.ConfidenceScore,
			"risk_level":       row.RiskLevel,
			"detected_at":      row.DetectedAt,
			"ignored_at":       row.IgnoredAt,
			"reason":           row.Reason,
			"updated_at":       now,
		}),
	}).Create(row).Error
	return translateError(err)
}

// ListAssociations returns the newest detections first.
func (r *Repo) ListAssociations(ctx context.Context, offset int, limit int) ([]domainsecurity.FingerprintAssociation, int64, error) {
	rows := make([]model.FingerprintAssociation, 0)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.FingerprintAssociation{}).Where("ignored_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.Order("detected_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, translateError(err)
	}
	return toDomainAssociations(rows), total, nil
}

// FindAssociationsByUserID returns active associations containing userID.
func (r *Repo) FindAssociationsByUserID(ctx context.Context, userID uint) ([]domainsecurity.FingerprintAssociation, error) {
	rows := make([]model.FingerprintAssociation, 0)
	if err := r.db.WithContext(ctx).
		Where("ignored_at IS NULL").
		Where("user_ids_json LIKE ?", "%"+jsonUserIDFragment(userID)+"%").
		Find(&rows).Error; err != nil {
		return nil, translateError(err)
	}
	result := make([]domainsecurity.FingerprintAssociation, 0, len(rows))
	for _, item := range toDomainAssociations(rows) {
		for _, associatedUserID := range item.UserIDs {
			if associatedUserID == userID {
				result = append(result, item)
				break
			}
		}
	}
	return result, nil
}

func toModelDeviceFingerprint(item *domainsecurity.DeviceFingerprint) *model.DeviceFingerprint {
	if item == nil {
		return &model.DeviceFingerprint{}
	}
	return &model.DeviceFingerprint{
		BaseModel: model.BaseModel{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		},
		UserID:              item.UserID,
		FingerprintID:       item.FingerprintID,
		ScreenResolution:    item.ScreenResolution,
		ColorDepth:          item.ColorDepth,
		PixelRatio:          item.PixelRatio,
		HardwareConcurrency: item.HardwareConcurrency,
		DeviceMemory:        item.DeviceMemory,
		MaxTouchPoints:      item.MaxTouchPoints,
		UserAgent:           item.UserAgent,
		Language:            item.Language,
		Timezone:            item.Timezone,
		Platform:            item.Platform,
		CanvasHash:          item.CanvasHash,
		WebGLVendor:         item.WebGLVendor,
		WebGLRenderer:       item.WebGLRenderer,
		FontsHash:           item.FontsHash,
		AudioHash:           item.AudioHash,
		IPAddress:           item.IPAddress,
		TLSFingerprint:      item.TLSFingerprint,
		FirstSeenAt:         item.FirstSeenAt,
		LastSeenAt:          item.LastSeenAt,
		SeenCount:           item.SeenCount,
	}
}

func toDomainDeviceFingerprints(rows []model.DeviceFingerprint) []domainsecurity.DeviceFingerprint {
	result := make([]domainsecurity.DeviceFingerprint, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainsecurity.DeviceFingerprint{
			ID:                  row.ID,
			UserID:              row.UserID,
			FingerprintID:       row.FingerprintID,
			ScreenResolution:    row.ScreenResolution,
			ColorDepth:          row.ColorDepth,
			PixelRatio:          row.PixelRatio,
			HardwareConcurrency: row.HardwareConcurrency,
			DeviceMemory:        row.DeviceMemory,
			MaxTouchPoints:      row.MaxTouchPoints,
			UserAgent:           row.UserAgent,
			Language:            row.Language,
			Timezone:            row.Timezone,
			Platform:            row.Platform,
			CanvasHash:          row.CanvasHash,
			WebGLVendor:         row.WebGLVendor,
			WebGLRenderer:       row.WebGLRenderer,
			FontsHash:           row.FontsHash,
			AudioHash:           row.AudioHash,
			IPAddress:           row.IPAddress,
			TLSFingerprint:      row.TLSFingerprint,
			FirstSeenAt:         row.FirstSeenAt,
			LastSeenAt:          row.LastSeenAt,
			SeenCount:           row.SeenCount,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
		})
	}
	return result
}

func toDomainAssociations(rows []model.FingerprintAssociation) []domainsecurity.FingerprintAssociation {
	result := make([]domainsecurity.FingerprintAssociation, 0, len(rows))
	for _, row := range rows {
		var userIDs []uint
		_ = json.Unmarshal([]byte(row.UserIDsJSON), &userIDs)
		result = append(result, domainsecurity.FingerprintAssociation{
			ID:              row.ID,
			FingerprintID:   row.FingerprintID,
			UserIDs:         userIDs,
			ConfidenceScore: row.ConfidenceScore,
			RiskLevel:       row.RiskLevel,
			DetectedAt:      row.DetectedAt,
			IgnoredAt:       row.IgnoredAt,
			Reason:          row.Reason,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return result
}

func jsonUserIDFragment(userID uint) string {
	payload, _ := json.Marshal(userID)
	return string(payload)
}

func keepExistingWhenExcludedStringEmpty(column string) clause.Expr {
	return gorm.Expr(`CASE WHEN EXCLUDED.` + column + ` <> '' THEN EXCLUDED.` + column + ` ELSE device_fingerprints.` + column + ` END`)
}

func keepExistingWhenExcludedIntZero(column string) clause.Expr {
	return gorm.Expr(`CASE WHEN EXCLUDED.` + column + ` <> 0 THEN EXCLUDED.` + column + ` ELSE device_fingerprints.` + column + ` END`)
}

func keepExistingWhenExcludedFloatZero(column string) clause.Expr {
	return gorm.Expr(`CASE WHEN EXCLUDED.` + column + ` <> 0 THEN EXCLUDED.` + column + ` ELSE device_fingerprints.` + column + ` END`)
}
