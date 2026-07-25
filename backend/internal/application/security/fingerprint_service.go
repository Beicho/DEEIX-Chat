package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// FingerprintService records browser fingerprints and flags shared devices.
type FingerprintService struct {
	repo      repository.FingerprintRepository
	alertHook MultiAccountAlertHook
}

// NewFingerprintService creates fingerprint risk service.
func NewFingerprintService(repo repository.FingerprintRepository) *FingerprintService {
	return &FingerprintService{repo: repo}
}

// MultiAccountAlertHook 在检测到高危多账号关联时被调用。
type MultiAccountAlertHook func(ctx context.Context, detection domainsecurity.MultiAccountDetection)

// SetMultiAccountAlertHook 注入多账号关联告警回调。
func (s *FingerprintService) SetMultiAccountAlertHook(hook MultiAccountAlertHook) {
	if s == nil {
		return
	}
	s.alertHook = hook
}

// notifyMultiAccount 仅在高危关联首次落库时触发告警。
func (s *FingerprintService) notifyMultiAccount(ctx context.Context, detection *domainsecurity.MultiAccountDetection) {
	if s == nil || s.alertHook == nil || detection == nil {
		return
	}
	if detection.RiskLevel != "high" {
		return
	}
	s.alertHook(ctx, *detection)
}

// RecordFingerprint saves a fingerprint and returns a detection when it is shared.
func (s *FingerprintService) RecordFingerprint(ctx context.Context, input domainsecurity.DeviceFingerprint) (*domainsecurity.MultiAccountDetection, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	input.FingerprintID = StableFingerprintID(input)
	if input.FingerprintID == "" || input.UserID == 0 {
		return nil, nil
	}
	if err := s.repo.UpsertDeviceFingerprint(ctx, &input); err != nil {
		return nil, err
	}
	detection, err := s.DetectMultiAccount(ctx, input.FingerprintID)
	if err != nil || detection == nil {
		return detection, err
	}
	if detection.RiskLevel == "medium" || detection.RiskLevel == "high" {
		association := &domainsecurity.FingerprintAssociation{
			FingerprintID:   detection.FingerprintID,
			UserIDs:         detection.AssociatedUsers,
			ConfidenceScore: detection.ConfidenceScore,
			RiskLevel:       detection.RiskLevel,
		}
		if err = s.repo.UpsertAssociation(ctx, association); err != nil {
			return nil, err
		}
		s.notifyMultiAccount(ctx, detection)
	}
	return detection, nil
}

// TouchFingerprint records a lightweight observation from protected requests.
func (s *FingerprintService) TouchFingerprint(ctx context.Context, userID uint, fingerprintID string, ipAddress string, userAgent string) {
	if s == nil || s.repo == nil || userID == 0 {
		return
	}
	fp := domainsecurity.DeviceFingerprint{
		UserID:        userID,
		FingerprintID: strings.TrimSpace(fingerprintID),
		IPAddress:     strings.TrimSpace(ipAddress),
		UserAgent:     strings.TrimSpace(userAgent),
	}
	if fp.FingerprintID == "" {
		return
	}
	_ = s.repo.UpsertDeviceFingerprint(ctx, &fp)
	detection, err := s.DetectMultiAccount(ctx, fp.FingerprintID)
	if err != nil || detection == nil {
		return
	}
	if detection.RiskLevel != "medium" && detection.RiskLevel != "high" {
		return
	}
	_ = s.repo.UpsertAssociation(ctx, &domainsecurity.FingerprintAssociation{
		FingerprintID:   detection.FingerprintID,
		UserIDs:         detection.AssociatedUsers,
		ConfidenceScore: detection.ConfidenceScore,
		RiskLevel:       detection.RiskLevel,
	})
	s.notifyMultiAccount(ctx, detection)
}

// DetectMultiAccount evaluates whether a fingerprint is shared by multiple users.
func (s *FingerprintService) DetectMultiAccount(ctx context.Context, fingerprintID string) (*domainsecurity.MultiAccountDetection, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	rows, err := s.repo.FindByFingerprintID(ctx, strings.TrimSpace(fingerprintID))
	if err != nil {
		return nil, err
	}
	userIDs := uniqueUserIDs(rows)
	if len(userIDs) <= 1 {
		return nil, nil
	}
	confidence := calculateFingerprintConfidence(rows)
	riskLevel := riskLevelForConfidence(confidence)
	return &domainsecurity.MultiAccountDetection{
		FingerprintID:   strings.TrimSpace(fingerprintID),
		AssociatedUsers: userIDs,
		ConfidenceScore: confidence,
		RiskLevel:       riskLevel,
	}, nil
}

