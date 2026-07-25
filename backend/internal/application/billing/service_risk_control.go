package billing

import (
	"context"
	"errors"
	"strings"
	"time"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// RiskControlConfig 描述调用前风控准入所需的运行时配置。
type RiskControlConfig struct {
	CostGuardEnabled        bool
	NewUserCooldownHours    int
	NewUserCooldownCostUSD  float64
	DailySpendLimitUSD      float64
	AutoSuspendDebtUSD      float64
	SpendAlertSingleCallUSD float64
	SpendAlertDailyUserUSD  float64
}

type riskConfigProvider func() RiskControlConfig

// SetRiskConfigProvider 注入风控运行时配置读取器。
func (s *Service) SetRiskConfigProvider(provider func() RiskControlConfig) {
	if s == nil {
		return
	}
	s.riskConfig = riskConfigProvider(provider)
}

func (s *Service) riskControlConfig() RiskControlConfig {
	if s == nil || s.riskConfig == nil {
		return RiskControlConfig{}
	}
	return s.riskConfig()
}

// EstimateCallCostNanousd 估算一次调用的下限成本，用于调用前的资损准入。
//
// 按次/按时长计费的模型（图片、视频）可以精确预估，必须严格拦截；
// token 计费模型无法在调用前知道输出长度，只用输入侧成本做保守下限，
// 避免误杀正常聊天。
func EstimateCallCostNanousd(pricing *domainbilling.ModelPricing, maxInputTokens int64) int64 {
	if pricing == nil || pricing.IsFree {
		return 0
	}
	switch strings.TrimSpace(pricing.PricingMode) {
	case domainbilling.PricingModeCall:
		if pricing.CallNanousdPerCall > 0 {
			return pricing.CallNanousdPerCall
		}
		return 0
	case domainbilling.PricingModeDuration:
		if pricing.DurationNanousdPerSecond > 0 {
			// 缺少时长入参时按 5 秒这一常见最短片段保守预估。
			return pricing.DurationNanousdPerSecond * defaultEstimatedDurationSeconds
		}
		return 0
	default:
		if maxInputTokens <= 0 {
			return 0
		}
		perMTokens := pricing.InputNanousdPerMTokens
		if perMTokens <= 0 {
			return 0
		}
		return perMTokens * maxInputTokens / 1_000_000
	}
}

const defaultEstimatedDurationSeconds = 5

// IsHighCostPricing 判断该模型是否属于可精确预估的高价（按次/按时长）计费。
func IsHighCostPricing(pricing *domainbilling.ModelPricing) bool {
	if pricing == nil || pricing.IsFree {
		return false
	}
	mode := strings.TrimSpace(pricing.PricingMode)
	return mode == domainbilling.PricingModeCall || mode == domainbilling.PricingModeDuration
}

// EstimateModelCallCostNanousd 对外暴露的单模型成本预估。
func (s *Service) EstimateModelCallCostNanousd(ctx context.Context, platformModelName string) (int64, error) {
	pricing, err := s.getResolvedModelPricing(ctx, platformModelName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return EstimateCallCostNanousd(pricing, 0), nil
}

// ensureRiskControlAccess 在计费准入之后执行资损风控校验。
func (s *Service) ensureRiskControlAccess(
	ctx context.Context,
	userID uint,
	pricing *domainbilling.ModelPricing,
	availableNanousd int64,
	now time.Time,
) error {
	if pricing == nil || pricing.IsFree {
		return nil
	}
	cfg := s.riskControlConfig()
	estimated := EstimateCallCostNanousd(pricing, 0)

	if err := s.ensureNewUserCooldown(ctx, userID, pricing, estimated, cfg, now); err != nil {
		return err
	}
	if err := s.ensureDailySpendLimit(ctx, userID, estimated, cfg, now); err != nil {
		return err
	}
	if !cfg.CostGuardEnabled || estimated <= 0 {
		return nil
	}
	// token 计费模型的预估只是输入侧下限，不足以作为硬闸；
	// 只有可精确预估的按次/按时长模型才严格要求覆盖全额。
	if !IsHighCostPricing(pricing) {
		return nil
	}
	if availableNanousd < estimated {
		return ErrCallCostNotCovered
	}
	return nil
}

func (s *Service) ensureNewUserCooldown(
	ctx context.Context,
	userID uint,
	pricing *domainbilling.ModelPricing,
	estimatedNanousd int64,
	cfg RiskControlConfig,
	now time.Time,
) error {
	if cfg.NewUserCooldownHours <= 0 || !IsHighCostPricing(pricing) {
		return nil
	}
	threshold := usdToNanousd(cfg.NewUserCooldownCostUSD)
	if threshold <= 0 || estimatedNanousd < threshold {
		return nil
	}
	createdAt, err := s.repo.GetUserCreatedAt(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if createdAt.IsZero() {
		return nil
	}
	if now.Sub(createdAt) < time.Duration(cfg.NewUserCooldownHours)*time.Hour {
		return ErrNewUserCooldown
	}
	return nil
}

func (s *Service) ensureDailySpendLimit(
	ctx context.Context,
	userID uint,
	estimatedNanousd int64,
	cfg RiskControlConfig,
	now time.Time,
) error {
	limit := usdToNanousd(cfg.DailySpendLimitUSD)
	if limit <= 0 {
		return nil
	}
	dayStart := now.UTC().Truncate(24 * time.Hour)
	spent, err := s.repo.SumBillableNanousd(ctx, userID, dayStart, dayStart.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	if spent+estimatedNanousd > limit {
		return ErrDailySpendLimitExceeded
	}
	return nil
}

// availableSpendNanousd 计算用户当前可用于覆盖本次调用的额度总量。
func (s *Service) availableSpendNanousd(
	ctx context.Context,
	userID uint,
	mode string,
	now time.Time,
) (int64, error) {
	account, err := s.repo.GetOrCreateBillingAccount(ctx, userID)
	if err != nil {
		return 0, err
	}
	available := account.BalanceNanousd
	if mode != "period" {
		return available, nil
	}
	plan, startAt, endAt, planErr := s.currentPeriodPlan(ctx, userID, now)
	if planErr != nil {
		return 0, planErr
	}
	if plan.PeriodCreditNanousd > 0 {
		usedNanousd, usedErr := s.repo.SumBillableNanousd(ctx, userID, startAt, endAt)
		if usedErr != nil {
			return 0, usedErr
		}
		remaining := plan.PeriodCreditNanousd - usedNanousd
		if remaining > 0 {
			available += remaining
		}
	}
	return available, nil
}

// EnforceDebtSuspension 在结算产生欠费后按阈值自动停用账号。
func (s *Service) EnforceDebtSuspension(ctx context.Context, userID uint) (bool, int64, error) {
	cfg := s.riskControlConfig()
	threshold := usdToNanousd(cfg.AutoSuspendDebtUSD)
	if threshold <= 0 {
		return false, 0, nil
	}
	account, err := s.repo.GetOrCreateBillingAccount(ctx, userID)
	if err != nil {
		return false, 0, err
	}
	if account.BalanceNanousd >= 0 {
		return false, account.BalanceNanousd, nil
	}
	if -account.BalanceNanousd < threshold {
		return false, account.BalanceNanousd, nil
	}
	return true, account.BalanceNanousd, nil
}

// SettlementRiskEvent 描述一次结算后的风控观测结果。
type SettlementRiskEvent struct {
	UserID            uint
	PlatformModelName string
	BilledNanousd     int64
	BalanceNanousd    int64
	DailySpendNanousd int64
	// ShouldSuspend 表示欠费已超过自动停用阈值。
	ShouldSuspend bool
	// SingleCallAlert 表示单次调用金额超过告警阈值。
	SingleCallAlert bool
	// DailySpendAlert 表示该用户当日消费超过告警阈值。
	DailySpendAlert bool
}

// NeedsAttention 表示该事件需要通知或处置。
func (e SettlementRiskEvent) NeedsAttention() bool {
	return e.ShouldSuspend || e.SingleCallAlert || e.DailySpendAlert
}

type settlementRiskHook func(ctx context.Context, event SettlementRiskEvent)

// SetSettlementRiskHook 注入结算后风控回调（告警 / 自动停用）。
func (s *Service) SetSettlementRiskHook(hook func(ctx context.Context, event SettlementRiskEvent)) {
	if s == nil {
		return
	}
	s.settlementRiskHook = settlementRiskHook(hook)
}

// evaluateSettlementRisk 在结算成功后评估风险并触发回调。
// 评估失败不影响主链路，仅放弃本次告警。
func (s *Service) evaluateSettlementRisk(ctx context.Context, usage *domainbilling.UsageLedger) {
	if s == nil || s.settlementRiskHook == nil || usage == nil || usage.UserID == 0 {
		return
	}
	cfg := s.riskControlConfig()
	event := SettlementRiskEvent{
		UserID:            usage.UserID,
		PlatformModelName: usage.PlatformModelName,
		BilledNanousd:     usage.BilledNanousd,
	}

	if threshold := usdToNanousd(cfg.SpendAlertSingleCallUSD); threshold > 0 && usage.BilledNanousd >= threshold {
		event.SingleCallAlert = true
	}
	if threshold := usdToNanousd(cfg.SpendAlertDailyUserUSD); threshold > 0 {
		dayStart := time.Now().UTC().Truncate(24 * time.Hour)
		spent, err := s.repo.SumBillableNanousd(ctx, usage.UserID, dayStart, dayStart.AddDate(0, 0, 1))
		if err == nil {
			event.DailySpendNanousd = spent
			if spent >= threshold {
				event.DailySpendAlert = true
			}
		}
	}
	shouldSuspend, balance, err := s.EnforceDebtSuspension(ctx, usage.UserID)
	if err == nil {
		event.BalanceNanousd = balance
		event.ShouldSuspend = shouldSuspend
	}
	if !event.NeedsAttention() {
		return
	}
	s.settlementRiskHook(ctx, event)
}
