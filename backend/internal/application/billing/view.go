package billing

import (
	"time"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
)

// BillingPriceView 表示套餐价格结果。
type BillingPriceView struct {
	ID              uint
	PlanID          uint
	Code            string
	BillingInterval string
	Currency        string
	AmountCents     int64
	IsDefault       bool
}

// BillingPlanView 表示套餐结果。
type BillingPlanView struct {
	ID                  uint
	Code                string
	Name                string
	Description         string
	FeatureJSON         string
	PeriodCreditNanousd int64
	DiscountPercent     int
	SortOrder           int
	IsActive            bool
	PermissionGroupID   *uint
	Prices              []BillingPriceView
}

// BillingOverview 表示当前用户计费概览。
type BillingOverview struct {
	Mode                     string
	Plan                     *BillingPlanView
	PeriodStartAt            *time.Time
	PeriodEndAt              *time.Time
	PeriodCreditNanousd      int64
	PeriodUsedNanousd        int64
	PeriodRemainingNanousd   int64
	Account                  *BillingAccountView
	SubscriptionEntitlements []SubscriptionEntitlementView
}

// SubscriptionEntitlementView 表示从当前时间起仍会生效的订阅权益段。
type SubscriptionEntitlementView struct {
	Subscription domainbilling.Subscription
	Plan         BillingPlanView
	IsCurrent    bool
}

// BillingAccountView 表示按量账户余额。
type BillingAccountView struct {
	UserID         uint
	Currency       string
	BalanceNanousd int64
	Status         string
	UpdatedAt      time.Time
}

// BalanceDeltaView 表示管理员余额增减结果。
type BalanceDeltaView struct {
	Account     domainbilling.BillingAccount
	Transaction domainbilling.BalanceTransaction
}

// CheckInStatusView 表示用户每日签到状态。
type CheckInStatusView struct {
	TodayClaimed      bool
	RewardNanousd     int64
	ConsecutiveDays   int
	LastCheckInDate   *time.Time
	NextCheckInDate   time.Time
	Account           *domainbilling.BillingAccount
	LatestTransaction *domainbilling.BalanceTransaction
}

// CheckInClaimView 表示用户每日签到领取结果。
type CheckInClaimView struct {
	Record         domainbilling.CheckInRecord
	Account        *domainbilling.BillingAccount
	Transaction    *domainbilling.BalanceTransaction
	AlreadyClaimed bool
	RewardNanousd  int64
}

// AdminCheckInView 表示后台签到概览和配置。
type AdminCheckInView struct {
	Stats         domainbilling.AdminCheckInStats
	RewardNanousd int64
}

// RiskSummaryView 表示管理员计费页风控摘要。
type RiskSummaryView struct {
	domainbilling.RiskSummary
}

// ModelPricingView 表示后台模型单价及其平台模型身份。
type ModelPricingView struct {
	domainbilling.ModelPricing
	ModelVendor string
	ModelIcon   string
}

// PublicModelPricing 表示用户侧模型选择器需要展示的结构化价格。
type PublicModelPricing struct {
	Currency                string
	IsFree                  bool
	Mode                    string
	InputUSDPerMTokens      float64
	CacheReadUSDPerMTokens  float64
	CacheWriteUSDPerMTokens float64
	OutputUSDPerMTokens     float64
	CallUSDPerCall          float64
	DurationUSDPerSecond    float64
	Tiers                   []PublicModelPricingTier
}

// PublicModelPricingTier 表示原始输入命中阶梯后的区间价格。
type PublicModelPricingTier struct {
	FromTokens              int64
	UpToTokens              *int64
	InputUSDPerMTokens      float64
	CacheReadUSDPerMTokens  float64
	CacheWriteUSDPerMTokens float64
	OutputUSDPerMTokens     float64
}
