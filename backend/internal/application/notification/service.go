package notification

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	appannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/announcement"
	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

const (
	defaultNotificationPageSize = 20
	maxNotificationPageSize     = 100
	maxNotificationTitleLength  = 200
	maxNotificationBodyLength   = 20000
	maxNotificationLinkLength   = 500
	announcementNotificationURL = "/announcements"
	subscriptionNotificationURL = "/setting/subscription"
	notificationWorkerPageSize  = 100
)

// AnnouncementProvider 描述通知中心需要的公告能力。
type AnnouncementProvider interface {
	ListActive(ctx context.Context, userID uint, now time.Time, includeDismissed bool) ([]domainannouncement.Announcement, error)
	Close(ctx context.Context, userID uint, announcementID uint, announcementUpdatedAt time.Time, now time.Time) error
}

// Service 封装站内通知业务逻辑。
type Service struct {
	repo          repository.NotificationRepository
	announcements AnnouncementProvider
	users         UserProvider
	billing       BillingProvider
	logger        *zap.Logger
}

// NewService 创建通知服务。
func NewService(repo repository.NotificationRepository, announcements AnnouncementProvider) *Service {
	return &Service{repo: repo, announcements: announcements}
}

// UserProvider supplies users for account-scoped notification producers.
type UserProvider interface {
	ListUsers(ctx context.Context, page int, pageSize int, filter repository.UserListFilter) ([]domainuser.User, int64, error)
}

// BillingProvider supplies billing data for lifecycle reminders and summaries.
type BillingProvider interface {
	GetCurrentSubscriptionSnapshot(ctx context.Context, userID uint, now time.Time) (*appbilling.UserSubscriptionSnapshot, error)
	ListDailyUsageRange(ctx context.Context, userID uint, startDate time.Time, endDate time.Time) ([]domainbilling.UsageDailySummary, error)
}

// SetUserNotificationProviders enables account and billing lifecycle notification producers.
func (s *Service) SetUserNotificationProviders(users UserProvider, billing BillingProvider) {
	s.users = users
	s.billing = billing
}

// SetLogger injects a logger for background notification producers.
func (s *Service) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// ListInput 定义通知列表入参。
type ListInput struct {
	Page       int
	PageSize   int
	UnreadOnly bool
}

// CreateInput 定义通知创建入参。
type CreateInput struct {
	Type      string
	Title     string
	Body      string
	ActionURL string
	Link      string
	Source    string
	SourceID  string
	Metadata  map[string]any
}

// SystemNotificationInput 定义系统通知生产者入参。
type SystemNotificationInput struct {
	Type      string
	Title     string
	Body      string
	ActionURL string
	Source    string
	SourceID  string
	Metadata  map[string]any
}

// NotificationView 是通知中心统一输出视图，可包含持久化通知或虚拟通知。
type NotificationView struct {
	ID        string
	Type      string
	Title     string
	Body      string
	ActionURL string
	Link      string
	ReadAt    *time.Time
	Source    string
	SourceID  string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Create 创建一条持久化站内通知，供后续生产者复用。
func (s *Service) Create(ctx context.Context, userID uint, input CreateInput) (*NotificationView, error) {
	if userID == 0 {
		return nil, repository.ErrInvalidInput
	}
	item, err := normalizeCreateInput(userID, input)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreateNotification(ctx, item)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	view := viewFromNotification(*created)
	return &view, nil
}

// NotifyUser 创建一条用户站内通知，供 C4/C9/C15 等生产者稳定复用。
func (s *Service) NotifyUser(ctx context.Context, userID uint, input CreateInput) (*NotificationView, error) {
	return s.Create(ctx, userID, input)
}

// CreateSystemNotification 创建一条系统通知。
func (s *Service) CreateSystemNotification(ctx context.Context, userID uint, input SystemNotificationInput) (*NotificationView, error) {
	return s.NotifyUser(ctx, userID, CreateInput{
		Type:      systemNotificationType(input.Type),
		Title:     input.Title,
		Body:      input.Body,
		ActionURL: input.ActionURL,
		Source:    input.Source,
		SourceID:  input.SourceID,
		Metadata:  input.Metadata,
	})
}

func systemNotificationType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return domainnotification.TypeSystem
	}
	return trimmed
}

