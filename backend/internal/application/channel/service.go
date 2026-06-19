package channel

import (
	"context"
	"sync"
	"time"

	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

type billingModelPricingFilter interface {
	GetBillingMode(ctx context.Context) (string, error)
	ListPublicModelPricing(ctx context.Context) (map[string]appbilling.PublicModelPricing, error)
}

// CircuitAlert 描述一次渠道熔断/恢复事件，供告警 sink 消费。
// 该结构与具体告警实现解耦，channel 模块只负责产生事件。
type CircuitAlert struct {
	// Open 为 true 表示熔断打开（CLOSED→OPEN）；false 表示恢复（OPEN→CLOSED）。
	Open bool
	// UpstreamID 上游（渠道）ID。
	UpstreamID uint
	// UpstreamName 上游（渠道）展示名。
	UpstreamName string
	// ModelNames 受影响的平台模型名（用户可见名）。
	ModelNames []string
}

// CircuitAlertSink 接收 channel 模块产生的熔断/恢复事件。
type CircuitAlertSink interface {
	EmitCircuitAlert(alert CircuitAlert)
}

// Service 封装上游、平台模型与路由绑定业务能力。
type Service struct {
	cfg                      *config.Runtime
	repo                     repository.ChannelRepository
	cache                    repository.ChannelCacheRepository
	llmClient                *llm.Client
	modelPricingFilter       billingModelPricingFilter
	modelAnnouncementService modelAnnouncementService
	alertSink                CircuitAlertSink
	logger                   *zap.Logger

	modelCatalogMu         sync.RWMutex
	modelCatalog           []ModelView
	modelCatalogValidUntil time.Time
}

func (s *Service) llmAttribution() (string, string) {
	if s == nil || s.cfg == nil {
		return "", ""
	}
	cfg := s.cfg.Snapshot()
	return cfg.PublicWebBaseURL, cfg.AppName
}

// ResolvedRoute 模型请求路由结果。
type ResolvedRoute struct {
	RouteID                    uint
	PlatformModelID            uint
	PlatformModelName          string
	UpstreamModelID            uint
	UpstreamID                 uint
	UpstreamName               string
	BindingCode                string
	Protocol                   string
	BaseURL                    string
	APIKey                     string
	ConnectTimeoutMS           int
	ReadTimeoutMS              int
	StreamIdleTimeoutMS        int
	HeadersJSON                string
	ModelVendor                string
	ModelIcon                  string
	ModelCapabilitiesJSON      string
	ModelSystemPrompt          string
	UpstreamModel              string
	ReasoningContentPassback   bool
	UpstreamCbFailureThreshold int
	UpstreamCbModelThreshold   int
	UpstreamCbThresholdLogic   string
	UpstreamCbDurationMin      int
	UpstreamCbWindowMin        int
	ModelCbFailureThreshold    int
	ModelCbDurationMin         int
	ModelCbWindowMin           int
	UpstreamProbeGranted       bool
	ModelProbeGranted          bool
}

// ResolveRouteInput 路由解析输入。
type ResolveRouteInput struct {
	PlatformModelName string
	TaskType          string
	Scope             string
	UserID            uint
	ConversationID    uint
	RequestID         string
}

type routeCandidate struct {
	row    repository.ChannelUpstreamRouteRow
	apiKey string
}

type routeFailureClass string

const (
	routeFailureCircuit   routeFailureClass = "circuit"
	routeFailureRateLimit routeFailureClass = "rate_limit"
	routeFailureIgnore    routeFailureClass = "ignore"
	circuitProbeTTLSec                      = 30
	modelCatalogCacheTTL                    = 30 * time.Second
)

const (
	ModelAccessScopePublic   = "public"
	ModelAccessScopeInternal = "internal"

	RouteScopeUser     = "user"
	RouteScopeInternal = "internal"
)

// localAPIKeyCounters 存储各上游的本地 round-robin 计数器（Redis 不可用时的降级实现）。
var localAPIKeyCounters sync.Map

// NewService 创建服务。
func NewService(cfg config.Config, repo repository.ChannelRepository, cache repository.ChannelCacheRepository, llmClient *llm.Client) *Service {
	return NewServiceWithRuntime(config.NewRuntime(cfg), repo, cache, llmClient)
}

// NewServiceWithRuntime 创建使用运行时配置容器的服务。
func NewServiceWithRuntime(cfg *config.Runtime, repo repository.ChannelRepository, cache repository.ChannelCacheRepository, llmClient *llm.Client) *Service {
	return &Service{
		cfg:       cfg,
		repo:      repo,
		cache:     cache,
		llmClient: llmClient,
	}
}

// SetBillingModelPricingFilter 注入计费模型过滤器，用于用户侧模型选择列表。
func (s *Service) SetBillingModelPricingFilter(filter billingModelPricingFilter) {
	s.modelPricingFilter = filter
}

// SetLogger 注入结构化日志记录器。
func (s *Service) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// SetCircuitAlertSink 注入熔断告警接收器，用于在熔断/恢复时发送告警。
func (s *Service) SetCircuitAlertSink(sink CircuitAlertSink) {
	s.alertSink = sink
}

func (s *Service) warn(message string, fields ...zap.Field) {
	if s.logger == nil {
		return
	}
	s.logger.Warn(message, fields...)
}

// InvalidateModelCatalog 清除用户侧模型目录缓存。
func (s *Service) InvalidateModelCatalog() {
	if s == nil {
		return
	}
	s.modelCatalogMu.Lock()
	s.modelCatalog = nil
	s.modelCatalogValidUntil = time.Time{}
	s.modelCatalogMu.Unlock()
}
