package alerting

import (
	"time"

	appchannel "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
)

// ChannelAlertSink 将 channel 模块的熔断事件适配为 alerting 事件并分发。
// 它实现 channel.CircuitAlertSink，从而把 channel 与 alerting 解耦。
type ChannelAlertSink struct {
	service *Service
}

// NewChannelAlertSink 创建 channel→alerting 适配器。
func NewChannelAlertSink(service *Service) *ChannelAlertSink {
	return &ChannelAlertSink{service: service}
}

// EmitCircuitAlert 实现 channel.CircuitAlertSink。
func (s *ChannelAlertSink) EmitCircuitAlert(alert appchannel.CircuitAlert) {
	if s == nil || s.service == nil {
		return
	}
	eventType := EventTypeCircuitOpen
	if !alert.Open {
		eventType = EventTypeCircuitClosed
	}
	s.service.Dispatch(AlertEvent{
		Type:        eventType,
		ChannelID:   int(alert.UpstreamID),
		ChannelName: alert.UpstreamName,
		ModelNames:  alert.ModelNames,
		Timestamp:   time.Now(),
	})
}

var _ appchannel.CircuitAlertSink = (*ChannelAlertSink)(nil)