// CreateSystemNotificationOnce creates a system notification once per user/source/sourceID key.
func (s *Service) CreateSystemNotificationOnce(ctx context.Context, userID uint, input SystemNotificationInput) (*NotificationView, bool, error) {
	source := strings.TrimSpace(input.Source)
	sourceID := strings.TrimSpace(input.SourceID)
	if userID == 0 || source == "" || sourceID == "" {
		return nil, false, repository.ErrInvalidInput
	}
	existing, err := s.repo.GetNotificationBySource(ctx, userID, source, sourceID)
	if err == nil && existing != nil {
		view := viewFromNotification(*existing)
		return &view, false, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, false, mapRepositoryError(err)
	}
	created, err := s.CreateSystemNotification(ctx, userID, input)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

// List 查询用户通知，并把当前 active 公告作为虚拟通知合入结果。
func (s *Service) List(ctx context.Context, userID uint, input ListInput, now time.Time) ([]NotificationView, int64, error) {
	if userID == 0 {
		return nil, 0, repository.ErrInvalidInput
	}
	page, pageSize := normalizePage(input.Page, input.PageSize)
	offset := (page - 1) * pageSize
	persistedLimit := offset + pageSize

	persisted, persistedTotal, err := s.repo.ListNotifications(ctx, userID, repository.NotificationListFilter{UnreadOnly: input.UnreadOnly}, 0, persistedLimit)
	if err != nil {
		return nil, 0, mapRepositoryError(err)
	}

	items := make([]NotificationView, 0, len(persisted))
	for _, item := range persisted {
		items = append(items, viewFromNotification(item))
	}

	announcementViews, err := s.listAnnouncementViews(ctx, userID, now, input.UnreadOnly)
	if err != nil {
		return nil, 0, err
	}
	items = append(items, announcementViews...)
	sortNotificationViews(items)

	total := persistedTotal + int64(len(announcementViews))
	if offset >= len(items) {
		return []NotificationView{}, total, nil
	}
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], total, nil
}

// UnreadCount 统计未读通知数量，包含公告虚拟通知。
func (s *Service) UnreadCount(ctx context.Context, userID uint, now time.Time) (int64, error) {
	if userID == 0 {
		return 0, repository.ErrInvalidInput
	}
	count, err := s.repo.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return 0, mapRepositoryError(err)
	}
	announcements, err := s.listAnnouncementViews(ctx, userID, now, true)
	if err != nil {
		return 0, err
	}
	return count + int64(len(announcements)), nil
}

// MarkRead 标记单条通知已读。
func (s *Service) MarkRead(ctx context.Context, userID uint, id string, now time.Time) error {
	if userID == 0 || strings.TrimSpace(id) == "" || now.IsZero() {
		return repository.ErrInvalidInput
	}
	if announcementID, updatedAt, ok := parseAnnouncementNotificationID(id); ok {
		if s.announcements == nil {
			return ErrNotificationNotFound
		}
		return mapAnnouncementError(s.announcements.Close(ctx, userID, announcementID, updatedAt, now))
	}

	notificationID, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil || notificationID == 0 {
		return repository.ErrInvalidInput
	}
	return mapRepositoryError(s.repo.MarkNotificationRead(ctx, userID, uint(notificationID), now))
}

// MarkAllRead 标记当前用户全部通知已读。
func (s *Service) MarkAllRead(ctx context.Context, userID uint, now time.Time) error {
	if userID == 0 || now.IsZero() {
		return repository.ErrInvalidInput
	}
	if err := s.repo.MarkAllNotificationsRead(ctx, userID, now); err != nil {
		return mapRepositoryError(err)
	}
	if s.announcements == nil {
		return nil
	}
	items, err := s.announcements.ListActive(ctx, userID, now, true)
	if err != nil {
		return mapAnnouncementError(err)
	}
	for _, item := range items {
		if item.ClosedAt != nil {
			continue
		}
		if err := s.announcements.Close(ctx, userID, item.ID, item.UpdatedAt, now); err != nil {
			return mapAnnouncementError(err)
		}
	}
	return nil
}

// StartLifecycleNotificationWorker scans users and emits lifecycle reminders.
func (s *Service) StartLifecycleNotificationWorker(ctx context.Context) {
	if s == nil || s.users == nil || s.billing == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		s.runLifecycleNotificationScan(ctx, time.Now())
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runLifecycleNotificationScan(ctx, time.Now())
			}
		}
	}()
}

