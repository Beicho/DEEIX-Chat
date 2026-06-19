package status

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Service 提供模型状态聚合能力。
type Service struct {
	db *gorm.DB
}

// NewService 创建状态服务。
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// GetModelsStatus 聚合并返回模型可用性状态。
func (s *Service) GetModelsStatus(ctx context.Context) (*ModelsStatusResponse, error) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)

	// 聚合最近 24 小时的模型调用数据
	type ModelStats struct {
		PlatformModelName string
		TotalCalls        int64
		SuccessCalls      int64
	}

	var stats []ModelStats

	// 从 usage_ledgers 表聚合数据
	// 我们认为有 call_count 记录的即为成功调用（因为失败的调用通常不会写入 usage_ledger）
	// 如果需要更精确的成功率，需要区分成功/失败状态字段
	err := s.db.WithContext(ctx).
		Table("billing_usage_ledgers").
		Select("platform_model_name, COUNT(*) as total_calls, COUNT(*) as success_calls").
		Where("created_at >= ? AND platform_model_name <> ''", since).
		Group("platform_model_name").
		Find(&stats).Error

	if err != nil {
		return nil, err
	}

	// 如果没有调用数据，返回默认状态（所有已知模型 operational）
	if len(stats) == 0 {
		return s.getDefaultStatus(ctx, now), nil
	}

	// 构建响应
	models := make([]ModelStatusDetail, 0, len(stats))
	worstStatus := "operational"

	for _, stat := range stats {
		if stat.PlatformModelName == "" {
			continue
		}

		// 计算可用性（这里假设所有记录都是成功的，因为失败的调用可能不写入 usage_ledger）
		// 实际生产中可能需要从错误日志或其他表获取失败率
		availability := 1.0
		if stat.TotalCalls > 0 {
			availability = float64(stat.SuccessCalls) / float64(stat.TotalCalls)
		}

		// 判定状态
		status := determineStatus(availability)
		if compareStatusSeverity(status, worstStatus) > 0 {
			worstStatus = status
		}

		models = append(models, ModelStatusDetail{
			ModelName:    stat.PlatformModelName,
			Availability: availability,
			Status:       status,
			LastChecked:  now,
		})
	}

	return &ModelsStatusResponse{
		OverallStatus: worstStatus,
		LastUpdated:   now,
		Models:        models,
	}, nil
}

// getDefaultStatus 在没有调用数据时返回默认状态。
func (s *Service) getDefaultStatus(ctx context.Context, now time.Time) *ModelsStatusResponse {
	// 查询所有启用的模型（从 channel_models 表）
	type ActiveModel struct {
		PlatformModelName string
	}

	var activeModels []ActiveModel
	err := s.db.WithContext(ctx).
		Table("llm_platform_models").
		Select("DISTINCT name AS platform_model_name").
		Where("status = ? AND access_scope = ?", "active", "public").
		Find(&activeModels).Error

	models := make([]ModelStatusDetail, 0)
	if err == nil && len(activeModels) > 0 {
		for _, m := range activeModels {
			if m.PlatformModelName == "" {
				continue
			}
			models = append(models, ModelStatusDetail{
				ModelName:    m.PlatformModelName,
				Availability: 1.0,
				Status:       "operational",
				LastChecked:  now,
			})
		}
	}

	return &ModelsStatusResponse{
		OverallStatus: "operational",
		LastUpdated:   now,
		Models:        models,
	}
}

// determineStatus 根据可用性判定状态。
func determineStatus(availability float64) string {
	if availability >= 0.95 {
		return "operational"
	}
	if availability >= 0.80 {
		return "degraded"
	}
	return "down"
}

// compareStatusSeverity 比较状态严重程度（返回 1 表示 a 更严重，-1 表示 b 更严重，0 表示相同）。
func compareStatusSeverity(a, b string) int {
	severity := map[string]int{
		"operational": 0,
		"degraded":    1,
		"down":        2,
	}
	aScore := severity[a]
	bScore := severity[b]
	if aScore > bScore {
		return 1
	}
	if aScore < bScore {
		return -1
	}
	return 0
}