// ListAssociations returns persisted multi-account detections.
func (s *FingerprintService) ListAssociations(ctx context.Context, page int, pageSize int) ([]domainsecurity.FingerprintAssociation, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, nil
	}
	offset, limit := normalizeFingerprintPage(page, pageSize)
	return s.repo.ListAssociations(ctx, offset, limit)
}

// GetUserRiskLevel returns the strongest risk level associated with a user.
func (s *FingerprintService) GetUserRiskLevel(ctx context.Context, userID uint) (string, error) {
	if s == nil || s.repo == nil || userID == 0 {
		return "low", nil
	}
	associations, err := s.repo.FindAssociationsByUserID(ctx, userID)
	if err != nil {
		return "low", err
	}
	level := "low"
	for _, association := range associations {
		switch association.RiskLevel {
		case "high":
			return "high", nil
		case "medium":
			level = "medium"
		}
	}
	return level, nil
}

// StableFingerprintID computes the server-side stable ID from durable signals.
func StableFingerprintID(fp domainsecurity.DeviceFingerprint) string {
	stable := []string{
		strings.TrimSpace(fp.ScreenResolution),
		intString(fp.ColorDepth),
		floatString(fp.PixelRatio),
		intString(fp.HardwareConcurrency),
		intString(fp.DeviceMemory),
		intString(fp.MaxTouchPoints),
		strings.TrimSpace(fp.Language),
		strings.TrimSpace(fp.Timezone),
		strings.TrimSpace(fp.Platform),
		strings.TrimSpace(fp.CanvasHash),
		strings.TrimSpace(fp.WebGLVendor),
		strings.TrimSpace(fp.WebGLRenderer),
		strings.TrimSpace(fp.FontsHash),
		strings.TrimSpace(fp.AudioHash),
	}
	allEmpty := true
	for _, value := range stable {
		if value != "" && value != "0" && value != "0.00" {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		return strings.TrimSpace(fp.FingerprintID)
	}
	sum := sha256.Sum256([]byte(strings.Join(stable, "|")))
	return hex.EncodeToString(sum[:])
}

func uniqueUserIDs(rows []domainsecurity.DeviceFingerprint) []uint {
	seen := make(map[uint]struct{}, len(rows))
	result := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.UserID == 0 {
			continue
		}
		if _, ok := seen[row.UserID]; ok {
			continue
		}
		seen[row.UserID] = struct{}{}
		result = append(result, row.UserID)
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i] < result[j]
	})
	return result
}

func calculateFingerprintConfidence(rows []domainsecurity.DeviceFingerprint) float64 {
	if len(rows) <= 1 {
		return 0
	}
	scores := []float64{
		sameValueScore(rows, func(fp domainsecurity.DeviceFingerprint) string {
			return strings.Join([]string{
				fp.ScreenResolution,
				intString(fp.ColorDepth),
				floatString(fp.PixelRatio),
				intString(fp.HardwareConcurrency),
				intString(fp.DeviceMemory),
				intString(fp.MaxTouchPoints),
			}, "|")
		}),
		sameValueScore(rows, func(fp domainsecurity.DeviceFingerprint) string {
			return strings.Join([]string{fp.CanvasHash, fp.WebGLVendor, fp.WebGLRenderer, fp.FontsHash, fp.AudioHash}, "|")
		}),
		sameValueScore(rows, func(fp domainsecurity.DeviceFingerprint) string {
			return strings.Join([]string{fp.Language, fp.Timezone, fp.Platform}, "|")
		}),
		sameValueScore(rows, func(fp domainsecurity.DeviceFingerprint) string {
			return fp.IPAddress
		}),
	}
	weights := []float64{0.35, 0.35, 0.20, 0.10}
	total := 0.0
	for i, score := range scores {
		total += score * weights[i]
	}
	if total > 1 {
		return 1
	}
	return total
}

func sameValueScore(rows []domainsecurity.DeviceFingerprint, valueFn func(domainsecurity.DeviceFingerprint) string) float64 {
	first := ""
	seen := false
	for _, row := range rows {
		value := strings.TrimSpace(valueFn(row))
		if value == "" || strings.Trim(value, "|0.") == "" {
			return 0
		}
		if !seen {
			first = value
			seen = true
			continue
		}
		if value != first {
			return 0
		}
	}
	if seen {
		return 1
	}
	return 0
}

func riskLevelForConfidence(confidence float64) string {
	switch {
	case confidence >= 0.8:
		return "high"
	case confidence >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

func normalizeFingerprintPage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return (page - 1) * pageSize, pageSize
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func floatString(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}
