package billing

import (
	"context"
	"testing"
	"time"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestClaimDailyCheckInIsIdempotentAndWritesOneBalanceTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:billing_checkin_claim?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.BillingAccount{}, &model.BalanceTransaction{}, &model.CheckInRecord{}, &model.TaskProgress{}); err != nil {
		t.Fatalf("migrate checkin tables: %v", err)
	}

	repo := NewRepo(db)
	ctx := context.Background()
	claimDate := time.Date(2026, 6, 12, 15, 30, 0, 0, time.UTC)
	input := repository.CheckInClaimInput{
		UserID:         42,
		CheckInDate:    claimDate,
		RewardNanousd:  10_000_000,
		RefNo:          "checkin-42-20260612",
		Description:    "Daily check-in reward",
		ConsecutiveDay: 1,
	}

	first, err := repo.ClaimDailyCheckIn(ctx, input)
	if err != nil {
		t.Fatalf("first ClaimDailyCheckIn() error = %v", err)
	}
	if first.AlreadyClaimed {
		t.Fatal("first claim marked already claimed")
	}
	if first.Record.UserID != 42 || first.Record.RewardNanousd != input.RewardNanousd {
		t.Fatalf("unexpected first record: %+v", first.Record)
	}
	if first.Account == nil || first.Account.BalanceNanousd != input.RewardNanousd {
		t.Fatalf("unexpected first account: %+v", first.Account)
	}
	if first.Transaction == nil || first.Transaction.AmountNanousd != input.RewardNanousd || first.Transaction.Type != domainbilling.BalanceTransactionTypeCheckIn {
		t.Fatalf("unexpected first transaction: %+v", first.Transaction)
	}

	second, err := repo.ClaimDailyCheckIn(ctx, input)
	if err != nil {
		t.Fatalf("second ClaimDailyCheckIn() error = %v", err)
	}
	if !second.AlreadyClaimed {
		t.Fatal("second claim should be idempotent")
	}
	if second.Account == nil || second.Account.BalanceNanousd != input.RewardNanousd {
		t.Fatalf("unexpected second account: %+v", second.Account)
	}
	if second.Transaction == nil || second.Transaction.ID != first.Transaction.ID {
		t.Fatalf("second claim should return original transaction, got %+v want id %d", second.Transaction, first.Transaction.ID)
	}

	var recordCount int64
	if err := db.Model(&model.CheckInRecord{}).Where("user_id = ?", 42).Count(&recordCount).Error; err != nil {
		t.Fatalf("count checkin records: %v", err)
	}
	if recordCount != 1 {
		t.Fatalf("checkin record count = %d, want 1", recordCount)
	}
	var txCount int64
	if err := db.Model(&model.BalanceTransaction{}).Where("user_id = ? AND type = ?", 42, domainbilling.BalanceTransactionTypeCheckIn).Count(&txCount).Error; err != nil {
		t.Fatalf("count checkin transactions: %v", err)
	}
	if txCount != 1 {
		t.Fatalf("checkin transaction count = %d, want 1", txCount)
	}
}
