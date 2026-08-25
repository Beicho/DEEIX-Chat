package status

import (
	"context"
	"strings"
	"time"

	domainstatus "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/status"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// Service aggregates model availability without exposing persistence details to HTTP handlers.
type Service struct {
	repo repository.ModelStatusRepository
	now  func() time.Time
}

func NewService(repo repository.ModelStatusRepository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) GetModelsStatus(ctx context.Context) (*domainstatus.ModelsStatus, error) {
	now := s.now().UTC()
	stats, err := s.repo.ListRecentModelUsage(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return s.defaultStatus(ctx, now), nil
	}

	models := make([]domainstatus.ModelDetail, 0, len(stats))
	worst := "operational"
	for _, stat := range stats {
		name := strings.TrimSpace(stat.PlatformModelName)
		if name == "" {
			continue
		}
		availability := 1.0
		if stat.TotalCalls > 0 {
			availability = float64(stat.SuccessCalls) / float64(stat.TotalCalls)
		}
		state := determineStatus(availability)
		if compareStatusSeverity(state, worst) > 0 {
			worst = state
		}
		models = append(models, domainstatus.ModelDetail{
			ModelName: name, Availability: availability, Status: state, LastChecked: now,
		})
	}
	return &domainstatus.ModelsStatus{OverallStatus: worst, LastUpdated: now, Models: models}, nil
}

func (s *Service) defaultStatus(ctx context.Context, now time.Time) *domainstatus.ModelsStatus {
	names, err := s.repo.ListPublicActiveModelNames(ctx)
	if err != nil {
		names = nil
	}
	models := make([]domainstatus.ModelDetail, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		models = append(models, domainstatus.ModelDetail{
			ModelName: name, Availability: 1, Status: "operational", LastChecked: now,
		})
	}
	return &domainstatus.ModelsStatus{OverallStatus: "operational", LastUpdated: now, Models: models}
}

func determineStatus(availability float64) string {
	if availability >= 0.95 {
		return "operational"
	}
	if availability >= 0.80 {
		return "degraded"
	}
	return "down"
}

func compareStatusSeverity(a string, b string) int {
	severity := map[string]int{"operational": 0, "degraded": 1, "down": 2}
	if severity[a] > severity[b] {
		return 1
	}
	if severity[a] < severity[b] {
		return -1
	}
	return 0
}
