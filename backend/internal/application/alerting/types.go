// Package alerting 实现渠道熔断/恢复告警通知（Telegram + Webhook），含去抖。
//
// 设计要点：
//   - 配置存储在 system_settings 表，namespace=alerting（敏感项 AES-GCM 加密）。
//   - 告警事件由 channel 熔断状态变化驱动（CLOSED→OPEN / OPEN→CLOSED）。
//   - 同一渠道（按 dedupeKey）默认 5 分钟去抖，避免抖动期间刷屏。
//   - 敏感信息（telegram token / webhook 地址）不写入日志，也不在 API 响应中明文回显。
package alerting

import (
	"context"
	"time"
)

// 告警事件类型。
const (
	EventTypeCircuitOpen   = "circuit_open"
	EventTypeCircuitClosed = "circuit_closed"
)

// 告警通道标识。
const (
	NotifierTelegram = "telegram"
	NotifierWebhook  = "webhook"
)

// AlertEvent 描述一次渠道熔断/恢复告警。
type AlertEvent struct {
	// Type 取值 circuit_open / circuit_closed。
	Type string
	// ChannelID 上游（渠道）ID。
	ChannelID int
	// ChannelName 上游（渠道）展示名。
	ChannelName string
	// ModelNames 受影响的平台模型名（用户可见名）。
	ModelNames []string
	// Timestamp 事件时间。
	Timestamp time.Time
	// Message 人类可读的告警文案。
	Message string
}

// Notifier 是单个告警通道的发送抽象。
type Notifier interface {
	// Notify 发送一次告警；实现需保证不会泄露敏感配置。
	Notify(ctx context.Context, event AlertEvent) error
	// Kind 返回通道标识（telegram / webhook），用于配置开关匹配与日志。
	Kind() string
}
