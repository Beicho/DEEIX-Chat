package billing

import "time"

const (
	// PricingModeToken 表示按 token 用量计费。
	PricingModeToken = "token"
	// PricingModeCall 表示按调用次数计费。
	PricingModeCall = "call"
	// PricingModeDuration 表示按生成秒数计费。
	PricingModeDuration = "duration"
	// PricingModeTiered 表示按 token 阶梯计费。
	PricingModeTiered = "tiered"

	// IntervalMonth 表示按月计费。
	IntervalMonth = "month"
	// IntervalYear 表示按年计费。
	IntervalYear = "year"
	// IntervalLifetime 表示永久价格。
	IntervalLifetime = "lifetime"
)

// Plan 表示订阅套餐。
type Plan struct {
	ID                  uint
	Code                string
	Name                string
	Description         string
	FeatureJSON         string
	PeriodCreditNanousd int64
	DiscountPercent     int
	SortOrder           int
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Price 表示套餐价格版本。
type Price struct {
	ID               uint
	PlanID           uint
	Code             string
	BillingInterval  string
	Currency         string
	AmountCents      int64
	IsActive         bool
	IsDefault        bool
	ExternalPriceRef string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Subscription 表示用户订阅。
type Subscription struct {
	ID                   uint
	UserID               uint
	PlanID               uint
	PriceID              uint
	Status               string
	StartAt              time.Time
	CurrentPeriodStartAt time.Time
	CurrentPeriodEndAt   *time.Time
	CancelAtPeriodEnd    bool
	CanceledAt           *time.Time
	AutoRenew            bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

const (
	// PaymentProviderStripe 表示 Stripe 支付。
	PaymentProviderStripe = "stripe"
	// PaymentProviderEPay 表示易支付。
	PaymentProviderEPay = "epay"

	// PaymentStatusPending 表示支付待完成。
	PaymentStatusPending = "pending"
	// PaymentStatusPaid 表示支付已完成。
	PaymentStatusPaid = "paid"
	// PaymentStatusFailed 表示支付失败。
	PaymentStatusFailed = "failed"
	// PaymentStatusExpired 表示支付单已过期。
	PaymentStatusExpired = "expired"

	// PaymentOrderTypeSubscription 表示订阅套餐支付单。
	PaymentOrderTypeSubscription = "subscription"
	// PaymentOrderTypeTopUp 表示按量余额充值支付单。
	PaymentOrderTypeTopUp = "topup"

	// BalanceTransactionTypeTopUp 表示充值入账。
	BalanceTransactionTypeTopUp = "topup"
	// BalanceTransactionTypeUsage 表示按量扣费。
	BalanceTransactionTypeUsage = "usage_debit"
	// BalanceTransactionTypeUsageReserve 表示按量调用前预扣。
	BalanceTransactionTypeUsageReserve = "usage_reserve"
	// BalanceTransactionTypeUsageRefund 表示按量预扣退回。
	BalanceTransactionTypeUsageRefund = "usage_refund"
	// BalanceTransactionTypeAdminSet 表示管理员设置余额。
	BalanceTransactionTypeAdminSet = "admin_set"
	// BalanceTransactionTypeRedemption 表示兑换码入账。
	BalanceTransactionTypeRedemption = "redemption"
	// BalanceTransactionTypeCheckIn 表示每日签到奖励入账。
	BalanceTransactionTypeCheckIn = "checkin"
	// BalanceTransactionTypeNewAPITransferIn 表示从 NewAPI 单向转入余额。
	BalanceTransactionTypeNewAPITransferIn = "newapi_transfer_in"

	// BalanceTransactionRefTypeCheckIn 表示余额流水关联签到记录。
	BalanceTransactionRefTypeCheckIn = "checkin"
	// BalanceTransactionRefTypeExternalTransfer 表示余额流水关联外部划转记录。
	BalanceTransactionRefTypeExternalTransfer = "external_transfer"
)

// PaymentOrder 表示一次支付单。
type PaymentOrder struct {
	ID                 uint
	OrderNo            string
	OrderType          string
	UserID             uint
	PlanID             uint
	PriceID            uint
	Provider           string
	Status             string
	BaseCurrency       string
	BaseAmountCents    int64
	PayCurrency        string
	PayAmountCents     int64
	FXRate             string
	CreditNanousd      int64
	BillingInterval    string
	Cycles             int
	ExternalPaymentID  string
	ExternalCheckoutID string
	CheckoutURL        string
	PaidAt             *time.Time
	ExpiredAt          *time.Time
	SnapshotJSON       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BillingAccount 表示用户按量计费账户。
type BillingAccount struct {
	ID             uint
	UserID         uint
	Currency       string
	BalanceNanousd int64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// BalanceTransaction 表示余额变动流水。
type BalanceTransaction struct {
	ID                  uint
	AccountID           uint
	UserID              uint
	Type                string
	AmountNanousd       int64
	BalanceAfterNanousd int64
	RefType             string
	RefID               uint
	RefNo               string
	Description         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CheckInRecord 表示用户每日签到奖励记录。
type CheckInRecord struct {
	ID                   uint
	UserID               uint
	CheckInDate          time.Time
	RewardNanousd        int64
	ConsecutiveDays      int
	BalanceTransactionID uint
	RefNo                string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TaskProgress 表示任务奖励预留进度结构。
type TaskProgress struct {
	ID                   uint
	UserID               uint
	TaskKey              string
	Progress             int
	Target               int
	Status               string
	RewardNanousd        int64
	BalanceTransactionID uint
	CompletedAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

const (
	// ExternalPlatformNewAPI 表示外部平台 NewAPI。
	ExternalPlatformNewAPI = "newapi"

	// ExternalAccountLinkStatusActive 表示绑定可用。
	ExternalAccountLinkStatusActive = "active"
	// ExternalAccountLinkStatusDisabled 表示绑定停用。
	ExternalAccountLinkStatusDisabled = "disabled"

	// ExternalTransferDirectionIn 表示外部余额转入本站。
	ExternalTransferDirectionIn = "in"

	// ExternalTransferStatusPending 表示划转已创建但尚未入账。
	ExternalTransferStatusPending = "pending"
	// ExternalTransferStatusCredited 表示本站已入账。
	ExternalTransferStatusCredited = "credited"
	// ExternalTransferStatusRolledBack 表示外部侧已回滚。
	ExternalTransferStatusRolledBack = "rolled_back"
	// ExternalTransferStatusCancelFailed 表示回滚失败，需要人工处理。
	ExternalTransferStatusCancelFailed = "cancel_failed"
)

// ExternalAccountLink 表示本站用户与外部平台账号的绑定关系。
type ExternalAccountLink struct {
	ID                  uint
	UserID              uint
	Platform            string
	ExternalUserID      string
	ExternalDisplayName string
	LinuxDOSub          string
	Status              string
	LinkedAt            time.Time
	LastSyncedAt        *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ExternalTransfer 表示外部余额单向划转记录。
type ExternalTransfer struct {
	ID                    uint
	UserID                uint
	LinkID                uint
	Platform              string
	Direction             string
	ExternalTransferID    string
	IdempotencyKey        string
	ExternalAmountUSD     float64
	CreditedAmountNanousd int64
	BalanceTransactionID  uint
	Status                string
	FailureReason         string
	RequestedAt           time.Time
	CompletedAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// RiskSummary describes billing-facing risk signals for the admin panel.
type RiskSummary struct {
	MultiAccountClusterCount int64
	HighRiskClusterCount     int64
	IgnoredClusterCount      int64
	UniqueFingerprintCount   int64
	UniqueIPCount            int64
	GeneratedAt              time.Time
}

const (
	// RedemptionCodeModeUsage 表示按量余额兑换码。
	RedemptionCodeModeUsage = "usage"
	// RedemptionCodeModePeriod 表示订阅套餐兑换码。
	RedemptionCodeModePeriod = "period"

	// RedemptionRewardTypeBalance 表示奖励按量余额。
	RedemptionRewardTypeBalance = "balance"
	// RedemptionRewardTypeSubscription 表示奖励订阅套餐。
	RedemptionRewardTypeSubscription = "subscription"

	// RedemptionCodeStatusActive 表示兑换码启用。
	RedemptionCodeStatusActive = "active"
	// RedemptionCodeStatusInactive 表示兑换码停用。
	RedemptionCodeStatusInactive = "inactive"
	// RedemptionCodeStatusDeleted 表示兑换码已删除，保留历史兑换审计。
	RedemptionCodeStatusDeleted = "deleted"
)

// RedemptionCode 表示一组可兑换的计费权益码。
type RedemptionCode struct {
	ID              uint
	CodeHash        string
	CodeEncrypted   string
	CodeHint        string
	Mode            string
	RewardType      string
	CreditNanousd   int64
	PlanID          uint
	DurationDays    int
	MaxRedemptions  *int
	PerUserLimit    int
	RedeemedCount   int
	Status          string
	ExpiresAt       *time.Time
	Description     string
	CreatedByUserID uint
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Redemption 表示用户的一次兑换记录。
type Redemption struct {
	ID                   uint
	CodeID               uint
	UserID               uint
	Mode                 string
	RewardType           string
	CreditNanousd        int64
	PlanID               uint
	SubscriptionID       uint
	BalanceTransactionID uint
	RefNo                string
	SnapshotJSON         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UsageBalanceReservation 表示一次按量调用的预扣记录。
type UsageBalanceReservation struct {
	UserID        uint
	AmountNanousd int64
	RefNo         string
}

// ModelPricing 表示平台模型名对应的统一计费单价。
type ModelPricing struct {
	ID                          uint
	PlatformModelName           string
	Currency                    string
	IsFree                      bool
	PricingMode                 string
	InputNanousdPerMTokens      int64
	CacheReadNanousdPerMTokens  int64
	CacheWriteNanousdPerMTokens int64
	OutputNanousdPerMTokens     int64
	CallNanousdPerCall          int64
	DurationNanousdPerSecond    int64
	TieredPricingJSON           string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

// UsageLedger 表示用量账本。
type UsageLedger struct {
	ID                  uint
	UserID              uint
	ConversationID      uint
	ProviderProtocol    string
	UpstreamName        string
	PlatformModelName   string
	RoutedBindingCode   string
	UpstreamModelName   string
	IsFreeModel         bool
	UsageDate           time.Time
	InputTokens         int64
	CacheReadTokens     int64
	CacheWriteTokens    int64
	CacheWrite5mTokens  int64
	CacheWrite1hTokens  int64
	OutputTokens        int64
	ReasoningTokens     int64
	CallCount           int64
	DurationSeconds     int64
	LatencyMS           int64
	UsageSpeed          string
	ServiceTier         string
	BilledCurrency      string
	BilledNanousd       int64
	PricingSnapshotJSON string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// UsageServiceItem 表示一次基础服务计费明细。
type UsageServiceItem struct {
	ServiceCode                   string
	ServiceName                   string
	PlatformModelName             string
	ProviderProtocol              string
	CacheTimeout                  string
	RequestSpeed                  string
	UsageSpeed                    string
	BillingSpeed                  string
	RequestServiceTier            string
	UsageServiceTier              string
	BillingServiceTier            string
	FastMode                      bool
	RateMultiplier                float64
	PricingMode                   string
	InputTokens                   int64
	CacheReadTokens               int64
	CacheWriteTokens              int64
	CacheWrite5mTokens            int64
	CacheWrite1hTokens            int64
	OutputTokens                  int64
	ReasoningTokens               int64
	CallCount                     int64
	DurationSeconds               int64
	InputNanousdPerMTokens        int64
	CacheReadNanousdPerMTokens    int64
	CacheWriteNanousdPerMTokens   int64
	CacheWrite5mNanousdPerMTokens int64
	CacheWrite1hNanousdPerMTokens int64
	OutputNanousdPerMTokens       int64
	CallNanousdPerCall            int64
	DurationNanousdPerSecond      int64
	InputBilledNanousd            int64
	CacheReadBilledNanousd        int64
	CacheWriteBilledNanousd       int64
	CacheWrite5mBilledNanousd     int64
	CacheWrite1hBilledNanousd     int64
	OutputBilledNanousd           int64
	CallBilledNanousd             int64
	DurationBilledNanousd         int64
	BilledNanousd                 int64
	TieredFromTokens              int64
	TieredUpToTokens              *int64
}

// UsageMonthlySummary 表示用户月度用量聚合。
type UsageMonthlySummary struct {
	MonthStartAt     time.Time
	RecordCount      int64
	InputTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	OutputTokens     int64
	ReasoningTokens  int64
	CallCount        int64
	DurationSeconds  int64
	AvgLatencyMS     int64
	BilledNanousd    int64
}

// UsageDailySummary 表示用户每日用量聚合。
type UsageDailySummary struct {
	UsageDate        time.Time
	RecordCount      int64
	InputTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	OutputTokens     int64
	ReasoningTokens  int64
	CallCount        int64
	DurationSeconds  int64
	AvgLatencyMS     int64
	BilledNanousd    int64
	Models           []UsageDailyModelSummary
}

// UsageDailyModelSummary 表示用户每日模型维度用量聚合。
type UsageDailyModelSummary struct {
	PlatformModelName string
	RecordCount       int64
	InputTokens       int64
	CacheReadTokens   int64
	CacheWriteTokens  int64
	OutputTokens      int64
	ReasoningTokens   int64
	CallCount         int64
	DurationSeconds   int64
	AvgLatencyMS      int64
	BilledNanousd     int64
}

// AdminDashboardStats is the aggregate billing snapshot shown in the admin dashboard.
type AdminDashboardStats struct {
	Usage       AdminDashboardUsageStats
	Sales       AdminDashboardSalesStats
	TopModels   []AdminDashboardModelStats
	GeneratedAt time.Time
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// AdminDashboardUsageStats summarizes usage ledgers in the selected period.
type AdminDashboardUsageStats struct {
	RecordCount     int64
	ActiveUserCount int64
	CallCount       int64
	TokenCount      int64
	DurationSeconds int64
	BilledNanousd   int64
}

// AdminDashboardSalesStats summarizes paid payment orders in the selected period.
type AdminDashboardSalesStats struct {
	PaidOrderCount  int64
	BaseAmountCents int64
	CreditNanousd   int64
}

// AdminDashboardModelStats summarizes usage by platform model.
type AdminDashboardModelStats struct {
	PlatformModelName string
	RecordCount       int64
	ActiveUserCount   int64
	CallCount         int64
	TokenCount        int64
	DurationSeconds   int64
	BilledNanousd     int64
}
