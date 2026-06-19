package status

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 创建测试表
	err = db.Exec(`
		CREATE TABLE usage_ledgers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform_model_name TEXT,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE channel_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform_model_name TEXT,
			status TEXT
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestGetModelsStatus_WithData(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// 插入测试数据
	now := time.Now()
	since := now.Add(-12 * time.Hour)

	db.Exec(`INSERT INTO usage_ledgers (platform_model_name, created_at) VALUES (?, ?)`, "GPT-4", since)
	db.Exec(`INSERT INTO usage_ledgers (platform_model_name, created_at) VALUES (?, ?)`, "GPT-4", since.Add(1*time.Hour))
	db.Exec(`INSERT INTO usage_ledgers (platform_model_name, created_at) VALUES (?, ?)`, "Claude-3", since)

	// 执行查询
	ctx := context.Background()
	result, err := service.GetModelsStatus(ctx)

	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "operational", result.OverallStatus)
	assert.GreaterOrEqual(t, len(result.Models), 2)

	// 验证模型存在
	modelMap := make(map[string]ModelStatusDetail)
	for _, m := range result.Models {
		modelMap[m.ModelName] = m
	}

	gpt4, ok := modelMap["GPT-4"]
	assert.True(t, ok)
	assert.Equal(t, "operational", gpt4.Status)
	assert.Equal(t, 1.0, gpt4.Availability)
}

func TestGetModelsStatus_NoData(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// 插入活跃模型
	db.Exec(`INSERT INTO channel_models (platform_model_name, status) VALUES (?, ?)`, "GPT-4", "active")

	// 执行查询（没有调用数据）
	ctx := context.Background()
	result, err := service.GetModelsStatus(ctx)

	// 验证返回默认状态
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "operational", result.OverallStatus)
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		availability float64
		expected     string
	}{
		{1.0, "operational"},
		{0.99, "operational"},
		{0.95, "operational"},
		{0.94, "degraded"},
		{0.80, "degraded"},
		{0.79, "down"},
		{0.50, "down"},
		{0.0, "down"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := determineStatus(tt.availability)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareStatusSeverity(t *testing.T) {
	assert.Equal(t, 1, compareStatusSeverity("down", "operational"))
	assert.Equal(t, 1, compareStatusSeverity("degraded", "operational"))
	assert.Equal(t, 1, compareStatusSeverity("down", "degraded"))
	assert.Equal(t, -1, compareStatusSeverity("operational", "degraded"))
	assert.Equal(t, 0, compareStatusSeverity("operational", "operational"))
}
