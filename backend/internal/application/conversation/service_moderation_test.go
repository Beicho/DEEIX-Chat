package conversation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

func TestNormalizeModerationRuntimeConfigDefaults(t *testing.T) {
	cfg := normalizeModerationRuntimeConfig(config.Config{})

	if cfg.Mode != moderationModeModerations {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, moderationModeModerations)
	}
	if cfg.FailStrategy != moderationFailOpen {
		t.Fatalf("FailStrategy = %q, want %q", cfg.FailStrategy, moderationFailOpen)
	}
	if cfg.Model != "omni-moderation-latest" {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.Threshold != 0.5 {
		t.Fatalf("Threshold = %v, want 0.5", cfg.Threshold)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Fatalf("TimeoutSeconds = %d, want 10", cfg.TimeoutSeconds)
	}
	if cfg.AutoWindowHours != 24 {
		t.Fatalf("AutoWindowHours = %d, want 24", cfg.AutoWindowHours)
	}
	if cfg.AutoLimitRPM != 5 {
		t.Fatalf("AutoLimitRPM = %d, want 5", cfg.AutoLimitRPM)
	}
	if cfg.AutoLimitDurationMinutes != 60 {
		t.Fatalf("AutoLimitDurationMinutes = %d, want 60", cfg.AutoLimitDurationMinutes)
	}
	if cfg.OutputWindowChars != defaultModerationOutputWindowChars {
		t.Fatalf("OutputWindowChars = %d, want %d", cfg.OutputWindowChars, defaultModerationOutputWindowChars)
	}
	if cfg.ClassifierTemplate == "" {
		t.Fatal("ClassifierTemplate is empty")
	}
}

func TestShouldBlockOnModerationError(t *testing.T) {
	if shouldBlockOnModerationError(moderationFailOpen) {
		t.Fatal("fail_open should not block on moderation errors")
	}
	if !shouldBlockOnModerationError(moderationFailClose) {
		t.Fatal("fail_close should block on moderation errors")
	}
	if shouldBlockOnModerationError("unexpected") {
		t.Fatal("unexpected strategy should normalize to fail_open")
	}
}

func TestParseChatClassifierContent(t *testing.T) {
	result, err := parseChatClassifierContent(`{"flagged":true,"score":0.84,"categories":{"hate":true},"reason":"policy"}`, 0.7, "classifier")
	if err != nil {
		t.Fatalf("parseChatClassifierContent returned error: %v", err)
	}
	if !result.Flagged {
		t.Fatal("Flagged = false, want true")
	}
	if result.Score != 0.84 {
		t.Fatalf("Score = %v, want 0.84", result.Score)
	}
	if result.Reason != "policy" {
		t.Fatalf("Reason = %q, want policy", result.Reason)
	}
	var categories map[string]interface{}
	if err := json.Unmarshal([]byte(result.CategoriesJSON), &categories); err != nil {
		t.Fatalf("CategoriesJSON is invalid JSON: %v", err)
	}
	if categories["categories"] == nil {
		t.Fatalf("CategoriesJSON missing categories: %s", result.CategoriesJSON)
	}
}

func TestBuildChatClassifierRequestAppliesTemplateVariables(t *testing.T) {
	body, err := buildChatClassifierRequest("guard-model", "Review {{DIRECTION}}", "output", "bad content")
	if err != nil {
		t.Fatalf("buildChatClassifierRequest returned error: %v", err)
	}
	var decoded struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("request body is invalid JSON: %v", err)
	}
	if decoded.Model != "guard-model" {
		t.Fatalf("Model = %q", decoded.Model)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(decoded.Messages))
	}
	if decoded.Messages[0].Content != "Review output" {
		t.Fatalf("system content = %q", decoded.Messages[0].Content)
	}
	if decoded.Messages[1].Content != "bad content" {
		t.Fatalf("user content = %q", decoded.Messages[1].Content)
	}
	if decoded.ResponseFormat.Type != "json_object" {
		t.Fatalf("response_format.type = %q", decoded.ResponseFormat.Type)
	}
}

