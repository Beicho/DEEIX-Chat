package notification

import (
	"context"
	"testing"
	"time"

	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

func TestListIncludesActiveAnnouncementsAsVirtualNotifications(t *testing.T) {
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-2 * time.Hour)
	service := NewService(&fakeNotificationRepo{}, &fakeAnnouncementProvider{
		items: []domainannouncement.Announcement{
			{
				ID:              12,
				Title:           "Maintenance",
				ContentMarkdown: "Window tonight",
				Type:            domainannouncement.TypeInfo,
				UpdatedAt:       updatedAt,
				CreatedAt:       updatedAt,
			},
		},
	})

	items, total, err := service.List(context.Background(), 7, ListInput{Page: 1, PageSize: 20}, now)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("List() total/items = %d/%d, want 1/1", total, len(items))
	}
	item := items[0]
	if item.Source != domainnotification.SourceAnnouncement || item.SourceID != "12" {
		t.Fatalf("virtual source = %q/%q, want announcement/12", item.Source, item.SourceID)
	}
	if item.ID == "" || item.ReadAt != nil || item.Link != "/announcements" {
		t.Fatalf("virtual notification = %#v", item)
	}
	if item.Title != "Maintenance" || item.Body != "Window tonight" {
		t.Fatalf("virtual content = %#v", item)
	}
}

func TestCreateNotificationTrimsInputAndDefaultsType(t *testing.T) {
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)

	item, err := service.Create(context.Background(), 7, CreateInput{
		Title:    "  Hello  ",
		Body:     "  Body  ",
		Link:     " /settings ",
		Source:   " billing ",
		SourceID: " order-1 ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.Type != "system" || item.Title != "Hello" || item.Body != "Body" || item.Link != "/settings" {
		t.Fatalf("Create() item = %#v", item)
	}
	if repo.created.UserID != 7 || repo.created.Source != "billing" || repo.created.SourceID != "order-1" {
		t.Fatalf("created notification = %#v", repo.created)
	}
}

func TestUnreadCountIncludesUnreadAnnouncements(t *testing.T) {
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	closedAt := now.Add(-time.Minute)
	service := NewService(&fakeNotificationRepo{unreadCount: 2}, &fakeAnnouncementProvider{
		items: []domainannouncement.Announcement{
			{ID: 1, Title: "Unread", UpdatedAt: now.Add(-time.Hour)},
			{ID: 2, Title: "Read", UpdatedAt: now.Add(-2 * time.Hour), ClosedAt: &closedAt},
		},
	})

	count, err := service.UnreadCount(context.Background(), 7, now)
	if err != nil {
		t.Fatalf("UnreadCount() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("UnreadCount() = %d, want 3", count)
	}
}

func TestMarkReadClosesAnnouncementNotificationVersion(t *testing.T) {
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	updatedAt := now.Add(-time.Hour)
	announcements := &fakeAnnouncementProvider{
		items: []domainannouncement.Announcement{
			{ID: 42, Title: "Notice", UpdatedAt: updatedAt},
		},
	}
	service := NewService(&fakeNotificationRepo{}, announcements)

	items, _, err := service.List(context.Background(), 7, ListInput{Page: 1, PageSize: 20}, now)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if err := service.MarkRead(context.Background(), 7, items[0].ID, now); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}

	if len(announcements.closeCalls) != 1 {
		t.Fatalf("close calls = %d, want 1", len(announcements.closeCalls))
	}
	call := announcements.closeCalls[0]
	if call.userID != 7 || call.announcementID != 42 || !call.updatedAt.Equal(updatedAt) {
		t.Fatalf("close call = %#v", call)
	}
}

func TestMarkAllReadMarksStoredAndAnnouncementNotifications(t *testing.T) {
	now := time.Date(2026, 6, 11, 8, 0, 0, 0, time.UTC)
	repo := &fakeNotificationRepo{}
	announcements := &fakeAnnouncementProvider{
		items: []domainannouncement.Announcement{
			{ID: 1, Title: "A", UpdatedAt: now.Add(-time.Hour)},
			{ID: 2, Title: "B", UpdatedAt: now.Add(-2 * time.Hour), ClosedAt: ptrTime(now.Add(-time.Minute))},
		},
	}
	service := NewService(repo, announcements)

	if err := service.MarkAllRead(context.Background(), 7, now); err != nil {
		t.Fatalf("MarkAllRead() error = %v", err)
	}
	if !repo.markAllCalled {
		t.Fatal("stored notifications were not marked read")
	}
	if len(announcements.closeCalls) != 1 || announcements.closeCalls[0].announcementID != 1 {
		t.Fatalf("announcement close calls = %#v", announcements.closeCalls)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

type fakeNotificationRepo struct {
	items         []domainnotification.Notification
	total         int64
	unreadCount   int64
	markedID      uint
	markAllCalled bool
	created       domainnotification.Notification
}

func (r *fakeNotificationRepo) CreateNotification(_ context.Context, item *domainnotification.Notification) (*domainnotification.Notification, error) {
	item.ID = 1
	r.created = *item
	return item, nil
}

func (r *fakeNotificationRepo) ListNotifications(context.Context, uint, repository.NotificationListFilter, int, int) ([]domainnotification.Notification, int64, error) {
	total := r.total
	if total == 0 {
		total = int64(len(r.items))
	}
	return r.items, total, nil
}

func (r *fakeNotificationRepo) CountUnreadNotifications(context.Context, uint) (int64, error) {
	return r.unreadCount, nil
}

func (r *fakeNotificationRepo) MarkNotificationRead(_ context.Context, _ uint, notificationID uint, _ time.Time) error {
	r.markedID = notificationID
	return nil
}

func (r *fakeNotificationRepo) MarkAllNotificationsRead(context.Context, uint, time.Time) error {
	r.markAllCalled = true
	return nil
}

type fakeAnnouncementProvider struct {
	items      []domainannouncement.Announcement
	closeCalls []announcementCloseCall
}

type announcementCloseCall struct {
	userID         uint
	announcementID uint
	updatedAt      time.Time
	now            time.Time
}

func (p *fakeAnnouncementProvider) ListActive(context.Context, uint, time.Time, bool) ([]domainannouncement.Announcement, error) {
	return p.items, nil
}

func (p *fakeAnnouncementProvider) Close(_ context.Context, userID uint, announcementID uint, updatedAt time.Time, now time.Time) error {
	p.closeCalls = append(p.closeCalls, announcementCloseCall{
		userID:         userID,
		announcementID: announcementID,
		updatedAt:      updatedAt,
		now:            now,
	})
	return nil
}
