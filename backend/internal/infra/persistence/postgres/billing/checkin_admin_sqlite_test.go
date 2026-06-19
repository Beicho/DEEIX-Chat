package billing

import (
	"context"
	"testing"
	"time"

	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminCheckInStatsAndRewardConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:billing_checkin_admin?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.CheckInRecord{}, &model.SystemSetting{}); err != nil {
		t.Fatalf("migrate checkin admin tables: %v", err)
	}

	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	records := []model.CheckInRecord{
		{UserID: 1, CheckInDate: now.AddDate(0, 0, -1), RewardNanousd: 10_000_000, ConsecutiveDays: 3, RefNo: "checkin-1"},
		{UserID: 2, CheckInDate: now.AddDate(0, 0, -2), RewardNanousd: 20_000_000, ConsecutiveDays: 1, RefNo: "checkin-2"},
		{UserID: 3, CheckInDate: now.AddDate(0, 0, -10), RewardNanousd: 30_000_000, ConsecutiveDays: 1, RefNo: "checkin-3"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed checkins: %v", err)
	}

	repo := NewRepo(db)
	ctx := context.Background()
	if err := repo.SetCheckInRewardNanousd(ctx, 25_000_000); err != nil {
		t.Fatalf("SetCheckInRewardNanousd() error = %v", err)
	}
	reward, err := repo.GetCheckInRewardNanousd(ctx)
	if err != nil {
		t.Fatalf("GetCheckInRewardNanousd() error = %v", err)
	}
	if reward != 25_000_000 {
		t.Fatalf("reward = %d, want 25000000", reward)
	}

	stats, err := repo.GetAdminCheckInStats(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("GetAdminCheckInStats() error = %v", err)
	}
	if stats.ActiveUsersLast7Days != 2 {
		t.Fatalf("ActiveUsersLast7Days = %d, want 2", stats.ActiveUsersLast7Days)
	}
	if stats.TotalClaims != 3 {
		t.Fatalf("TotalClaims = %d, want 3", stats.TotalClaims)
	}
	if stats.TotalRewardNanousd != 60_000_000 {
		t.Fatalf("TotalRewardNanousd = %d, want 60000000", stats.TotalRewardNanousd)
	}
	if stats.AverageConsecutiveDays != 2 {
		t.Fatalf("AverageConsecutiveDays = %v, want 2", stats.AverageConsecutiveDays)
	}
}