func (s *Service) runLifecycleNotificationScan(ctx context.Context, now time.Time) {
	if s == nil || s.users == nil || s.billing == nil {
		return
	}
	page := 1
	for {
		items, total, err := s.users.ListUsers(ctx, page, notificationWorkerPageSize, repository.UserListFilter{})
		if err != nil {
			s.logLifecycleNotificationError("list_users", err)
			return
		}
		if len(items) == 0 {
			return
		}
		for _, user := range items {
			if domainuser.IsAdminRole(user.Role) || user.Status != domainuser.StatusActive {
				continue
			}
			s.sendSubscriptionExpiryNotification(ctx, user, now)
			s.sendWeeklyUsageSummary(ctx, user, now)
		}
		if int64(page*notificationWorkerPageSize) >= total {
			return
		}
		page++
	}
}

func (s *Service) sendSubscriptionExpiryNotification(ctx context.Context, user domainuser.User, now time.Time) {
	snapshot, err := s.billing.GetCurrentSubscriptionSnapshot(ctx, user.ID, now)
	if err != nil || snapshot == nil || snapshot.ExpiresAt == nil {
		return
	}
	remaining := snapshot.ExpiresAt.Sub(now)
	if remaining <= 0 || remaining > 72*time.Hour {
		return
	}
	dayKey := snapshot.ExpiresAt.UTC().Format("2006-01-02")
	input := SystemNotificationInput{
		Type:      domainnotification.TypeBillingExpiry,
		Title:     "订阅即将到期",
		Body:      "你的当前订阅将在 " + snapshot.ExpiresAt.UTC().Format("2006-01-02 15:04") + " 到期，请及时续费。",
		ActionURL: subscriptionNotificationURL,
		Source:    "billing_expiry",
		SourceID:  "billing_expiry:" + dayKey,
		Metadata: map[string]any{
			"plan_name": snapshot.PlanName,
			"tier":      snapshot.Tier,
			"expiresAt": snapshot.ExpiresAt.UTC().Format(time.RFC3339),
		},
	}
	_, _, _ = s.CreateSystemNotificationOnce(ctx, user.ID, input)
}

func (s *Service) sendWeeklyUsageSummary(ctx context.Context, user domainuser.User, now time.Time) {
	if now.Weekday() != time.Monday {
		return
	}
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startDate := endDate.AddDate(0, 0, -7)
	items, err := s.billing.ListDailyUsageRange(ctx, user.ID, startDate, endDate)
	if err != nil {
		return
	}
	var recordCount, callCount, billedNanousd int64
	for _, item := range items {
		recordCount += item.RecordCount
		callCount += item.CallCount
		billedNanousd += item.BilledNanousd
	}
	if recordCount == 0 && callCount == 0 && billedNanousd == 0 {
		return
	}
	year, week := startDate.ISOWeek()
	input := SystemNotificationInput{
		Type:      domainnotification.TypeWeeklySummary,
		Title:     "上周使用摘要",
		Body:      "你上周共使用 " + itoa64(callCount) + " 次，对话记录 " + itoa64(recordCount) + " 条，消耗 " + formatUSD(billedNanousd) + "。",
		ActionURL: subscriptionNotificationURL,
		Source:    "weekly_summary",
		SourceID:  "weekly_summary:" + strconv.Itoa(year) + "-W" + pad2(week),
		Metadata: map[string]any{
			"week_start":     startDate.Format("2006-01-02"),
			"week_end":       endDate.Format("2006-01-02"),
			"record_count":   recordCount,
			"call_count":     callCount,
			"billed_nanousd": billedNanousd,
		},
	}
	_, _, _ = s.CreateSystemNotificationOnce(ctx, user.ID, input)
}

func (s *Service) logLifecycleNotificationError(scope string, err error) {
	if s == nil || s.logger == nil || err == nil {
		return
	}
	s.logger.Warn("notification_lifecycle_scan_failed", zap.String("scope", scope), zap.Error(err))
}

func itoa64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func pad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func formatUSD(nanousd int64) string {
	sign := ""
	if nanousd < 0 {
		sign = "-"
		nanousd = -nanousd
	}
	cents := (nanousd + 5_000_000) / 10_000_000
	return sign + "$" + strconv.FormatInt(cents/100, 10) + "." + pad2(int(cents%100))
}

