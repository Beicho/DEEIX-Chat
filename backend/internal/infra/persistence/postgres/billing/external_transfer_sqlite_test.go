package billing

import (
	"context"
	"testing"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExternalTransferCreditIsIdempotentAndCreatesBalanceTransaction(t *testing.T) {
	db := openExternalTransferSQLiteTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	link := &domainbilling.ExternalAccountLink{
		UserID:              42,
		Platform:            domainbilling.ExternalPlatformNewAPI,
		ExternalUserID:      "1001",
		ExternalDisplayName: "newapi-user",
		LinuxDOSub:          "linuxdo-sub-42",
		Status:              domainbilling.ExternalAccountLinkStatusActive,
	}
	createdLink, err := repo.UpsertExternalAccountLink(ctx, link)
	if err != nil {
		t.Fatalf("upsert external account link: %v", err)
	}

	transfer, transaction, err := repo.CreditExternalTransfer(ctx, repository.ExternalTransferCreditInput{
		UserID:                42,
		LinkID:                createdLink.ID,
		Platform:              domainbilling.ExternalPlatformNewAPI,
		Direction:             domainbilling.ExternalTransferDirectionIn,
		ExternalTransferID:    "newapi-transfer-1",
		IdempotencyKey:        "idem-1",
		ExternalAmountUSD:     10,
		CreditedAmountNanousd: 1_000_000_000,
		RefNo:                 "req-1",
		Description:           "NewAPI transfer in",
	})
	if err != nil {
		t.Fatalf("credit external transfer: %v", err)
	}
	if transfer.Status != domainbilling.ExternalTransferStatusCredited {
		t.Fatalf("expected credited transfer, got %#v", transfer)
	}
	if transaction == nil || transaction.Type != domainbilling.BalanceTransactionTypeNewAPITransferIn {
		t.Fatalf("expected newapi transfer balance transaction, got %#v", transaction)
	}

	again, againTransaction, err := repo.CreditExternalTransfer(ctx, repository.ExternalTransferCreditInput{
		UserID:                42,
		LinkID:                createdLink.ID,
		Platform:              domainbilling.ExternalPlatformNewAPI,
		Direction:             domainbilling.ExternalTransferDirectionIn,
		ExternalTransferID:    "newapi-transfer-1",
		IdempotencyKey:        "idem-1",
		ExternalAmountUSD:     10,
		CreditedAmountNanousd: 1_000_000_000,
		RefNo:                 "req-1",
		Description:           "NewAPI transfer in",
	})
	if err != nil {
		t.Fatalf("idempotent credit external transfer: %v", err)
	}
	if again.ID != transfer.ID {
		t.Fatalf("expected same transfer id, got %d and %d", again.ID, transfer.ID)
	}
	if againTransaction != nil {
		t.Fatalf("expected no duplicate balance transaction, got %#v", againTransaction)
	}

	var txCount int64
	if err := db.Model(&model.BalanceTransaction{}).Where("user_id = ?", 42).Count(&txCount).Error; err != nil {
		t.Fatalf("count balance transactions: %v", err)
	}
	if txCount != 1 {
		t.Fatalf("expected exactly one balance transaction, got %d", txCount)
	}

	account, err := repo.GetOrCreateBillingAccount(ctx, 42)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if account.BalanceNanousd != 1_000_000_000 {
		t.Fatalf("expected balance 1e9, got %d", account.BalanceNanousd)
	}
}

func TestListBalanceTransactionsIncludesExternalTransferType(t *testing.T) {
	db := openExternalTransferSQLiteTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	_, _, err := repo.CreditExternalTransfer(ctx, repository.ExternalTransferCreditInput{
		UserID:                7,
		Platform:              domainbilling.ExternalPlatformNewAPI,
		Direction:             domainbilling.ExternalTransferDirectionIn,
		ExternalTransferID:    "newapi-transfer-history",
		IdempotencyKey:        "history-idem",
		ExternalAmountUSD:     20,
		CreditedAmountNanousd: 2_000_000_000,
		RefNo:                 "history-ref",
		Description:           "NewAPI transfer in",
	})
	if err != nil {
		t.Fatalf("credit external transfer: %v", err)
	}

	items, total, err := repo.ListBalanceTransactions(ctx, repository.BalanceTransactionListFilter{
		UserID: 7,
		Type:   domainbilling.BalanceTransactionTypeNewAPITransferIn,
	}, 0, 20)
	if err != nil {
		t.Fatalf("list balance transactions: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one transfer transaction, total=%d len=%d", total, len(items))
	}
	if items[0].RefType != domainbilling.BalanceTransactionRefTypeExternalTransfer || items[0].AmountNanousd != 2_000_000_000 {
		t.Fatalf("unexpected transfer transaction: %#v", items[0])
	}
}

func openExternalTransferSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:external_transfer?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(
		&model.BillingAccount{},
		&model.BalanceTransaction{},
		&model.ExternalAccountLink{},
		&model.ExternalTransfer{},
		&model.SystemSetting{},
	); err != nil {
		t.Fatalf("migrate external transfer tables: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Namespace: "billing", Key: "mode", Value: "usage", ValueType: "string"}).Error; err != nil {
		t.Fatalf("seed billing mode: %v", err)
	}
	return db
}
