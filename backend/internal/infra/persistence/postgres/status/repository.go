package status

import (
	"context"
	"time"

	domainstatus "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/status"
	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListRecentModelUsage(ctx context.Context, since time.Time) ([]domainstatus.UsageStat, error) {
	items := make([]domainstatus.UsageStat, 0)
	err := r.db.WithContext(ctx).
		Table("billing_usage_ledgers").
		Select("platform_model_name, COUNT(*) AS total_calls, COUNT(*) AS success_calls").
		Where("created_at >= ? AND platform_model_name <> ''", since).
		Group("platform_model_name").
		Order("platform_model_name ASC").
		Scan(&items).Error
	return items, err
}

func (r *Repo) ListPublicActiveModelNames(ctx context.Context) ([]string, error) {
	items := make([]string, 0)
	err := r.db.WithContext(ctx).
		Table("llm_platform_models").
		Where("status = ? AND access_scope = ?", "active", "public").
		Distinct().
		Order("name ASC").
		Pluck("name", &items).Error
	return items, err
}
