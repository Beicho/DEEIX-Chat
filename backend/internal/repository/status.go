package repository

import (
	"context"
	"time"

	domainstatus "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/status"
)

// ModelStatusRepository provides the persistence queries used by the public status page.
type ModelStatusRepository interface {
	ListRecentModelUsage(ctx context.Context, since time.Time) ([]domainstatus.UsageStat, error)
	ListPublicActiveModelNames(ctx context.Context) ([]string, error)
}
