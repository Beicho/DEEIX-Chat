package collaboration

import (
	"context"
	"testing"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	domaincollab "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/collaboration"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

func TestComputeNextRunAtForDailySchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	next, err := computeNextRunAt("daily", "08:15", 0, "", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 13, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestComputeNextRunAtForWeeklySchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC) // Friday
	next, err := computeNextRunAt("weekly", "08:15", int(time.Monday), "", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 15, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestComputeNextRunAtForCronSchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	next, err := computeNextRunAt("cron", "", 0, "15 8 * * *", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 13, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestRunScheduledPromptCreatesScheduledNotificationType(t *testing.T) {
	conversationID := uint(55)
	notificationRepo := &scheduledNotificationRepo{}
	service := &Service{
		repo:          &assistantModerationRepo{},
		conversations: &scheduledConversationService{},
		notifications: appnotification.NewService(notificationRepo, nil),
	}

	err := service.runScheduledPrompt(context.Background(), domaincollab.ScheduledPrompt{
		ID:                         9,
		PublicID:                   "sched_1",
		UserID:                     7,
		TargetConversationID:       &conversationID,
		TargetConversationPublicID: "conv_55",
		Title:                      "Daily summary",
		Content:                    "summarize",
		ScheduleType:               "once",
		Enabled:                    true,
	}, time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("runScheduledPrompt() error = %v", err)
	}
	if notificationRepo.created == nil {
		t.Fatal("expected notification to be created")
	}
	if notificationRepo.created.Type != domainnotification.TypeScheduledPrompt ||
		notificationRepo.created.Source != domainnotification.SourceScheduledPrompt ||
		notificationRepo.created.SourceID != "sched_1" {
		t.Fatalf("notification contract = %#v", notificationRepo.created)
	}
	if notificationRepo.created.Link != "/chat/conv_55" {
		t.Fatalf("notification link = %q", notificationRepo.created.Link)
	}
}

type scheduledConversationService struct{}

func (s *scheduledConversationService) CreateConversation(context.Context, uint, string, string, string) (*domainconversation.Conversation, error) {
	return &domainconversation.Conversation{ID: 55, PublicID: "conv_55"}, nil
}

func (s *scheduledConversationService) SendScheduledPromptMessage(context.Context, ScheduledPromptSendInput) (*ScheduledPromptSendResult, error) {
	return &ScheduledPromptSendResult{AssistantContent: "done"}, nil
}

type scheduledNotificationRepo struct {
	created *domainnotification.Notification
}

func (r *scheduledNotificationRepo) CreateNotification(_ context.Context, item *domainnotification.Notification) (*domainnotification.Notification, error) {
	copied := *item
	if copied.ID == 0 {
		copied.ID = 1
	}
	r.created = &copied
	return &copied, nil
}

func (r *scheduledNotificationRepo) GetNotificationBySource(context.Context, uint, string, string) (*domainnotification.Notification, error) {
	return nil, repository.ErrNotFound
}

func (r *scheduledNotificationRepo) ListNotifications(context.Context, uint, repository.NotificationListFilter, int, int) ([]domainnotification.Notification, int64, error) {
	return nil, 0, nil
}

func (r *scheduledNotificationRepo) CountUnreadNotifications(context.Context, uint) (int64, error) {
	return 0, nil
}

func (r *scheduledNotificationRepo) MarkNotificationRead(context.Context, uint, uint, time.Time) error {
	return nil
}

func (r *scheduledNotificationRepo) MarkAllNotificationsRead(context.Context, uint, time.Time) error {
	return nil
}
