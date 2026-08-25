package status

import (
	"context"
	"testing"
	"time"

	domainstatus "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/status"
)

type fakeModelStatusRepo struct {
	usage []domainstatus.UsageStat
	names []string
}

func (f fakeModelStatusRepo) ListRecentModelUsage(context.Context, time.Time) ([]domainstatus.UsageStat, error) {
	return f.usage, nil
}

func (f fakeModelStatusRepo) ListPublicActiveModelNames(context.Context) ([]string, error) {
	return f.names, nil
}

func TestGetModelsStatusWithUsage(t *testing.T) {
	service := NewService(fakeModelStatusRepo{usage: []domainstatus.UsageStat{
		{PlatformModelName: "GPT-4", TotalCalls: 2, SuccessCalls: 2},
		{PlatformModelName: "Claude-3", TotalCalls: 10, SuccessCalls: 8},
	}})
	service.now = func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }

	result, err := service.GetModelsStatus(context.Background())
	if err != nil {
		t.Fatalf("GetModelsStatus() error = %v", err)
	}
	if result.OverallStatus != "degraded" || len(result.Models) != 2 {
		t.Fatalf("unexpected status: %#v", result)
	}
}

func TestGetModelsStatusDefaultsToPublicModels(t *testing.T) {
	service := NewService(fakeModelStatusRepo{names: []string{"GPT-4"}})
	result, err := service.GetModelsStatus(context.Background())
	if err != nil {
		t.Fatalf("GetModelsStatus() error = %v", err)
	}
	if result.OverallStatus != "operational" || len(result.Models) != 1 || result.Models[0].ModelName != "GPT-4" {
		t.Fatalf("unexpected default status: %#v", result)
	}
}

func TestDetermineStatus(t *testing.T) {
	cases := map[float64]string{1: "operational", 0.95: "operational", 0.94: "degraded", 0.80: "degraded", 0.79: "down"}
	for availability, expected := range cases {
		if got := determineStatus(availability); got != expected {
			t.Fatalf("determineStatus(%v) = %q, want %q", availability, got, expected)
		}
	}
}