func TestCheckContentForPolicyRecordsDirectionAndBlocks(t *testing.T) {
	moderationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/moderations" {
			t.Fatalf("unexpected moderation path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"flagged":true,"categories":{"violence":true},"category_scores":{"violence":0.88}}]}`))
	}))
	defer moderationServer.Close()

	repo := &fakeModerationDispositionRepo{}
	service := &Service{
		cfg: config.NewRuntime(config.Config{
			ModerationEnabled:   true,
			ModerationBaseURL:   moderationServer.URL,
			ModerationModel:     "guard-model",
			ModerationThreshold: 0.5,
		}),
		repo: repo,
	}

	err := service.CheckContentForPolicy(context.Background(), 7, "unsafe output", "output")

	if err != ErrModerationBlocked {
		t.Fatalf("CheckContentForPolicy() error = %v, want ErrModerationBlocked", err)
	}
	if len(repo.createdEvents) != 1 {
		t.Fatalf("created moderation events = %d, want 1", len(repo.createdEvents))
	}
	event := repo.createdEvents[0]
	if event.Direction != "output" || event.EventType != moderationEventPolicyHit || !event.Flagged {
		t.Fatalf("moderation event = %#v", event)
	}
	if event.ContentSnapshot != "unsafe output" {
		t.Fatalf("ContentSnapshot = %q", event.ContentSnapshot)
	}
}

func TestApplyModerationAutoDispositionSendsLimitNotificationOnce(t *testing.T) {
	service := &Service{
		cfg: config.NewRuntime(config.Config{
			ModerationAutoWindowHours:          24,
			ModerationAutoLimitThreshold:       2,
			ModerationAutoLimitRPM:             3,
			ModerationAutoLimitDurationMinutes: 30,
		}),
		repo:         &fakeModerationDispositionRepo{flaggedCount: 2},
		userEnforcer: &fakeModerationUserEnforcer{user: &domainuser.User{ID: 7, Role: domainuser.RoleUser, Status: domainuser.StatusActive}},
		rateLimiter:  &fakeModerationRateLimiter{},
	}
	notifier := &fakeModerationNotifier{}
	service.SetModerationNotifier(notifier)

	disposition, appliedAt, err := service.applyModerationAutoDisposition(context.Background(), 7)
	if err != nil {
		t.Fatalf("applyModerationAutoDisposition() error = %v", err)
	}
	if disposition != moderationDispositionLimit || appliedAt == nil {
		t.Fatalf("disposition = %q, appliedAt = %v", disposition, appliedAt)
	}
	limiter := service.rateLimiter.(*fakeModerationRateLimiter)
	if limiter.setUserID != 7 || limiter.setRPM != 3 || limiter.setTTL != 30*time.Minute {
		t.Fatalf("rate limit override = user:%d rpm:%d ttl:%s", limiter.setUserID, limiter.setRPM, limiter.setTTL)
	}
	if len(notifier.items) != 1 {
		t.Fatalf("notification count = %d, want 1", len(notifier.items))
	}
	if notifier.items[0].Title != moderationLimitNotificationTitle ||
		notifier.items[0].Source != moderationNotificationSource ||
		notifier.items[0].ActionURL != moderationNotificationLink {
		t.Fatalf("limit notification = %#v", notifier.items[0])
	}

	disposition, appliedAt, err = service.applyModerationAutoDisposition(context.Background(), 7)
	if err != nil {
		t.Fatalf("second applyModerationAutoDisposition() error = %v", err)
	}
	if disposition != moderationDispositionLimit || appliedAt == nil {
		t.Fatalf("second disposition = %q, appliedAt = %v", disposition, appliedAt)
	}
	if len(notifier.items) != 1 {
		t.Fatalf("notification count after existing override = %d, want 1", len(notifier.items))
	}
}

func TestApplyModerationAutoDispositionSendsSuspendNotificationOnce(t *testing.T) {
	enforcer := &fakeModerationUserEnforcer{user: &domainuser.User{ID: 9, Role: domainuser.RoleUser, Status: domainuser.StatusActive}}
	service := &Service{
		cfg: config.NewRuntime(config.Config{
			ModerationAutoWindowHours:      24,
			ModerationAutoSuspendThreshold: 2,
		}),
		repo:         &fakeModerationDispositionRepo{flaggedCount: 2},
		userEnforcer: enforcer,
	}
	notifier := &fakeModerationNotifier{}
	service.SetModerationNotifier(notifier)

	disposition, appliedAt, err := service.applyModerationAutoDisposition(context.Background(), 9)
	if err != nil {
		t.Fatalf("applyModerationAutoDisposition() error = %v", err)
	}
	if disposition != moderationDispositionSuspend || appliedAt == nil {
		t.Fatalf("disposition = %q, appliedAt = %v", disposition, appliedAt)
	}
	if enforcer.user.Status != domainuser.StatusSuspended || enforcer.revokeReason != moderationAutoSuspendRevokeReason {
		t.Fatalf("suspension state = status:%q revoke:%q", enforcer.user.Status, enforcer.revokeReason)
	}
	if len(notifier.items) != 1 || notifier.items[0].Title != moderationSuspendNotificationTitle {
		t.Fatalf("suspend notification = %#v", notifier.items)
	}

	disposition, appliedAt, err = service.applyModerationAutoDisposition(context.Background(), 9)
	if err != nil {
		t.Fatalf("second applyModerationAutoDisposition() error = %v", err)
	}
	if disposition != "" || appliedAt != nil {
		t.Fatalf("second disposition = %q, appliedAt = %v", disposition, appliedAt)
	}
	if len(notifier.items) != 1 {
		t.Fatalf("notification count after existing suspension = %d, want 1", len(notifier.items))
	}
}

