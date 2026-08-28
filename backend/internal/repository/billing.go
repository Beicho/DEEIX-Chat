package repository

import (
	"context"
	"time"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
)

// BillingRepository 定义计费流程依赖的持久化能力。
type BillingRepository interface {
	ListActivePlans(ctx context.Context) ([]domainbilling.Plan, error)
	ListActivePricesByPlanIDs(ctx context.Context, planIDs []uint) ([]domainbilling.Price, error)
	GetPriceByID(ctx context.Context, priceID uint) (*domainbilling.Price, error)
	GetPlanByID(ctx context.Context, planID uint) (*domainbilling.Plan, error)
	ListPlansByIDs(ctx context.Context, planIDs []uint) ([]domainbilling.Plan, error)
	GetActivePlanByCode(ctx context.Context, code string) (*domainbilling.Plan, error)
	CreatePlanWithDefaultPrice(ctx context.Context, plan *domainbilling.Plan, price *domainbilling.Price) (*domainbilling.Plan, *domainbilling.Price, error)
	UpdatePlanWithDefaultPrice(ctx context.Context, plan *domainbilling.Plan, price *domainbilling.Price) error
	DeletePlan(ctx context.Context, planID uint) error
	ListCurrentSubscriptionsByUserIDs(ctx context.Context, userIDs []uint, now time.Time) ([]domainbilling.Subscription, error)
	ListSubscriptionEntitlementsByUserIDs(ctx context.Context, userIDs []uint, now time.Time) ([]domainbilling.Subscription, error)
	ReplaceSubscription(ctx context.Context, item *domainbilling.Subscription) error
	CreatePaymentOrder(ctx context.Context, item *domainbilling.PaymentOrder) (*domainbilling.PaymentOrder, error)
	UpdatePaymentOrderCheckout(ctx context.Context, orderNo string, externalCheckoutID string, checkoutURL string) error
	GetPaymentOrderByOrderNo(ctx context.Context, orderNo string) (*domainbilling.PaymentOrder, error)
	UpdatePaymentOrderStatus(ctx context.Context, orderNo string, status string) (*domainbilling.PaymentOrder, error)
	MarkPaymentOrderPaidAndGrantSubscription(ctx context.Context, orderNo string, externalPaymentID string, paidAt time.Time, subscription *domainbilling.Subscription) (*domainbilling.PaymentOrder, bool, error)
	AddUsage(ctx context.Context, usage *domainbilling.UsageLedger) error
	AddUsageAndSettleBalance(ctx context.Context, usage *domainbilling.UsageLedger, reservation *domainbilling.UsageBalanceReservation) error
	AddPeriodUsageAndSettleOverage(ctx context.Context, usage *domainbilling.UsageLedger, periodStart time.Time, periodEnd time.Time, periodCreditNanousd int64, reservation *domainbilling.UsageBalanceReservation) error
	ReserveUsageBalance(ctx context.Context, input domainbilling.UsageBalanceReservationRequest) (*domainbilling.UsageBalanceReservation, error)
	RenewUsageBalanceReservation(ctx context.Context, userID uint, refNo string) error
	ReleaseUsageBalanceReservation(ctx context.Context, userID uint, refNo string) error
	MarkUsageReservationReconciliationRequired(ctx context.Context, userID uint, refNo string, failureCode string) error
	GetOrCreateBillingAccount(ctx context.Context, userID uint) (*domainbilling.BillingAccount, error)
	ListBillingAccountsByUserIDs(ctx context.Context, userIDs []uint) ([]domainbilling.BillingAccount, error)
	SetBillingAccountBalance(ctx context.Context, userID uint, balanceNanousd int64, refNo string, description string) (*domainbilling.BillingAccount, error)
	AdjustBillingAccountBalance(ctx context.Context, userID uint, deltaNanousd int64, refNo string, description string) (*domainbilling.BillingAccount, *domainbilling.BalanceTransaction, error)
	ListBalanceTransactions(ctx context.Context, filter BalanceTransactionListFilter, offset int, limit int) ([]domainbilling.BalanceTransaction, int64, error)
	ClaimDailyCheckIn(ctx context.Context, input CheckInClaimInput) (*CheckInClaimResult, error)
	GetLatestCheckIn(ctx context.Context, userID uint) (*domainbilling.CheckInRecord, error)
	GetCheckInRewardNanousd(ctx context.Context) (int64, error)
	SetCheckInRewardNanousd(ctx context.Context, rewardNanousd int64) error
	GetAdminCheckInStats(ctx context.Context, activeSince time.Time) (*domainbilling.AdminCheckInStats, error)
	GetExternalAccountLink(ctx context.Context, userID uint, platform string) (*domainbilling.ExternalAccountLink, error)
	FindUserLinuxDOSub(ctx context.Context, userID uint) (string, error)
	UpsertExternalAccountLink(ctx context.Context, link *domainbilling.ExternalAccountLink) (*domainbilling.ExternalAccountLink, error)
	CreditExternalTransfer(ctx context.Context, input ExternalTransferCreditInput) (*domainbilling.ExternalTransfer, *domainbilling.BalanceTransaction, error)
	ListExternalTransfers(ctx context.Context, filter ExternalTransferListFilter, offset int, limit int) ([]domainbilling.ExternalTransfer, int64, error)
	UpdateExternalTransferStatus(ctx context.Context, idempotencyKey string, status string, failureReason string) error
	MarkPaymentOrderPaidAndCreditBalance(ctx context.Context, orderNo string, externalPaymentID string, paidAt time.Time) (*domainbilling.PaymentOrder, bool, error)
	ListRedemptionCodes(ctx context.Context, filter RedemptionCodeListFilter, offset int, limit int) ([]domainbilling.RedemptionCode, int64, error)
	GetRedemptionCodeByID(ctx context.Context, id uint) (*domainbilling.RedemptionCode, error)
	CreateRedemptionCode(ctx context.Context, item *domainbilling.RedemptionCode) (*domainbilling.RedemptionCode, error)
	PatchRedemptionCode(ctx context.Context, id uint, patch RedemptionCodePatch) (*domainbilling.RedemptionCode, error)
	DeleteRedemptionCode(ctx context.Context, id uint) error
	RedeemCode(ctx context.Context, input RedemptionApplyInput) (*RedemptionApplyResult, error)
	ListRedemptions(ctx context.Context, filter RedemptionListFilter, offset int, limit int) ([]domainbilling.Redemption, int64, error)
	ListRedemptionRecords(ctx context.Context, filter RedemptionListFilter, offset int, limit int) ([]RedemptionRecord, int64, error)
	GetBillingMode(ctx context.Context) (string, error)
	GetBillingPrepaidAmountNanousd(ctx context.Context) (int64, error)
	GetNativeToolBillingEnabled(ctx context.Context) (bool, error)
	GetNativeToolPricingJSON(ctx context.Context) (string, error)
	GetModelPricing(ctx context.Context, platformModelName string) (*domainbilling.ModelPricing, error)
	ListModelPricing(ctx context.Context, query string, offset int, limit int) ([]domainbilling.ModelPricing, int64, error)
	UpsertModelPricing(ctx context.Context, item *domainbilling.ModelPricing) (*domainbilling.ModelPricing, error)
	ListUsageByUser(ctx context.Context, userID uint, filter UsageListFilter, offset int, limit int) ([]domainbilling.UsageLedger, int64, error)
	ListUsageLogs(ctx context.Context, filter UsageLogListFilter, offset int, limit int) ([]domainbilling.UsageLedger, int64, error)
	ListPaymentOrders(ctx context.Context, filter PaymentOrderListFilter, offset int, limit int) ([]domainbilling.PaymentOrder, int64, error)
	GetUserCreatedAt(ctx context.Context, userID uint) (time.Time, error)
	ListMonthlyUsageByUser(ctx context.Context, userID uint, limit int) ([]domainbilling.UsageMonthlySummary, error)
	ListDailyUsageByUser(ctx context.Context, userID uint, startDate time.Time, endDate time.Time) ([]domainbilling.UsageDailySummary, error)
	SumBillableNanousd(ctx context.Context, userID uint, startAt time.Time, endAt time.Time) (int64, error)
	GetAdminDashboardStats(ctx context.Context, startAt time.Time, endAt time.Time, limit int) (*domainbilling.AdminDashboardStats, error)
	GetBillingRiskSummary(ctx context.Context) (*domainbilling.RiskSummary, error)
}

