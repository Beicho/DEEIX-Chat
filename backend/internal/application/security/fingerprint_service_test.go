package security

import (
	"context"
	"testing"

	domainsecurity "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/security"
)

func TestFingerprintServiceRecordsSharedFingerprintAssociation(t *testing.T) {
	repo := newMemoryFingerprintRepo()
	service := NewFingerprintService(repo)

	first := domainsecurity.DeviceFingerprint{
		UserID:              10,
		ScreenResolution:    "1920x1080",
		ColorDepth:          24,
		PixelRatio:          2,
		HardwareConcurrency: 8,
		DeviceMemory:        8,
		Language:            "zh-CN",
		Timezone:            "Asia/Shanghai",
		Platform:            "MacIntel",
		CanvasHash:          "canvas-a",
		WebGLVendor:         "Apple",
		WebGLRenderer:       "Apple M4",
		FontsHash:           "fonts-a",
		AudioHash:           "audio-a",
	}
	second := first
	second.UserID = 20

	if _, err := service.RecordFingerprint(context.Background(), first); err != nil {
		t.Fatalf("record first fingerprint: %v", err)
	}
	detection, err := service.RecordFingerprint(context.Background(), second)
	if err != nil {
		t.Fatalf("record second fingerprint: %v", err)
	}
	if detection == nil {
		t.Fatal("expected shared fingerprint detection")
	}
	if detection.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %q", detection.RiskLevel)
	}
	if detection.ConfidenceScore < 0.8 {
		t.Fatalf("expected confidence >= 0.8, got %f", detection.ConfidenceScore)
	}
	if len(detection.AssociatedUsers) != 2 {
		t.Fatalf("expected two associated users, got %v", detection.AssociatedUsers)
	}

	associations, total, err := service.ListAssociations(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("list associations: %v", err)
	}
	if total != 1 || len(associations) != 1 {
		t.Fatalf("expected one association, total=%d len=%d", total, len(associations))
	}
	if associations[0].RiskLevel != "high" {
		t.Fatalf("expected persisted high risk, got %q", associations[0].RiskLevel)
	}
}

func TestFingerprintServiceUsesStableServerSideID(t *testing.T) {
	repo := newMemoryFingerprintRepo()
	service := NewFingerprintService(repo)

	input := domainsecurity.DeviceFingerprint{
		UserID:              10,
		FingerprintID:       "client-supplied",
		ScreenResolution:    "1920x1080",
		ColorDepth:          24,
		PixelRatio:          2,
		HardwareConcurrency: 8,
		DeviceMemory:        8,
		CanvasHash:          "canvas-a",
		WebGLVendor:         "Apple",
		WebGLRenderer:       "Apple M4",
		FontsHash:           "fonts-a",
	}

	if _, err := service.RecordFingerprint(context.Background(), input); err != nil {
		t.Fatalf("record fingerprint: %v", err)
	}
	items, err := repo.FindByFingerprintID(context.Background(), StableFingerprintID(input))
	if err != nil {
		t.Fatalf("find by fingerprint id: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one row stored with server-side id, got %d", len(items))
	}
	if items[0].FingerprintID == "client-supplied" {
		t.Fatal("expected service to ignore client-supplied fingerprint id")
	}
}

func TestPoWServiceRaisesDifficultyForHighRiskUsers(t *testing.T) {
	store := newMemoryProofStore()
	service := NewPoWService(PoWServiceOptions{
		Store:             store,
		BaseDifficulty:    map[string]int{"send_message": 5},
		DefaultDifficulty: 5,
		MaxDifficulty:     10,
		RiskResolver:      fixedRiskResolver{level: "high"},
	})

	challenge, err := service.GenerateChallenge(context.Background(), 42, "send_message")
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	if challenge.Difficulty != 7 {
		t.Fatalf("expected high risk difficulty 7, got %d", challenge.Difficulty)
	}
}

type fixedRiskResolver struct {
	level string
}

func (r fixedRiskResolver) GetUserRiskLevel(context.Context, uint) (string, error) {
	return r.level, nil
}