func (s *Service) listAnnouncementViews(ctx context.Context, userID uint, now time.Time, unreadOnly bool) ([]NotificationView, error) {
	if s.announcements == nil {
		return []NotificationView{}, nil
	}
	items, err := s.announcements.ListActive(ctx, userID, now, true)
	if err != nil {
		return nil, mapAnnouncementError(err)
	}
	results := make([]NotificationView, 0, len(items))
	for _, item := range items {
		if unreadOnly && item.ClosedAt != nil {
			continue
		}
		results = append(results, viewFromAnnouncement(item))
	}
	return results, nil
}

func viewFromNotification(item domainnotification.Notification) NotificationView {
	return NotificationView{
		ID:        strconv.FormatUint(uint64(item.ID), 10),
		Type:      normalizeNotificationType(item.Type),
		Title:     item.Title,
		Body:      item.Body,
		ActionURL: item.ActionURL,
		Link:      item.Link,
		ReadAt:    item.ReadAt,
		Source:    item.Source,
		SourceID:  item.SourceID,
		Metadata:  cloneMetadata(item.Metadata),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func viewFromAnnouncement(item domainannouncement.Announcement) NotificationView {
	return NotificationView{
		ID:        announcementNotificationID(item.ID, item.UpdatedAt),
		Type:      domainnotification.TypeAnnouncement,
		Title:     item.Title,
		Body:      item.ContentMarkdown,
		ActionURL: announcementNotificationURL,
		Link:      announcementNotificationURL,
		ReadAt:    item.ClosedAt,
		Source:    domainnotification.SourceAnnouncement,
		SourceID:  strconv.FormatUint(uint64(item.ID), 10),
		Metadata: map[string]any{
			"announcementId": item.ID,
		},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func announcementNotificationID(announcementID uint, updatedAt time.Time) string {
	return fmt.Sprintf("%s:%d:%d", domainnotification.SourceAnnouncement, announcementID, updatedAt.UnixNano())
}

func parseAnnouncementNotificationID(id string) (uint, time.Time, bool) {
	trimmed := strings.TrimSpace(id)
	if decoded, err := url.PathUnescape(trimmed); err == nil {
		trimmed = decoded
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) != 3 || parts[0] != domainnotification.SourceAnnouncement {
		return 0, time.Time{}, false
	}
	announcementID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || announcementID == 0 {
		return 0, time.Time{}, false
	}
	nanos, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || nanos <= 0 {
		return 0, time.Time{}, false
	}
	return uint(announcementID), time.Unix(0, nanos).UTC(), true
}

func normalizeNotificationType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return domainnotification.TypeSystem
	}
	return trimmed
}

func normalizeCreateInput(userID uint, input CreateInput) (*domainnotification.Notification, error) {
	notificationType := normalizeNotificationType(input.Type)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	link := strings.TrimSpace(input.ActionURL)
	if link == "" {
		link = strings.TrimSpace(input.Link)
	}
	source := strings.TrimSpace(input.Source)
	sourceID := strings.TrimSpace(input.SourceID)
	if title == "" || len(title) > maxNotificationTitleLength {
		return nil, ErrInvalidNotification
	}
	if len(body) > maxNotificationBodyLength || len(link) > maxNotificationLinkLength {
		return nil, ErrInvalidNotification
	}
	return &domainnotification.Notification{
		UserID:    userID,
		Type:      notificationType,
		Title:     title,
		Body:      body,
		ActionURL: link,
		Link:      link,
		Source:    source,
		SourceID:  sourceID,
		Metadata:  cloneMetadata(input.Metadata),
	}, nil
}

func cloneMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func sortNotificationViews(items []NotificationView) {
	sort.SliceStable(items, func(i int, j int) bool {
		left := items[i]
		right := items[j]
		if (left.ReadAt == nil) != (right.ReadAt == nil) {
			return left.ReadAt == nil
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.ID > right.ID
	})
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultNotificationPageSize
	}
	if pageSize > maxNotificationPageSize {
		pageSize = maxNotificationPageSize
	}
	return page, pageSize
}

func mapRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotificationNotFound
	}
	return err
}

func mapAnnouncementError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, appannouncement.ErrAnnouncementNotFound) {
		return ErrNotificationNotFound
	}
	return err
}