// BalanceTransactionListFilter 描述余额流水分页筛选条件。
type BalanceTransactionListFilter struct {
	UserID uint
	Type   string
	Query  string
	Sort   string
	From   *time.Time
	To     *time.Time
}

// CheckInClaimInput 描述每日签到奖励入账请求。
type CheckInClaimInput struct {
	UserID         uint
	CheckInDate    time.Time
	RewardNanousd  int64
	RefNo          string
	Description    string
	ConsecutiveDay int
}

// CheckInClaimResult 描述每日签到事务结果。
type CheckInClaimResult struct {
	Record         domainbilling.CheckInRecord
	Account        *domainbilling.BillingAccount
	Transaction    *domainbilling.BalanceTransaction
	AlreadyClaimed bool
}

// ExternalTransferCreditInput 描述一次外部余额划转本地入账。
type ExternalTransferCreditInput struct {
	UserID                uint
	LinkID                uint
	Platform              string
	Direction             string
	ExternalTransferID    string
	IdempotencyKey        string
	ExternalAmountUSD     float64
	CreditedAmountNanousd int64
	RefNo                 string
	Description           string
}

// ExternalTransferListFilter 描述外部划转分页筛选条件。
type ExternalTransferListFilter struct {
	UserID      uint
	Platform    string
	Status      string
	Sort        string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// RedemptionListFilter 描述用户兑换记录分页筛选条件。
type RedemptionListFilter struct {
	UserID uint
	Mode   string
	Query  string
	Sort   string
}

// RedemptionCodeListFilter 描述管理员兑换码列表筛选条件。
type RedemptionCodeListFilter struct {
	Mode         string
	Modes        []string
	Status       string
	Availability string
	Query        string
}

// RedemptionCodePatch 描述可更新的兑换码管理字段。
type RedemptionCodePatch struct {
	Status            *string
	MaxRedemptionsSet bool
	MaxRedemptions    *int
	PerUserLimit      *int
	ExpiresAtSet      bool
	ExpiresAt         *time.Time
	Description       *string
}

// RedemptionListFilter 描述管理员兑换记录列表筛选条件。
type RedemptionListFilter struct {
	CodeID      uint
	UserID      uint
	RewardType  string
	Query       string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Sort        string
}

// RedemptionRecord 表示带兑换码与余额流水上下文的兑换记录。
// 兑换码删除为状态软删，历史记录始终可联表查询。
type RedemptionRecord struct {
	Redemption      domainbilling.Redemption
	CodeHint        string
	CodeDescription string
	CodeStatus      string
	PlanName        string
	// BalanceAmountNanousd / BalanceAfterNanousd 来自余额流水；订阅类兑换无流水时为 nil。
	BalanceAmountNanousd *int64
	BalanceAfterNanousd  *int64
}

// RedemptionApplyInput 描述一次兑换需要在事务中完成的写入参数。
type RedemptionApplyInput struct {
	CodeHash       string
	UserID         uint
	CurrentMode    string
	RefNo          string
	SubscriptionAt time.Time
}

// RedemptionApplyResult 描述兑换事务写入结果。
type RedemptionApplyResult struct {
	Code         domainbilling.RedemptionCode
	Redemption   domainbilling.Redemption
	Account      *domainbilling.BillingAccount
	Subscription *domainbilling.Subscription
}

// UsageListFilter 描述用户用量账本的筛选和排序条件。
type UsageListFilter struct {
	Query       string
	Status      string
	Sort        string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// UsageLogListFilter 描述管理员调用日志筛选和排序条件。
type UsageLogListFilter struct {
	Query             string
	PlatformModelName string
	BillingMode       string
	UserID            uint
	CreatedFrom       *time.Time
	CreatedTo         *time.Time
	Sort              string
}

// UsageStatisticsFilter 描述管理员用量统计的聚合范围和维度。
type UsageStatisticsFilter struct {
	StartDate         time.Time
	EndDateExclusive  time.Time
	UserID            uint
	PermissionGroupID uint
	MembershipAt      time.Time
	PlatformModelName string
	BillingScope      string
	Granularity       string
	Section           string
	ModelRankBy       string
	UserRankBy        string
	RankLimit         int
}

// UsageStatisticsRepository 提供独立于计费写入仓储的管理员统计能力。
// 独立接口可避免为统计功能扩大 BillingRepository 及其测试桩。
type UsageStatisticsRepository interface {
	GetUsageStatistics(ctx context.Context, filter UsageStatisticsFilter) (domainbilling.UsageStatistics, error)
}

// PaymentOrderListFilter 描述管理员支付订单列表筛选和排序条件。
type PaymentOrderListFilter struct {
	Query       string
	OrderType   string
	Provider    string
	Status      string
	UserID      uint
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Sort        string
}
