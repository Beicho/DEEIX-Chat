package notification

import (
	"context"
	"testing"
	"time"

	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
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
	if item.ID == "" || item.ReadAt != nil || item.ActionURL != "/announcements" {
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
		Title:     "  Hello  ",
		Body:      "  Body  ",
		ActionURL: " /settings ",
		Source:    " billing ",
		SourceID:  " order-1 ",
		Metadata: map[string]any{
			"kind": "renewal",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.Type != "system" || item.Title != "Hello" || item.Body != "Body" || item.ActionURL != "/settings" {
		t.Fatalf("Create() item = %#v", item)
	}
	if repo.created.UserID != 7 || repo.created.Source != "billing" || repo.created.SourceID != "order-1" {
		t.Fatalf("created notification = %#v", repo.created)
	}
	if repo.created.ActionURL != "/settings" || repo.created.Metadata["kind"] != "renewal" {
		t.Fatalf("created notification action/metadata = %#v", repo.created)
	}
}

func TestNotifyUserAndCreateSystemNotificationExposeStableProducerAPI(t *testing.T) {
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)

	notified, err := service.NotifyUser(context.Background(), 7, CreateInput{
		Type:      "subscription",
		Title:     "Plan renewed",
		Body:      "Your plan is active.",
		ActionURL: "/settings/billing",
		Source:    "billing",
		SourceID:  "order-2",
		Metadata: map[string]any{
			"plan": "pro",
		},
	})
	if err != nil {
		t.Fatalf("NotifyUser() error = %v", err)
	}
	if notified.Type != "subscription" || notified.ActionURL != "/settings/billing" || notified.Metadata["plan"] != "pro" {
		t.Fatalf("NotifyUser() = %#v", notified)
	}

	system, err := service.CreateSystemNotification(context.Background(), 7, SystemNotificationInput{
		Title:     "New sign-in",
		Body:      "A device signed in.",
		ActionURL: "/settings/security",
		Source:    "auth",
		SourceID:  "session-1",
		Metadata: map[string]any{
			"device": "desktop",
		},
	})
	if err != nil {
		t.Fatalf("CreateSystemNotification() error = %v", err)
	}
	if system.Type != "system" || system.Source != "auth" || system.Metadata["device"] != "desktop" {
		t.Fatalf("CreateSystemNotification() = %#v", system)
	}
}

func TestCreateSystemNotificationOnceDeduplicatesBySource(t *testing.T) {
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)
	input := SystemNotificationInput{
		Title:    "Weekly summary",
		Body:     "You used 3 chats this week.",
		Source:   "weekly_summary",
		SourceID: "2026-W24",
	}

	first, created, err := service.CreateSystemNotificationOnce(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("CreateSystemNotificationOnce() first error = %v", err)
	}
	if !created || first == nil {
		t.Fatalf("first created = %v, item = %#v", created, first)
	}
	second, created, err := service.CreateSystemNotificationOnce(context.Background(), 7, input)
	if err != nil {
		t.Fatalf("CreateSystemNotificationOnce() second error = %v", err)
	}
	if created || second == nil || second.ID != first.ID {
		t.Fatalf("second created = %v, item = %#v, first = %#v", created, second, first)
	}
	if repo.createCount != 1 {
		t.Fatalf("create count = %d, want 1", repo.createCount)
	}
}

func TestCreateSystemNotificationUsesPublicTypeWhenProvided(t *testing.T) {
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)

	item, err := service.CreateSystemNotification(context.Background(), 7, SystemNotificationInput{
		Type:     "billing_expiry",
		Title:    "Subscription reminder",
		Body:     "Your subscription is expiring soon.",
		Source:   "billing_expiry",
		SourceID: "billing_expiry:2026-06-17",
	})
	if err != nil {
		t.Fatalf("CreateSystemNotification() error = %v", err)
	}
	if item.Type != "billing_expiry" {
		t.Fatalf("public type = %q, want billing_expiry", item.Type)
	}

	defaulted, err := service.CreateSystemNotification(context.Background(), 7, SystemNotificationInput{
		Title:    "System notice",
		Body:     "A system event happened.",
		Source:   "system",
		SourceID: "system:1",
	})
	if err != nil {
		t.Fatalf("CreateSystemNotification() default error = %v", err)
	}
	if defaulted.Type != domainnotification.TypeSystem {
		t.Fatalf("default type = %q, want %q", defaulted.Type, domainnotification.TypeSystem)
	}
}

