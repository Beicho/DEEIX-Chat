package notification

import (
	"context"
	"errors"
	"time"

	domainnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/notification"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
)

// Repo 封装站内通知数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建通知仓储。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// CreateNotification 创建一条站内通知。
func (r *Repo) CreateNotification(ctx context.Context, item *domainnotification.Notification) (*domainnotification.Notification, error) {
	if item == nil || item.UserID == 0 {
		return nil, repository.ErrInvalidInput
	}
	record := model.Notification{
		UserID:   item.UserID,
		Type:     item.Type,
		Title:    item.Title,
		Body:     item.Body,
		Link:     item.Link,
		ReadAt:   item.ReadAt,
		Source:   item.Source,
		SourceID: item.SourceID,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomain(record)
	return &result, nil
}

// ListNotifications 分页查询用户通知。
func (r *Repo) ListNotifications(ctx context.Context, userID uint, filter repository.NotificationListFilter, offset int, limit int) ([]domainnotification.Notification, int64, error) {
	if userID == 0 {
		return nil, 0, repository.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	items := make([]model.Notification, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID)
	if filter.UnreadOnly {
		query = query.Where("read_at IS NULL")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.
		Order("CASE WHEN read_at IS NULL THEN 0 ELSE 1 END ASC, updated_at DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}

	results := make([]domainnotification.Notification, 0, len(items))
	for _, item := range items {
		results = append(results, toDomain(item))
	}
	return results, total, nil
}

// CountUnreadNotifications 统计用户未读持久化通知。
func (r *Repo) CountUnreadNotifications(ctx context.Context, userID uint) (int64, error) {
	if userID == 0 {
		return 0, repository.ErrInvalidInput
	}
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID).
		Where("read_at IS NULL").
		Count(&total).Error; err != nil {
		return 0, translateError(err)
	}
	return total, nil
}

// MarkNotificationRead 标记单条持久化通知已读。
func (r *Repo) MarkNotificationRead(ctx context.Context, userID uint, notificationID uint, now time.Time) error {
	if userID == 0 || notificationID == 0 || now.IsZero() {
		return repository.ErrInvalidInput
	}
	result := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("read_at", now)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// MarkAllNotificationsRead 标记当前用户全部持久化通知已读。
func (r *Repo) MarkAllNotificationsRead(ctx context.Context, userID uint, now time.Time) error {
	if userID == 0 || now.IsZero() {
		return repository.ErrInvalidInput
	}
	if err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID).
		Where("read_at IS NULL").
		Update("read_at", now).Error; err != nil {
		return translateError(err)
	}
	return nil
}

func toDomain(item model.Notification) domainnotification.Notification {
	return domainnotification.Notification{
		ID:        item.ID,
		UserID:    item.UserID,
		Type:      item.Type,
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

func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	return err
}