func TestReleaseModerationDispositionClearsLimitAndSendsNotification(t *testing.T) {
	repo := &fakeModerationDispositionRepo{event: &model.ModerationEvent{
		ID:          33,
		UserID:      7,
		Disposition: moderationDispositionLimit,
	}}
	limiter := &fakeModerationRateLimiter{overrideExists: true}
	service := &Service{
		repo:        repo,
		rateLimiter: limiter,
	}
	notifier := &fakeModerationNotifier{}
	service.SetModerationNotifier(notifier)

	item, err := service.ReleaseModerationDisposition(context.Background(), 33, 99, "resolved")
	if err != nil {
		t.Fatalf("ReleaseModerationDisposition() error = %v", err)
	}
	if item == nil || item.ReviewStatus != "resolved" || item.DispositionReleasedBy != 99 {
		t.Fatalf("released item = %#v", item)
	}
	if limiter.overrideExists {
		t.Fatal("rate limit override was not cleared")
	}
	if !repo.released || repo.reviewStatus != "resolved" || repo.reviewNote != "resolved" {
		t.Fatalf("release repo state = released:%v review:%q note:%q", repo.released, repo.reviewStatus, repo.reviewNote)
	}
	if len(notifier.items) != 1 {
		t.Fatalf("notification count = %d, want 1", len(notifier.items))
	}
	if notifier.items[0].Title != moderationReleaseNotificationTitle ||
		notifier.items[0].Source != moderationNotificationSource ||
		notifier.items[0].SourceID != "release:33" ||
		notifier.items[0].ActionURL != moderationNotificationLink {
		t.Fatalf("release notification = %#v", notifier.items[0])
	}
}

func TestReleaseModerationDispositionRestoresModerationOwnedSuspensionAndSendsNotification(t *testing.T) {
	repo := &fakeModerationDispositionRepo{event: &model.ModerationEvent{
		ID:          35,
		UserID:      8,
		Disposition: moderationDispositionSuspend,
	}}
	enforcer := &fakeModerationUserEnforcer{user: &domainuser.User{
		ID:               8,
		Role:             domainuser.RoleUser,
		Status:           domainuser.StatusSuspended,
		SuspensionReason: moderationVisibleSuspensionReason,
		SuspensionDetail: moderationAutoSuspendDetail,
	}}
	service := &Service{
		repo:         repo,
		userEnforcer: enforcer,
	}
	notifier := &fakeModerationNotifier{}
	service.SetModerationNotifier(notifier)

	item, err := service.ReleaseModerationDisposition(context.Background(), 35, 99, "resolved")
	if err != nil {
		t.Fatalf("ReleaseModerationDisposition() error = %v", err)
	}
	if item == nil || item.ReviewStatus != "resolved" || item.DispositionReleasedBy != 99 {
		t.Fatalf("released item = %#v", item)
	}
	if enforcer.user.Status != domainuser.StatusActive ||
		enforcer.user.SuspensionReason != "" ||
		enforcer.user.SuspensionDetail != "" ||
		enforcer.user.SuspendedAt != nil ||
		enforcer.user.SuspendedBy != nil {
		t.Fatalf("suspension state = %#v", enforcer.user)
	}
	if len(notifier.items) != 1 || notifier.items[0].SourceID != "release:35" {
		t.Fatalf("release notification = %#v", notifier.items)
	}
}

func TestReleaseModerationDispositionRejectsManualSuspensionWithoutNotification(t *testing.T) {
	repo := &fakeModerationDispositionRepo{event: &model.ModerationEvent{
		ID:          34,
		UserID:      8,
		Disposition: moderationDispositionSuspend,
	}}
	enforcer := &fakeModerationUserEnforcer{user: &domainuser.User{
		ID:               8,
		Role:             domainuser.RoleUser,
		Status:           domainuser.StatusSuspended,
		SuspensionReason: "manual_review",
		SuspensionDetail: "admin_decision",
	}}
	service := &Service{
		repo:         repo,
		userEnforcer: enforcer,
	}
	notifier := &fakeModerationNotifier{}
	service.SetModerationNotifier(notifier)

	_, err := service.ReleaseModerationDisposition(context.Background(), 34, 99, "resolved")
	if err != ErrInvalidMessageContent {
		t.Fatalf("ReleaseModerationDisposition() error = %v, want ErrInvalidMessageContent", err)
	}
	if enforcer.user.Status != domainuser.StatusSuspended || repo.released {
		t.Fatalf("manual suspension changed, user = %#v, released = %v", enforcer.user, repo.released)
	}
	if len(notifier.items) != 0 {
		t.Fatalf("notification count = %d, want 0", len(notifier.items))
	}
}

