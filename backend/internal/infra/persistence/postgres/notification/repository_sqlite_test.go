package notification

import (
	"context"
	"testing"
	"time"

	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationRepositoryListsAndMarksUnreadRows(t *testing.T) {
	db := openNotificationSQLiteTestDB(t)
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	readAt := now.Add(-time.Hour)
	records := []model.Notification{
		{UserID: 7, Type: "system", Title: "Unread", Body: "Body", Link: "/settings", Source: "test", SourceID: "a"},
		{UserID: 7, Type: "system", Title: "Read", Body: "Body", Link: "/settings", Source: "test", SourceID: "b", ReadAt: &readAt},
		{UserID: 8, Type: "system", Title: "Other", Body: "Body", Link: "/settings", Source: "test", SourceID: "c"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create notifications: %v", err)
	}

	repo := NewRepo(db)
	items, total, err := repo.ListNotifications(context.Background(), 7, repository.NotificationListFilter{UnreadOnly: true}, 0, 20)
	if err != nil {
		t.Fatalf("ListNotifications() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Title != "Unread" {
		t.Fatalf("unread list = total %d items %#v", total, items)
	}

	count, err := repo.CountUnreadNotifications(context.Background(), 7)
	if err != nil {
		t.Fatalf("CountUnreadNotifications() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("CountUnreadNotifications() = %d, want 1", count)
	}

	if err := repo.MarkNotificationRead(context.Background(), 7, records[0].ID, now); err != nil {
		t.Fatalf("MarkNotificationRead() error = %v", err)
	}
	count, err = repo.CountUnreadNotifications(context.Background(), 7)
	if err != nil {
		t.Fatalf("CountUnreadNotifications() after read error = %v", err)
	}
	if count != 0 {
		t.Fatalf("CountUnreadNotifications() after read = %d, want 0", count)
	}
}

func TestNotificationRepositoryCreatesNotification(t *testing.T) {
	db := openNotificationSQLiteTestDB(t)
	repo := NewRepo(db)

	item, err := repo.CreateNotification(context.Background(), &domainnotification.Notification{
		UserID:   7,
		Type:     "system",
		Title:    "Created",
		Body:     "Body",
		Link:     "/settings",
		Source:   "test",
		SourceID: "created",
	})
	if err != nil {
		t.Fatalf("CreateNotification() error = %v", err)
	}
	if item.ID == 0 || item.UserID != 7 || item.Title != "Created" {
		t.Fatalf("CreateNotification() item = %#v", item)
	}
}

func TestNotificationRepositoryMarkAllReadScopesToUser(t *testing.T) {
	db := openNotificationSQLiteTestDB(t)
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	records := []model.Notification{
		{UserID: 7, Type: "system", Title: "A", Source: "test", SourceID: "a"},
		{UserID: 7, Type: "system", Title: "B", Source: "test", SourceID: "b"},
		{UserID: 8, Type: "system", Title: "C", Source: "test", SourceID: "c"},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create notifications: %v", err)
	}

	repo := NewRepo(db)
	if err := repo.MarkAllNotificationsRead(context.Background(), 7, now); err != nil {
		t.Fatalf("MarkAllNotificationsRead() error = %v", err)
	}

	count, err := repo.CountUnreadNotifications(context.Background(), 7)
	if err != nil {
		t.Fatalf("CountUnreadNotifications(user 7) error = %v", err)
	}
	if count != 0 {
		t.Fatalf("user 7 unread count = %d, want 0", count)
	}
	count, err = repo.CountUnreadNotifications(context.Background(), 8)
	if err != nil {
		t.Fatalf("CountUnreadNotifications(user 8) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("user 8 unread count = %d, want 1", count)
	}
}

func openNotificationSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:notifications?mode=memory&cache=shared"), &gorm.Config{})
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

	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate notification table: %v", err)
	}
	return db
}
