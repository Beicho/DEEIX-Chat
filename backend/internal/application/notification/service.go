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
	domainannouncement "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/announcement"
	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

const (
	defaultNotificationPageSize = 20
	maxNotificationPageSize     = 100
	maxNotificationTitleLength  = 200
	maxNotificationBodyLength   = 20000
	maxNotificationLinkLength   = 500
	announcementNotificationURL = "/announcements"
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
}

// NewService 创建通知服务。
func NewService(repo repository.NotificationRepository, announcements AnnouncementProvider) *Service {
	return &Service{repo: repo, announcements: announcements}
}

// ListInput 定义通知列表入参。
type ListInput struct {
	Page       int
	PageSize   int
	UnreadOnly bool
}

// CreateInput 定义通知创建入参。
type CreateInput struct {
	Type     string
	Title    string
	Body     string
	Link     string
	Source   string
	SourceID string
}

// NotificationView 是通知中心统一输出视图，可包含持久化通知或虚拟通知。
type NotificationView struct {
	ID        string
	Type      string
	Title     string
	Body      string
	Link      string
	ReadAt    *time.Time
	Source    string
	SourceID  string
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
		Link:      item.Link,
		ReadAt:    item.ReadAt,
		Source:    item.Source,
		SourceID:  item.SourceID,
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
		Link:      announcementNotificationURL,
		ReadAt:    item.ClosedAt,
		Source:    domainnotification.SourceAnnouncement,
		SourceID:  strconv.FormatUint(uint64(item.ID), 10),
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
		return "system"
	}
	return trimmed
}

func normalizeCreateInput(userID uint, input CreateInput) (*domainnotification.Notification, error) {
	notificationType := normalizeNotificationType(input.Type)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	link := strings.TrimSpace(input.Link)
	source := strings.TrimSpace(input.Source)
	sourceID := strings.TrimSpace(input.SourceID)
	if title == "" || len(title) > maxNotificationTitleLength {
		return nil, ErrInvalidNotification
	}
	if len(body) > maxNotificationBodyLength || len(link) > maxNotificationLinkLength {
		return nil, ErrInvalidNotification
	}
	return &domainnotification.Notification{
		UserID:   userID,
		Type:     notificationType,
		Title:    title,
		Body:     body,
		Link:     link,
		Source:   source,
		SourceID: sourceID,
	}, nil
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