type fakeModerationDispositionRepo struct {
	repository.ConversationRepository
	flaggedCount  int64
	event         *model.ModerationEvent
	createdEvents []model.ModerationEvent
	released      bool
	reviewStatus  string
	reviewNote    string
}

func (r *fakeModerationDispositionRepo) CountFlaggedModerationEvents(context.Context, uint, time.Time) (int64, error) {
	return r.flaggedCount, nil
}

func (r *fakeModerationDispositionRepo) CreateModerationEvent(_ context.Context, event *model.ModerationEvent) error {
	if event == nil {
		return nil
	}
	if event.ID == 0 {
		event.ID = uint(len(r.createdEvents) + 1)
	}
	copied := *event
	r.createdEvents = append(r.createdEvents, copied)
	return nil
}

func (r *fakeModerationDispositionRepo) GetModerationEvent(_ context.Context, id uint) (*model.ModerationEvent, error) {
	if r.event == nil || r.event.ID != id {
		return nil, repository.ErrNotFound
	}
	item := *r.event
	return &item, nil
}

func (r *fakeModerationDispositionRepo) ReleaseModerationEventDisposition(_ context.Context, id uint, reviewerID uint, releasedAt *time.Time) (*model.ModerationEvent, error) {
	if r.event == nil || r.event.ID != id {
		return nil, repository.ErrNotFound
	}
	r.released = true
	r.event.DispositionReleasedAt = releasedAt
	r.event.DispositionReleasedBy = reviewerID
	item := *r.event
	return &item, nil
}

func (r *fakeModerationDispositionRepo) UpdateModerationEventReview(_ context.Context, id uint, status string, reviewerID uint, reviewedAt *time.Time, note string) (*model.ModerationEvent, error) {
	if r.event == nil || r.event.ID != id {
		return nil, repository.ErrNotFound
	}
	r.reviewStatus = status
	r.reviewNote = note
	r.event.ReviewStatus = status
	r.event.ReviewedBy = reviewerID
	r.event.ReviewedAt = reviewedAt
	r.event.ReviewNote = note
	item := *r.event
	return &item, nil
}

type fakeModerationUserEnforcer struct {
	user         *domainuser.User
	revokeReason string
}

func (e *fakeModerationUserEnforcer) GetByID(context.Context, uint) (*domainuser.User, error) {
	return e.user, nil
}

func (e *fakeModerationUserEnforcer) UpdateUserStatus(_ context.Context, _ uint, status string) error {
	e.user.Status = status
	return nil
}

func (e *fakeModerationUserEnforcer) SetUserSuspension(_ context.Context, _ uint, reason string, detail string, suspendedAt *time.Time, suspendedBy *uint) error {
	e.user.SuspensionReason = reason
	e.user.SuspensionDetail = detail
	e.user.SuspendedAt = suspendedAt
	e.user.SuspendedBy = suspendedBy
	return nil
}

func (e *fakeModerationUserEnforcer) RevokeAllSessions(_ context.Context, _ uint, reason string) error {
	e.revokeReason = reason
	return nil
}

type fakeModerationRateLimiter struct {
	setUserID      uint
	setRPM         int
	setTTL         time.Duration
	overrideExists bool
}

func (l *fakeModerationRateLimiter) SetUserRateLimitOverride(_ context.Context, userID uint, rpm int, ttl time.Duration) error {
	l.setUserID = userID
	l.setRPM = rpm
	l.setTTL = ttl
	l.overrideExists = true
	return nil
}

func (l *fakeModerationRateLimiter) GetUserRateLimitOverride(context.Context, uint) (int, bool, error) {
	return l.setRPM, l.overrideExists, nil
}

func (l *fakeModerationRateLimiter) ClearUserRateLimitOverride(context.Context, uint) error {
	l.overrideExists = false
	return nil
}

type fakeModerationNotifier struct {
	items []appnotification.SystemNotificationInput
}

func (n *fakeModerationNotifier) CreateSystemNotification(_ context.Context, _ uint, input appnotification.SystemNotificationInput) (*appnotification.NotificationView, error) {
	n.items = append(n.items, input)
	return &appnotification.NotificationView{}, nil
}
