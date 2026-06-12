package notification

import (
	"time"

	appnotification "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/notification"
)

// ErrorDoc 用于 Swagger 标注通用错误响应。
type ErrorDoc struct {
	ErrorMsg  string      `json:"errorMsg" example:"invalid request"`
	ErrorCode string      `json:"errorCode,omitempty" example:"invalid_request"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"requestId,omitempty" example:""`
	Data      interface{} `json:"data"`
}

// NotificationResponse 面向前端的通知响应。
type NotificationResponse struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	ReadAt    *time.Time `json:"readAt"`
	Source    string     `json:"source"`
	SourceID  string     `json:"sourceId"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// NotificationListResponseDoc 通知分页响应文档。
type NotificationListResponseDoc struct {
	ErrorMsg string `json:"errorMsg"`
	Data     struct {
		Total   int64                  `json:"total"`
		Results []NotificationResponse `json:"results"`
	} `json:"data"`
}

// UnreadCountDataResponse 未读数量响应。
type UnreadCountDataResponse struct {
	UnreadCount int64 `json:"unreadCount"`
}

// UnreadCountResponseDoc 未读数量响应文档。
type UnreadCountResponseDoc struct {
	ErrorMsg string                  `json:"errorMsg"`
	Data     UnreadCountDataResponse `json:"data"`
}

// NotificationReadDataResponse 已读操作响应。
type NotificationReadDataResponse struct {
	Read bool `json:"read"`
}

// NotificationReadResponseDoc 已读操作响应文档。
type NotificationReadResponseDoc struct {
	ErrorMsg string                       `json:"errorMsg"`
	Data     NotificationReadDataResponse `json:"data"`
}

func toNotificationResponse(item appnotification.NotificationView) NotificationResponse {
	return NotificationResponse{
		ID:        item.ID,
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

func toNotificationResponses(items []appnotification.NotificationView) []NotificationResponse {
	results := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		results = append(results, toNotificationResponse(item))
	}
	return results
}