func TestLifecycleNotificationScanCreatesExpiryAndWeeklySummaryOnce(t *testing.T) {
	now := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC) // Monday
	expiresAt := now.Add(48 * time.Hour)
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)
	service.SetUserNotificationProviders(
		&fakeUserProvider{users: []domainuser.User{
			{ID: 7, Role: domainuser.RoleUser, Status: domainuser.StatusActive},
		}},
		&fakeBillingProvider{
			snapshot: &appbilling.UserSubscriptionSnapshot{
				UserID:    7,
				PlanName:  "Pro",
				Tier:      "pro",
				Status:    "active",
				ExpiresAt: &expiresAt,
			},
			daily: []domainbilling.UsageDailySummary{
				{UsageDate: now.AddDate(0, 0, -1), RecordCount: 3, CallCount: 4, BilledNanousd: 1_230_000_000},
			},
		},
	)

	service.runLifecycleNotificationScan(context.Background(), now)
	service.runLifecycleNotificationScan(context.Background(), now)

	if repo.createCount != 2 {
		t.Fatalf("create count = %d, want 2", repo.createCount)
	}
	sources := map[string]bool{}
	sourceIDs := map[string]string{}
	for _, item := range repo.items {
		sources[item.Source] = true
		sourceIDs[item.Source] = item.SourceID
	}
	if !sources["billing_expiry"] || !sources["weekly_summary"] {
		t.Fatalf("created sources = %#v", sources)
	}
	if sourceIDs["billing_expiry"] != "billing_expiry:2026-06-17" {
		t.Fatalf("billing expiry source ID = %q", sourceIDs["billing_expiry"])
	}
	if sourceIDs["weekly_summary"] != "weekly_summary:2026-W24" {
		t.Fatalf("weekly summary source ID = %q", sourceIDs["weekly_summary"])
	}
}

func TestLifecycleNotificationScanSkipsInactiveUsers(t *testing.T) {
	now := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	expiresAt := now.Add(48 * time.Hour)
	repo := &fakeNotificationRepo{}
	service := NewService(repo, nil)
	service.SetUserNotificationProviders(
		&fakeUserProvider{users: []domainuser.User{
			{ID: 7, Role: domainuser.RoleUser, Status: domainuser.StatusSuspended},
			{ID: 8, Role: domainuser.RoleAdmin, Status: domainuser.StatusActive},
		}},
		&fakeBillingProvider{
			snapshot: &appbilling.UserSubscriptionSnapshot{UserID: 7, ExpiresAt: &expiresAt},
			daily:    []domainbilling.UsageDailySummary{{RecordCount: 1, CallCount: 1}},
		},
	)

	service.runLifecycleNotificationScan(context.Background(), now)

	if repo.createCount != 0 {
		t.Fatalf("create count = %d, want 0", repo.createCount)
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
	createCount   int
}

func (r *fakeNotificationRepo) CreateNotification(_ context.Context, item *domainnotification.Notification) (*domainnotification.Notification, error) {
	r.createCount++
	item.ID = uint(r.createCount)
	r.created = *item
	r.items = append(r.items, *item)
	return item, nil
}

func (r *fakeNotificationRepo) GetNotificationBySource(_ context.Context, userID uint, source string, sourceID string) (*domainnotification.Notification, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.Source == source && item.SourceID == sourceID {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, repository.ErrNotFound
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

type fakeUserProvider struct {
	users []domainuser.User
}

func (p *fakeUserProvider) ListUsers(_ context.Context, page int, pageSize int) ([]domainuser.User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = len(p.users)
	}
	offset := (page - 1) * pageSize
	if offset >= len(p.users) {
		return []domainuser.User{}, int64(len(p.users)), nil
	}
	end := offset + pageSize
	if end > len(p.users) {
		end = len(p.users)
	}
	return p.users[offset:end], int64(len(p.users)), nil
}

type fakeBillingProvider struct {
	snapshot *appbilling.UserSubscriptionSnapshot
	daily    []domainbilling.UsageDailySummary
}

func (p *fakeBillingProvider) GetCurrentSubscriptionSnapshot(context.Context, uint, time.Time) (*appbilling.UserSubscriptionSnapshot, error) {
	return p.snapshot, nil
}

func (p *fakeBillingProvider) ListDailyUsageRange(context.Context, uint, time.Time, time.Time) ([]domainbilling.UsageDailySummary, error) {
	return p.daily, nil
}
