package billing

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/dberror"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// translateError 将 gorm 底层错误统一映射为仓储语义错误。
func translateError(err error) error {
	if err == nil {
		return nil
	}
	if dberror.IsRecordNotFound(err) {
		return repository.ErrNotFound
	}
	if dberror.IsUniqueConstraint(err) {
		return repository.ErrDuplicate
	}
	return err
}

// Repo 封装计费数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建仓储。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) sqliteDialect() bool {
	return r != nil && r.db != nil && r.db.Dialector != nil && r.db.Dialector.Name() == "sqlite"
}

func (r *Repo) pricingModeExpression() string {
	if r.sqliteDialect() {
		return "COALESCE(NULLIF(json_extract(pricing_snapshot_json, '$.pricing_mode'), ''), 'token')"
	}
	return "COALESCE(NULLIF(pricing_snapshot_json, '')::jsonb ->> 'pricing_mode', 'token')"
}

func (r *Repo) usageDayKeyExpression() string {
	if r.sqliteDialect() {
		return "strftime('%Y-%m-%d', usage_date)"
	}
	return "TO_CHAR(usage_date, 'YYYY-MM-DD')"
}

func (r *Repo) usageMonthKeyExpression() string {
	if r.sqliteDialect() {
		return "strftime('%Y-%m-01', usage_date)"
	}
	return "TO_CHAR(date_trunc('month', usage_date), 'YYYY-MM-DD')"
}

// ListActivePlans 查询启用套餐。
func (r *Repo) ListActivePlans(ctx context.Context) ([]domainbilling.Plan, error) {
	items := make([]model.BillingPlan, 0)
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.Plan, 0, len(items))
	for _, item := range items {
		results = append(results, domainbilling.Plan{
			ID:                  item.ID,
			Code:                item.Code,
			Name:                item.Name,
			Description:         item.Description,
			FeatureJSON:         item.FeatureJSON,
			PeriodCreditNanousd: item.PeriodCreditNanousd,
			DiscountPercent:     item.DiscountPercent,
			SortOrder:           item.SortOrder,
			IsActive:            item.IsActive,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		})
	}
	return results, nil
}

// ListActivePricesByPlanIDs 查询一批套餐的启用价格。
func (r *Repo) ListActivePricesByPlanIDs(ctx context.Context, planIDs []uint) ([]domainbilling.Price, error) {
	items := make([]model.BillingPrice, 0)
	if len(planIDs) == 0 {
		return []domainbilling.Price{}, nil
	}
	if err := r.db.WithContext(ctx).
		Where("plan_id IN ? AND is_active = ?", planIDs, true).
		Order("plan_id ASC, amount_cents ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.Price, 0, len(items))
	for _, item := range items {
		results = append(results, domainbilling.Price{
			ID:               item.ID,
			PlanID:           item.PlanID,
			Code:             item.Code,
			BillingInterval:  item.BillingInterval,
			Currency:         item.Currency,
			AmountCents:      item.AmountCents,
			IsActive:         item.IsActive,
			IsDefault:        item.IsDefault,
			ExternalPriceRef: item.ExternalPriceRef,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	return results, nil
}

// GetPriceByID 查询价格。
func (r *Repo) GetPriceByID(ctx context.Context, priceID uint) (*domainbilling.Price, error) {
	var item model.BillingPrice
	if err := r.db.WithContext(ctx).Where("id = ?", priceID).First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	return &domainbilling.Price{
		ID:               item.ID,
		PlanID:           item.PlanID,
		Code:             item.Code,
		BillingInterval:  item.BillingInterval,
		Currency:         item.Currency,
		AmountCents:      item.AmountCents,
		IsActive:         item.IsActive,
		IsDefault:        item.IsDefault,
		ExternalPriceRef: item.ExternalPriceRef,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}, nil
}

// GetPlanByID 查询套餐。
func (r *Repo) GetPlanByID(ctx context.Context, planID uint) (*domainbilling.Plan, error) {
	var item model.BillingPlan
	if err := r.db.WithContext(ctx).Where("id = ?", planID).First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	return &domainbilling.Plan{
		ID:                  item.ID,
		Code:                item.Code,
		Name:                item.Name,
		Description:         item.Description,
		FeatureJSON:         item.FeatureJSON,
		PeriodCreditNanousd: item.PeriodCreditNanousd,
		DiscountPercent:     item.DiscountPercent,
		SortOrder:           item.SortOrder,
		IsActive:            item.IsActive,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}, nil
}

// ListPlansByIDs 查询一批套餐。
func (r *Repo) ListPlansByIDs(ctx context.Context, planIDs []uint) ([]domainbilling.Plan, error) {
	items := make([]model.BillingPlan, 0)
	if len(planIDs) == 0 {
		return []domainbilling.Plan{}, nil
	}
	if err := r.db.WithContext(ctx).
		Where("id IN ?", planIDs).
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.Plan, 0, len(items))
	for _, item := range items {
		results = append(results, domainbilling.Plan{
			ID:                  item.ID,
			Code:                item.Code,
			Name:                item.Name,
			Description:         item.Description,
			FeatureJSON:         item.FeatureJSON,
			PeriodCreditNanousd: item.PeriodCreditNanousd,
			DiscountPercent:     item.DiscountPercent,
			SortOrder:           item.SortOrder,
			IsActive:            item.IsActive,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		})
	}
	return results, nil
}

// GetActivePlanByCode 按编码查询启用套餐。
func (r *Repo) GetActivePlanByCode(ctx context.Context, code string) (*domainbilling.Plan, error) {
	var item model.BillingPlan
	if err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = ?", strings.TrimSpace(code), true).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	return &domainbilling.Plan{
		ID:                  item.ID,
		Code:                item.Code,
		Name:                item.Name,
		Description:         item.Description,
		FeatureJSON:         item.FeatureJSON,
		PeriodCreditNanousd: item.PeriodCreditNanousd,
		DiscountPercent:     item.DiscountPercent,
		SortOrder:           item.SortOrder,
		IsActive:            item.IsActive,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}, nil
}

// CreatePlanWithDefaultPrice 创建套餐与默认价格。
func (r *Repo) CreatePlanWithDefaultPrice(ctx context.Context, plan *domainbilling.Plan, price *domainbilling.Price) (*domainbilling.Plan, *domainbilling.Price, error) {
	if plan == nil || price == nil || strings.TrimSpace(plan.Code) == "" {
		return nil, nil, repository.ErrInvalidInput
	}
	var resultPlan domainbilling.Plan
	var resultPrice domainbilling.Price
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		planRecord := model.BillingPlan{
			Code:                strings.TrimSpace(plan.Code),
			Name:                strings.TrimSpace(plan.Name),
			Description:         strings.TrimSpace(plan.Description),
			FeatureJSON:         firstNonEmpty(strings.TrimSpace(plan.FeatureJSON), "[]"),
			PeriodCreditNanousd: clampNonNegative(plan.PeriodCreditNanousd),
			DiscountPercent:     clampPercent(plan.DiscountPercent),
			SortOrder:           plan.SortOrder,
			IsActive:            plan.IsActive,
		}
		if err := tx.Create(&planRecord).Error; err != nil {
			return translateError(err)
		}
		priceRecord := model.BillingPrice{
			PlanID:          planRecord.ID,
			Code:            strings.TrimSpace(price.Code),
			BillingInterval: normalizeInterval(price.BillingInterval),
			Currency:        normalizeCurrency(price.Currency),
			AmountCents:     clampNonNegative(price.AmountCents),
			IsActive:        true,
			IsDefault:       true,
		}
		if priceRecord.Code == "" {
			priceRecord.Code = planRecord.Code + "-default"
		}
		if err := tx.Create(&priceRecord).Error; err != nil {
			return translateError(err)
		}
		resultPlan = toDomainPlan(planRecord)
		resultPrice = toDomainPrice(priceRecord)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &resultPlan, &resultPrice, nil
}

// UpdatePlanWithDefaultPrice 更新套餐与默认价格。
func (r *Repo) UpdatePlanWithDefaultPrice(ctx context.Context, plan *domainbilling.Plan, price *domainbilling.Price) error {
	if plan == nil || price == nil || plan.ID == 0 {
		return repository.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		planUpdates := map[string]interface{}{
			"name":                  strings.TrimSpace(plan.Name),
			"description":           strings.TrimSpace(plan.Description),
			"period_credit_nanousd": clampNonNegative(plan.PeriodCreditNanousd),
			"discount_percent":      clampPercent(plan.DiscountPercent),
			"is_active":             true,
		}
		if err := tx.Model(&model.BillingPlan{}).
			Where("id = ?", plan.ID).
			Updates(planUpdates).Error; err != nil {
			return translateError(err)
		}

		if err := tx.Model(&model.BillingPrice{}).
			Where("plan_id = ? AND is_default = ?", plan.ID, true).
			Update("is_default", false).Error; err != nil {
			return translateError(err)
		}

		var record model.BillingPrice
		err := tx.Where("plan_id = ? AND code = ?", plan.ID, strings.TrimSpace(price.Code)).
			First(&record).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}

		updates := map[string]interface{}{
			"plan_id":            plan.ID,
			"code":               strings.TrimSpace(price.Code),
			"billing_interval":   normalizeInterval(price.BillingInterval),
			"currency":           normalizeCurrency(price.Currency),
			"amount_cents":       clampNonNegative(price.AmountCents),
			"is_active":          true,
			"is_default":         true,
			"external_price_ref": strings.TrimSpace(price.ExternalPriceRef),
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = model.BillingPrice{
				PlanID: plan.ID,
				Code:   strings.TrimSpace(price.Code),
			}
			if err := tx.Create(&record).Error; err != nil {
				return translateError(err)
			}
		}
		return translateError(tx.Model(&record).Updates(updates).Error)
	})
}

// DeletePlan 软删除套餐并停用其价格。
func (r *Repo) DeletePlan(ctx context.Context, planID uint) error {
	if planID == 0 {
		return repository.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var activeSubscriptions int64
		if err := tx.Model(&model.Subscription{}).
			Where("plan_id = ? AND status = ?", planID, "active").
			Count(&activeSubscriptions).Error; err != nil {
			return translateError(err)
		}
		if activeSubscriptions > 0 {
			return repository.ErrInvalidInput
		}
		if err := tx.Model(&model.BillingPrice{}).
			Where("plan_id = ?", planID).
			Update("is_active", false).Error; err != nil {
			return translateError(err)
		}
		result := tx.Model(&model.BillingPlan{}).
			Where("id = ?", planID).
			Update("is_active", false)
		if result.Error != nil {
			return translateError(result.Error)
		}
		if result.RowsAffected == 0 {
			return repository.ErrNotFound
		}
		return nil
	})
}

// ListCurrentSubscriptionsByUserIDs 查询一批用户当前有效的活跃订阅。
func (r *Repo) ListCurrentSubscriptionsByUserIDs(
	ctx context.Context,
	userIDs []uint,
	now time.Time,
) ([]domainbilling.Subscription, error) {
	items := make([]model.Subscription, 0)
	if len(userIDs) == 0 {
		return []domainbilling.Subscription{}, nil
	}

	if err := r.db.WithContext(ctx).
		Where(
			"user_id IN ? AND status = ? AND current_period_start_at <= ? AND (current_period_end_at IS NULL OR current_period_end_at > ?)",
			userIDs,
			"active",
			now,
			now,
		).
		Order("user_id ASC, current_period_start_at ASC, current_period_end_at ASC NULLS LAST, id ASC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.Subscription, 0, len(items))
	for _, item := range items {
		results = append(results, domainbilling.Subscription{
			ID:                   item.ID,
			UserID:               item.UserID,
			PlanID:               item.PlanID,
			PriceID:              item.PriceID,
			Status:               item.Status,
			StartAt:              item.StartAt,
			CurrentPeriodStartAt: item.CurrentPeriodStartAt,
			CurrentPeriodEndAt:   item.CurrentPeriodEndAt,
			CancelAtPeriodEnd:    item.CancelAtPeriodEnd,
			CanceledAt:           item.CanceledAt,
			AutoRenew:            item.AutoRenew,
			CreatedAt:            item.CreatedAt,
			UpdatedAt:            item.UpdatedAt,
		})
	}
	return results, nil
}

// ListSubscriptionEntitlementsByUserIDs 查询一批用户从 now 起仍有效的当前与未来订阅权益。
func (r *Repo) ListSubscriptionEntitlementsByUserIDs(
	ctx context.Context,
	userIDs []uint,
	now time.Time,
) ([]domainbilling.Subscription, error) {
	items := make([]model.Subscription, 0)
	if len(userIDs) == 0 {
		return []domainbilling.Subscription{}, nil
	}

	if err := r.db.WithContext(ctx).
		Where(
			"user_id IN ? AND status = ? AND (current_period_end_at IS NULL OR current_period_end_at > ?)",
			userIDs,
			"active",
			now,
		).
		Order("user_id ASC, current_period_start_at ASC, current_period_end_at ASC NULLS LAST, id ASC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.Subscription, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainSubscription(item))
	}
	return results, nil
}

// CreateSubscription 创建订阅。
func (r *Repo) CreateSubscription(ctx context.Context, item *model.Subscription) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// ReplaceSubscription 原子替换用户当前活跃订阅。
func (r *Repo) ReplaceSubscription(ctx context.Context, item *domainbilling.Subscription) error {
	if item == nil {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Subscription{}).
			Where("user_id = ? AND status = ?", item.UserID, "active").
			Updates(map[string]interface{}{
				"status":                "expired",
				"auto_renew":            false,
				"cancel_at_period_end":  false,
				"current_period_end_at": time.Now(),
			}).Error; err != nil {
			return err
		}
		record := model.Subscription{
			UserID:               item.UserID,
			PlanID:               item.PlanID,
			PriceID:              item.PriceID,
			Status:               item.Status,
			StartAt:              item.StartAt,
			CurrentPeriodStartAt: item.CurrentPeriodStartAt,
			CurrentPeriodEndAt:   item.CurrentPeriodEndAt,
			CancelAtPeriodEnd:    item.CancelAtPeriodEnd,
			CanceledAt:           item.CanceledAt,
			AutoRenew:            item.AutoRenew,
		}
		return tx.Create(&record).Error
	})
}

// CreatePaymentOrder 创建支付单。
func (r *Repo) CreatePaymentOrder(ctx context.Context, item *domainbilling.PaymentOrder) (*domainbilling.PaymentOrder, error) {
	if item == nil || strings.TrimSpace(item.OrderNo) == "" {
		return nil, repository.ErrInvalidInput
	}
	record := model.PaymentOrder{
		OrderNo:         strings.TrimSpace(item.OrderNo),
		OrderType:       normalizeOrderType(item.OrderType),
		UserID:          item.UserID,
		PlanID:          item.PlanID,
		PriceID:         item.PriceID,
		Provider:        strings.TrimSpace(item.Provider),
		Status:          firstNonEmpty(strings.TrimSpace(item.Status), domainbilling.PaymentStatusPending),
		BaseCurrency:    normalizeCurrency(item.BaseCurrency),
		BaseAmountCents: clampNonNegative(item.BaseAmountCents),
		PayCurrency:     normalizeCurrency(item.PayCurrency),
		PayAmountCents:  clampNonNegative(item.PayAmountCents),
		FXRate:          strings.TrimSpace(item.FXRate),
		CreditNanousd:   clampNonNegative(item.CreditNanousd),
		BillingInterval: normalizeInterval(item.BillingInterval),
		Cycles:          item.Cycles,
		ExpiredAt:       item.ExpiredAt,
		SnapshotJSON:    strings.TrimSpace(item.SnapshotJSON),
	}
	if record.Cycles <= 0 {
		record.Cycles = 1
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainPaymentOrder(record)
	return &result, nil
}

// UpdatePaymentOrderCheckout 保存外部收银台信息。
func (r *Repo) UpdatePaymentOrderCheckout(ctx context.Context, orderNo string, externalCheckoutID string, checkoutURL string) error {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return repository.ErrInvalidInput
	}
	return translateError(r.db.WithContext(ctx).
		Model(&model.PaymentOrder{}).
		Where("order_no = ?", orderNo).
		Updates(map[string]interface{}{
			"external_checkout_id": strings.TrimSpace(externalCheckoutID),
			"checkout_url":         strings.TrimSpace(checkoutURL),
		}).Error)
}

// GetPaymentOrderByOrderNo 查询支付单。
func (r *Repo) GetPaymentOrderByOrderNo(ctx context.Context, orderNo string) (*domainbilling.PaymentOrder, error) {
	var record model.PaymentOrder
	if err := r.db.WithContext(ctx).Where("order_no = ?", strings.TrimSpace(orderNo)).First(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainPaymentOrder(record)
	return &result, nil
}

// ListPaymentOrders 分页查询支付单。
func (r *Repo) ListPaymentOrders(ctx context.Context, filter repository.PaymentOrderListFilter, offset int, limit int) ([]domainbilling.PaymentOrder, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	items := make([]model.PaymentOrder, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.PaymentOrder{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if orderType := strings.TrimSpace(filter.OrderType); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if provider := strings.TrimSpace(filter.Provider); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(order_no) LIKE ? OR LOWER(provider) LIKE ? OR LOWER(status) LIKE ? OR LOWER(external_payment_id) LIKE ? OR LOWER(external_checkout_id) LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "created_at DESC, id DESC"
	switch strings.TrimSpace(filter.Sort) {
	case "oldest", "created_asc":
		order = "created_at ASC, id ASC"
	case "paid_desc":
		order = "paid_at DESC NULLS LAST, id DESC"
	case "amount_desc":
		order = "pay_amount_cents DESC, id DESC"
	}
	if r.sqliteDialect() && strings.TrimSpace(filter.Sort) == "paid_desc" {
		order = "paid_at IS NULL ASC, paid_at DESC, id DESC"
	}
	if err := query.Order(order).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.PaymentOrder, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainPaymentOrder(item))
	}
	return results, total, nil
}

// UpdatePaymentOrderStatus 更新待处理支付单状态。
func (r *Repo) UpdatePaymentOrderStatus(ctx context.Context, orderNo string, status string) (*domainbilling.PaymentOrder, error) {
	orderNo = strings.TrimSpace(orderNo)
	status = strings.TrimSpace(status)
	if orderNo == "" || (status != domainbilling.PaymentStatusFailed && status != domainbilling.PaymentStatusExpired) {
		return nil, repository.ErrInvalidInput
	}
	var result domainbilling.PaymentOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		if order.Status == domainbilling.PaymentStatusPaid {
			return repository.ErrInvalidInput
		}
		if err := tx.Model(&order).Update("status", status).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		result = toDomainPaymentOrder(order)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// MarkPaymentOrderPaidAndGrantSubscription 标记支付成功并发放订阅权益，重复回调保持幂等。
func (r *Repo) MarkPaymentOrderPaidAndGrantSubscription(
	ctx context.Context,
	orderNo string,
	externalPaymentID string,
	paidAt time.Time,
	subscription *domainbilling.Subscription,
) (*domainbilling.PaymentOrder, bool, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" || subscription == nil {
		return nil, false, repository.ErrInvalidInput
	}
	if paidAt.IsZero() {
		paidAt = time.Now()
	}

	var result domainbilling.PaymentOrder
	activated := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		if order.Status == domainbilling.PaymentStatusPaid {
			result = toDomainPaymentOrder(order)
			return nil
		}
		if order.Status != domainbilling.PaymentStatusPending {
			return repository.ErrInvalidInput
		}
		if order.OrderType != "" && order.OrderType != domainbilling.PaymentOrderTypeSubscription {
			return repository.ErrInvalidInput
		}
		if order.ExpiredAt != nil && order.ExpiredAt.Before(paidAt) {
			if err := tx.Model(&order).Updates(map[string]interface{}{
				"status": domainbilling.PaymentStatusExpired,
			}).Error; err != nil {
				return translateError(err)
			}
			return repository.ErrInvalidInput
		}

		var plan model.BillingPlan
		if err := tx.Where("id = ? AND is_active = ?", subscription.PlanID, true).First(&plan).Error; err != nil {
			return translateError(err)
		}
		var price model.BillingPrice
		if err := tx.Where("id = ? AND plan_id = ? AND is_active = ?", subscription.PriceID, subscription.PlanID, true).First(&price).Error; err != nil {
			return translateError(err)
		}
		if subscription.CurrentPeriodEndAt == nil {
			return repository.ErrInvalidInput
		}
		duration := subscription.CurrentPeriodEndAt.Sub(paidAt)
		if duration <= 0 {
			return repository.ErrInvalidInput
		}
		if _, err := grantSubscriptionOnTimeline(tx, subscriptionTimelineGrantRequest{
			UserID:            subscription.UserID,
			Plan:              plan,
			Price:             price,
			StartAt:           paidAt,
			Duration:          duration,
			CancelAtPeriodEnd: subscription.CancelAtPeriodEnd,
			AutoRenew:         subscription.AutoRenew,
			NewGrant:          true,
		}); err != nil {
			return err
		}

		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":              domainbilling.PaymentStatusPaid,
			"external_payment_id": strings.TrimSpace(externalPaymentID),
			"paid_at":             paidAt,
		}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		result = toDomainPaymentOrder(order)
		activated = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &result, activated, nil
}

// AddUsage 写入账本。
func (r *Repo) AddUsage(ctx context.Context, usage *domainbilling.UsageLedger) error {
	if usage == nil {
		return nil
	}
	record := toModelUsageLedger(usage)
	return r.db.WithContext(ctx).Create(&record).Error
}

// AddUsageAndDebitBalance 写入用量并按实际金额扣减余额。
func (r *Repo) AddUsageAndDebitBalance(ctx context.Context, usage *domainbilling.UsageLedger) error {
	if usage == nil {
		return nil
	}
	return r.AddUsageAndSettleBalance(ctx, usage, nil)
}

// AddUsageAndSettleBalance 写入用量，并结算预扣差额。
func (r *Repo) AddUsageAndSettleBalance(ctx context.Context, usage *domainbilling.UsageLedger, reservation *domainbilling.UsageBalanceReservation) error {
	if usage == nil {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toModelUsageLedger(usage)
		chargeNanousd := usage.BilledNanousd
		if usage.IsFreeModel || chargeNanousd <= 0 {
			chargeNanousd = 0
		}
		reservedNanousd := int64(0)
		if reservation != nil {
			reservedNanousd = reservation.AmountNanousd
			if reservedNanousd < 0 {
				return repository.ErrInvalidInput
			}
		}
		deltaNanousd := chargeNanousd - reservedNanousd
		needsBalanceChange := deltaNanousd != 0

		var account *model.BillingAccount
		if needsBalanceChange {
			var err error
			account, err = getOrCreateBillingAccountForUpdate(tx, usage.UserID)
			if err != nil {
				return err
			}
			if deltaNanousd > 0 && account.BalanceNanousd < deltaNanousd {
				return repository.ErrInsufficientBalance
			}
		}

		if err := tx.Create(&record).Error; err != nil {
			return translateError(err)
		}
		if !needsBalanceChange {
			return nil
		}

		nextBalance := account.BalanceNanousd - deltaNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transactionType := domainbilling.BalanceTransactionTypeUsage
		description := "按量模型用量扣费"
		if deltaNanousd < 0 {
			transactionType = domainbilling.BalanceTransactionTypeUsageRefund
			description = "按量模型预扣差额退回"
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              usage.UserID,
			Type:                transactionType,
			AmountNanousd:       -deltaNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "usage_ledger",
			RefID:               record.ID,
			RefNo:               reservationRefNo(reservation),
			Description:         description,
		}
		return translateError(tx.Create(&transaction).Error)
	})
}

// AddPeriodUsageAndSettleOverage 写入周期用量，并将超出套餐额度的部分按量结算。
func (r *Repo) AddPeriodUsageAndSettleOverage(
	ctx context.Context,
	usage *domainbilling.UsageLedger,
	periodStart time.Time,
	periodEnd time.Time,
	periodCreditNanousd int64,
	reservation *domainbilling.UsageBalanceReservation,
) error {
	if usage == nil {
		return nil
	}
	if usage.UserID == 0 || periodStart.IsZero() || !periodEnd.After(periodStart) || periodCreditNanousd < 0 {
		return repository.ErrInvalidInput
	}
	settledSnapshotJSON := ""
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 余额账户行锁是周期额度结算的串行化锚点，确保同一用户并发请求按提交顺序重算已用额度。
		account, err := getOrCreateBillingAccountForUpdate(tx, usage.UserID)
		if err != nil {
			return err
		}

		var usedBeforeNanousd int64
		if err = tx.Model(&model.UsageLedger{}).
			Select("COALESCE(SUM(billed_nanousd), 0)").
			Where("user_id = ? AND is_free_model = ? AND billing_at >= ? AND billing_at < ?", usage.UserID, false, periodStart, periodEnd).
			Scan(&usedBeforeNanousd).Error; err != nil {
			return translateError(err)
		}

		chargeNanousd := usage.BilledNanousd
		if usage.IsFreeModel || chargeNanousd <= 0 {
			chargeNanousd = 0
		}
		remainingNanousd := periodCreditNanousd - usedBeforeNanousd
		if remainingNanousd < 0 {
			remainingNanousd = 0
		}
		coveredNanousd := minInt64(chargeNanousd, remainingNanousd)
		overageNanousd := chargeNanousd - coveredNanousd
		reservedNanousd := int64(0)
		if reservation != nil {
			reservedNanousd = reservation.AmountNanousd
			if reservedNanousd < 0 {
				return repository.ErrInvalidInput
			}
		}
		deltaNanousd := overageNanousd - reservedNanousd
		if deltaNanousd > 0 && account.BalanceNanousd < deltaNanousd {
			return repository.ErrInsufficientBalance
		}

		ledger := *usage
		ledger.PricingSnapshotJSON = withPeriodSettlementSnapshot(ledger.PricingSnapshotJSON, map[string]interface{}{
			"period_credit_nanousd":                   periodCreditNanousd,
			"period_used_before_nanousd":              usedBeforeNanousd,
			"period_used_after_nanousd":               usedBeforeNanousd + chargeNanousd,
			"period_credit_covered_nanousd":           coveredNanousd,
			"period_overage_billed_nanousd":           overageNanousd,
			"period_balance_charged_nanousd":          overageNanousd,
			"period_balance_reserved_nanousd":         reservedNanousd,
			"period_balance_settlement_delta_nanousd": deltaNanousd,
		})
		record := toModelUsageLedger(&ledger)
		if err := tx.Create(&record).Error; err != nil {
			return translateError(err)
		}
		if deltaNanousd == 0 {
			settledSnapshotJSON = ledger.PricingSnapshotJSON
			return nil
		}

		nextBalance := account.BalanceNanousd - deltaNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transactionType := domainbilling.BalanceTransactionTypeUsage
		description := "周期套餐超额按量扣费"
		if deltaNanousd < 0 {
			transactionType = domainbilling.BalanceTransactionTypeUsageRefund
			description = "周期套餐超额预扣差额退回"
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              usage.UserID,
			Type:                transactionType,
			AmountNanousd:       -deltaNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "usage_ledger",
			RefID:               record.ID,
			RefNo:               reservationRefNo(reservation),
			Description:         description,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		settledSnapshotJSON = ledger.PricingSnapshotJSON
		return nil
	})
	if err != nil {
		return err
	}
	if settledSnapshotJSON != "" {
		usage.PricingSnapshotJSON = settledSnapshotJSON
	}
	return nil
}

// ReserveUsageBalance 在真实调用前预扣固定金额，避免并发请求透支余额。
func (r *Repo) ReserveUsageBalance(ctx context.Context, userID uint, amountNanousd int64, refNo string) (*domainbilling.UsageBalanceReservation, error) {
	refNo = strings.TrimSpace(refNo)
	if userID == 0 || amountNanousd < 0 || refNo == "" {
		return nil, repository.ErrInvalidInput
	}
	if amountNanousd == 0 {
		return nil, nil
	}
	var result *domainbilling.UsageBalanceReservation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := getOrCreateBillingAccountForUpdate(tx, userID)
		if err != nil {
			return err
		}
		var existing model.BalanceTransaction
		err = tx.Where("user_id = ? AND type = ? AND ref_no = ?", userID, domainbilling.BalanceTransactionTypeUsageReserve, refNo).
			First(&existing).Error
		if err == nil {
			result = &domainbilling.UsageBalanceReservation{
				UserID:        userID,
				AmountNanousd: -existing.AmountNanousd,
				RefNo:         refNo,
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}
		if account.BalanceNanousd < amountNanousd {
			return repository.ErrInsufficientBalance
		}
		nextBalance := account.BalanceNanousd - amountNanousd
		if err = tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              userID,
			Type:                domainbilling.BalanceTransactionTypeUsageReserve,
			AmountNanousd:       -amountNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "usage_reservation",
			RefNo:               refNo,
			Description:         "按量模型调用预扣",
		}
		if err = tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		result = &domainbilling.UsageBalanceReservation{
			UserID:        userID,
			AmountNanousd: amountNanousd,
			RefNo:         refNo,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ReleaseUsageBalanceReservation 在调用失败时退回预扣金额，重复调用保持幂等。
func (r *Repo) ReleaseUsageBalanceReservation(ctx context.Context, userID uint, refNo string, description string) error {
	refNo = strings.TrimSpace(refNo)
	if userID == 0 || refNo == "" {
		return repository.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := getOrCreateBillingAccountForUpdate(tx, userID)
		if err != nil {
			return err
		}
		var reserve model.BalanceTransaction
		if err = tx.Where("user_id = ? AND type = ? AND ref_no = ?", userID, domainbilling.BalanceTransactionTypeUsageReserve, refNo).
			First(&reserve).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return translateError(err)
		}
		var existingRefund model.BalanceTransaction
		err = tx.Where("user_id = ? AND type = ? AND ref_no = ?", userID, domainbilling.BalanceTransactionTypeUsageRefund, refNo).
			First(&existingRefund).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}
		refundNanousd := -reserve.AmountNanousd
		if refundNanousd <= 0 {
			return nil
		}
		nextBalance := account.BalanceNanousd + refundNanousd
		if err = tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              userID,
			Type:                domainbilling.BalanceTransactionTypeUsageRefund,
			AmountNanousd:       refundNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "usage_reservation",
			RefID:               reserve.ID,
			RefNo:               refNo,
			Description:         firstNonEmpty(strings.TrimSpace(description), "按量模型调用失败退回预扣"),
		}
		return translateError(tx.Create(&transaction).Error)
	})
}

func reservationRefNo(reservation *domainbilling.UsageBalanceReservation) string {
	if reservation == nil {
		return ""
	}
	return strings.TrimSpace(reservation.RefNo)
}

func withPeriodSettlementSnapshot(raw string, values map[string]interface{}) string {
	snapshot := map[string]interface{}{}
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		_ = json.Unmarshal([]byte(trimmed), &snapshot)
	}
	for key, value := range values {
		snapshot[key] = value
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return raw
	}
	return string(encoded)
}

// GetOrCreateBillingAccount 查询或创建用户按量余额账户。
func (r *Repo) GetOrCreateBillingAccount(ctx context.Context, userID uint) (*domainbilling.BillingAccount, error) {
	if userID == 0 {
		return nil, repository.ErrInvalidInput
	}
	var result *domainbilling.BillingAccount
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := getOrCreateBillingAccountForUpdate(tx, userID)
		if err != nil {
			return err
		}
		domain := toDomainBillingAccount(*account)
		result = &domain
		return nil
	})
	return result, err
}

// ListBillingAccountsByUserIDs 批量查询用户按量余额账户。
func (r *Repo) ListBillingAccountsByUserIDs(ctx context.Context, userIDs []uint) ([]domainbilling.BillingAccount, error) {
	if len(userIDs) == 0 {
		return []domainbilling.BillingAccount{}, nil
	}
	items := make([]model.BillingAccount, 0, len(userIDs))
	if err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	results := make([]domainbilling.BillingAccount, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainBillingAccount(item))
	}
	return results, nil
}

// SetBillingAccountBalance 设置用户按量余额并记录余额流水。
func (r *Repo) SetBillingAccountBalance(ctx context.Context, userID uint, balanceNanousd int64, refNo string, description string) (*domainbilling.BillingAccount, error) {
	if userID == 0 || balanceNanousd < 0 {
		return nil, repository.ErrInvalidInput
	}
	var result domainbilling.BillingAccount
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := getOrCreateBillingAccountForUpdate(tx, userID)
		if err != nil {
			return err
		}
		amount := balanceNanousd - account.BalanceNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": balanceNanousd,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		if amount != 0 {
			transaction := model.BalanceTransaction{
				AccountID:           account.ID,
				UserID:              userID,
				Type:                domainbilling.BalanceTransactionTypeAdminSet,
				AmountNanousd:       amount,
				BalanceAfterNanousd: balanceNanousd,
				RefType:             "admin",
				RefNo:               strings.TrimSpace(refNo),
				Description:         firstNonEmpty(strings.TrimSpace(description), "管理员设置余额"),
			}
			if err := tx.Create(&transaction).Error; err != nil {
				return translateError(err)
			}
		}
		if err := tx.Where("id = ?", account.ID).First(account).Error; err != nil {
			return translateError(err)
		}
		result = toDomainBillingAccount(*account)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdjustBillingAccountBalance 原子增减用户按量余额并记录流水。
func (r *Repo) AdjustBillingAccountBalance(ctx context.Context, userID uint, deltaNanousd int64, refNo string, description string) (*domainbilling.BillingAccount, *domainbilling.BalanceTransaction, error) {
	if userID == 0 || deltaNanousd == 0 {
		return nil, nil, repository.ErrInvalidInput
	}
	var result domainbilling.BillingAccount
	var transactionResult *domainbilling.BalanceTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := getOrCreateBillingAccountForUpdate(tx, userID)
		if err != nil {
			return err
		}
		nextBalance := account.BalanceNanousd + deltaNanousd
		if nextBalance < 0 {
			return repository.ErrInsufficientBalance
		}
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		txType := domainbilling.BalanceTransactionTypeAdminSet
		txDescription := strings.TrimSpace(description)
		if txDescription == "" {
			if deltaNanousd > 0 {
				txDescription = "管理员增加余额"
			} else {
				txDescription = "管理员扣减余额"
			}
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              userID,
			Type:                txType,
			AmountNanousd:       deltaNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "admin_delta",
			RefNo:               strings.TrimSpace(refNo),
			Description:         txDescription,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("id = ?", account.ID).First(account).Error; err != nil {
			return translateError(err)
		}
		result = toDomainBillingAccount(*account)
		domainTx := toDomainBalanceTransaction(transaction)
		transactionResult = &domainTx
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &result, transactionResult, nil
}

// ListBalanceTransactions 分页查询余额流水。
func (r *Repo) ListBalanceTransactions(ctx context.Context, filter repository.BalanceTransactionListFilter, offset int, limit int) ([]domainbilling.BalanceTransaction, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	items := make([]model.BalanceTransaction, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.BalanceTransaction{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if txType := strings.TrimSpace(filter.Type); txType != "" {
		query = query.Where("type = ?", txType)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", *filter.To)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(ref_no) LIKE ? OR LOWER(description) LIKE ? OR LOWER(ref_type) LIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "created_at DESC, id DESC"
	if strings.TrimSpace(filter.Sort) == "oldest" {
		order = "created_at ASC, id ASC"
	}
	if err := query.Order(order).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.BalanceTransaction, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainBalanceTransaction(item))
	}
	return results, total, nil
}

// ClaimDailyCheckIn 原子写入每日签到、余额账户变更和余额流水；同日重复请求幂等返回原记录。
func (r *Repo) ClaimDailyCheckIn(ctx context.Context, input repository.CheckInClaimInput) (*repository.CheckInClaimResult, error) {
	if input.UserID == 0 || input.RewardNanousd <= 0 {
		return nil, repository.ErrInvalidInput
	}
	checkInDate := dateOnly(input.CheckInDate)
	if checkInDate.IsZero() {
		checkInDate = dateOnly(time.Now())
	}
	refNo := strings.TrimSpace(input.RefNo)
	if refNo == "" {
		refNo = "checkin"
	}
	consecutiveDays := input.ConsecutiveDay
	if consecutiveDays <= 0 {
		consecutiveDays = 1
	}

	var result repository.CheckInClaimResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.CheckInRecord
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND check_in_date = ?", input.UserID, checkInDate).
			First(&existing).Error
		if err == nil {
			result.Record = toDomainCheckInRecord(existing)
			if account, accountErr := billingAccountByUserID(tx, input.UserID); accountErr == nil {
				domainAccount := toDomainBillingAccount(*account)
				result.Account = &domainAccount
			} else if !errors.Is(accountErr, repository.ErrNotFound) {
				return accountErr
			}
			if existing.BalanceTransactionID > 0 {
				var transaction model.BalanceTransaction
				if txErr := tx.Where("id = ?", existing.BalanceTransactionID).First(&transaction).Error; txErr == nil {
					domainTx := toDomainBalanceTransaction(transaction)
					result.Transaction = &domainTx
				} else if !errors.Is(txErr, gorm.ErrRecordNotFound) {
					return translateError(txErr)
				}
			}
			result.AlreadyClaimed = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}

		account, err := getOrCreateBillingAccountForUpdate(tx, input.UserID)
		if err != nil {
			return err
		}
		nextBalance := account.BalanceNanousd + input.RewardNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}

		record := model.CheckInRecord{
			UserID:          input.UserID,
			CheckInDate:     checkInDate,
			RewardNanousd:   input.RewardNanousd,
			ConsecutiveDays: consecutiveDays,
			RefNo:           refNo,
		}
		if err := tx.Create(&record).Error; err != nil {
			return translateError(err)
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              input.UserID,
			Type:                domainbilling.BalanceTransactionTypeCheckIn,
			AmountNanousd:       input.RewardNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             domainbilling.BalanceTransactionRefTypeCheckIn,
			RefID:               record.ID,
			RefNo:               refNo,
			Description:         firstNonEmpty(strings.TrimSpace(input.Description), "Daily check-in reward"),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Model(&record).Update("balance_transaction_id", transaction.ID).Error; err != nil {
			return translateError(err)
		}
		record.BalanceTransactionID = transaction.ID
		if err := tx.Where("id = ?", account.ID).First(account).Error; err != nil {
			return translateError(err)
		}

		result.Record = toDomainCheckInRecord(record)
		domainAccount := toDomainBillingAccount(*account)
		domainTx := toDomainBalanceTransaction(transaction)
		result.Account = &domainAccount
		result.Transaction = &domainTx
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLatestCheckIn 查询用户最近一次签到记录。
func (r *Repo) GetLatestCheckIn(ctx context.Context, userID uint) (*domainbilling.CheckInRecord, error) {
	if userID == 0 {
		return nil, repository.ErrInvalidInput
	}
	var record model.CheckInRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("check_in_date DESC, id DESC").
		First(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainCheckInRecord(record)
	return &result, nil
}

const checkInRewardSettingKey = "checkin_reward_nanousd"

// GetCheckInRewardNanousd returns the configured daily check-in reward.
func (r *Repo) GetCheckInRewardNanousd(ctx context.Context) (int64, error) {
	var item model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ?", "billing", checkInRewardSettingKey).
		First(&item).Error; err != nil {
		return 0, translateError(err)
	}
	value, err := strconv.ParseInt(strings.TrimSpace(item.Value), 10, 64)
	if err != nil || value <= 0 {
		return 0, repository.ErrInvalidInput
	}
	return value, nil
}

// SetCheckInRewardNanousd persists the configured daily check-in reward.
func (r *Repo) SetCheckInRewardNanousd(ctx context.Context, rewardNanousd int64) error {
	if rewardNanousd <= 0 {
		return repository.ErrInvalidInput
	}
	item := model.SystemSetting{
		Namespace:   "billing",
		Key:         checkInRewardSettingKey,
		Value:       strconv.FormatInt(rewardNanousd, 10),
		ValueType:   "int",
		Description: "每日签到奖励金额(纳美元)",
	}
	return translateError(r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "namespace"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "description", "updated_at"}),
		}).
		Create(&item).Error)
}

// GetAdminCheckInStats returns aggregate check-in metrics for admin pages.
func (r *Repo) GetAdminCheckInStats(ctx context.Context, activeSince time.Time) (*domainbilling.AdminCheckInStats, error) {
	stats := &domainbilling.AdminCheckInStats{}
	todaySQL := `
		SELECT COALESCE(count(DISTINCT user_id), 0)
		FROM billing_checkin_records
		WHERE deleted_at IS NULL
		  AND check_in_date = CAST(? AS date)`
	if err := r.db.WithContext(ctx).Raw(todaySQL, activeSince.AddDate(0, 0, 6)).Scan(&stats.TodayCheckIns).Error; err != nil {
		return nil, translateError(err)
	}
	activeSQL := `
		SELECT COALESCE(count(DISTINCT user_id), 0)
		FROM billing_checkin_records
		WHERE deleted_at IS NULL
		  AND check_in_date >= CAST(? AS date)`
	if err := r.db.WithContext(ctx).Raw(activeSQL, activeSince).Scan(&stats.ActiveUsersLast7Days).Error; err != nil {
		return nil, translateError(err)
	}
	totalSQL := `
		SELECT
			COALESCE(count(*), 0) AS total_claims,
			COALESCE(sum(reward_nanousd), 0) AS total_reward_nanousd,
			COALESCE(avg(consecutive_days), 0) AS average_consecutive_days
		FROM billing_checkin_records
		WHERE deleted_at IS NULL`
	if err := r.db.WithContext(ctx).Raw(totalSQL).Scan(stats).Error; err != nil {
		return nil, translateError(err)
	}
	return stats, nil
}

// GetExternalAccountLink 查询当前用户指定外部平台绑定。
func (r *Repo) GetExternalAccountLink(ctx context.Context, userID uint, platform string) (*domainbilling.ExternalAccountLink, error) {
	if userID == 0 || strings.TrimSpace(platform) == "" {
		return nil, repository.ErrInvalidInput
	}
	var item model.ExternalAccountLink
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND platform = ?", userID, strings.TrimSpace(platform)).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainExternalAccountLink(item)
	return &result, nil
}

// FindUserLinuxDOSub 从用户已绑定身份中查找 LinuxDo subject。
func (r *Repo) FindUserLinuxDOSub(ctx context.Context, userID uint) (string, error) {
	if userID == 0 {
		return "", repository.ErrInvalidInput
	}
	type row struct {
		ProviderSubject string
	}
	var item row
	err := r.db.WithContext(ctx).
		Table("identity_user_links AS links").
		Select("links.provider_subject").
		Joins("JOIN identity_providers AS providers ON providers.id = links.provider_id").
		Where("links.user_id = ?", userID).
		Where("(LOWER(providers.slug) LIKE ? OR LOWER(providers.name) LIKE ?)", "%linuxdo%", "%linuxdo%").
		Order("links.linked_at DESC, links.id DESC").
		Limit(1).
		Scan(&item).Error
	if err != nil {
		return "", translateError(err)
	}
	if strings.TrimSpace(item.ProviderSubject) == "" {
		return "", repository.ErrNotFound
	}
	return strings.TrimSpace(item.ProviderSubject), nil
}

// UpsertExternalAccountLink 创建或更新外部账号绑定。
func (r *Repo) UpsertExternalAccountLink(ctx context.Context, link *domainbilling.ExternalAccountLink) (*domainbilling.ExternalAccountLink, error) {
	if link == nil || link.UserID == 0 || strings.TrimSpace(link.Platform) == "" || strings.TrimSpace(link.ExternalUserID) == "" {
		return nil, repository.ErrInvalidInput
	}
	now := time.Now().UTC()
	if !link.LinkedAt.IsZero() {
		now = link.LinkedAt
	}
	var result domainbilling.ExternalAccountLink
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.ExternalAccountLink
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND platform = ?", link.UserID, strings.TrimSpace(link.Platform)).
			First(&item).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}
		values := map[string]interface{}{
			"external_user_id":      strings.TrimSpace(link.ExternalUserID),
			"external_display_name": strings.TrimSpace(link.ExternalDisplayName),
			"linux_do_sub":          strings.TrimSpace(link.LinuxDOSub),
			"status":                firstNonEmpty(strings.TrimSpace(link.Status), domainbilling.ExternalAccountLinkStatusActive),
			"linked_at":             now,
			"last_synced_at":        now,
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			item = model.ExternalAccountLink{
				UserID:              link.UserID,
				Platform:            strings.TrimSpace(link.Platform),
				ExternalUserID:      strings.TrimSpace(link.ExternalUserID),
				ExternalDisplayName: strings.TrimSpace(link.ExternalDisplayName),
				LinuxDOSub:          strings.TrimSpace(link.LinuxDOSub),
				Status:              firstNonEmpty(strings.TrimSpace(link.Status), domainbilling.ExternalAccountLinkStatusActive),
				LinkedAt:            now,
				LastSyncedAt:        &now,
			}
			if err := tx.Create(&item).Error; err != nil {
				return translateError(err)
			}
			result = toDomainExternalAccountLink(item)
			return nil
		}
		if err := tx.Model(&item).Updates(values).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("id = ?", item.ID).First(&item).Error; err != nil {
			return translateError(err)
		}
		result = toDomainExternalAccountLink(item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreditExternalTransfer 幂等地将外部划转入账到本站余额。
func (r *Repo) CreditExternalTransfer(ctx context.Context, input repository.ExternalTransferCreditInput) (*domainbilling.ExternalTransfer, *domainbilling.BalanceTransaction, error) {
	if input.UserID == 0 || strings.TrimSpace(input.Platform) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExternalAmountUSD <= 0 || input.CreditedAmountNanousd <= 0 {
		return nil, nil, repository.ErrInvalidInput
	}
	var result domainbilling.ExternalTransfer
	var transactionResult *domainbilling.BalanceTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ExternalTransfer
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("idempotency_key = ?", strings.TrimSpace(input.IdempotencyKey)).
			First(&existing).Error
		if err == nil {
			result = toDomainExternalTransfer(existing)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return translateError(err)
		}

		account, err := getOrCreateBillingAccountForUpdate(tx, input.UserID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		transfer := model.ExternalTransfer{
			UserID:                input.UserID,
			LinkID:                input.LinkID,
			Platform:              strings.TrimSpace(input.Platform),
			Direction:             firstNonEmpty(strings.TrimSpace(input.Direction), domainbilling.ExternalTransferDirectionIn),
			ExternalTransferID:    strings.TrimSpace(input.ExternalTransferID),
			IdempotencyKey:        strings.TrimSpace(input.IdempotencyKey),
			ExternalAmountUSD:     input.ExternalAmountUSD,
			CreditedAmountNanousd: input.CreditedAmountNanousd,
			Status:                domainbilling.ExternalTransferStatusPending,
			RequestedAt:           now,
		}
		if err := tx.Create(&transfer).Error; err != nil {
			return translateError(err)
		}
		nextBalance := account.BalanceNanousd + input.CreditedAmountNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              input.UserID,
			Type:                domainbilling.BalanceTransactionTypeNewAPITransferIn,
			AmountNanousd:       input.CreditedAmountNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             domainbilling.BalanceTransactionRefTypeExternalTransfer,
			RefID:               transfer.ID,
			RefNo:               firstNonEmpty(strings.TrimSpace(input.RefNo), strings.TrimSpace(input.IdempotencyKey)),
			Description:         firstNonEmpty(strings.TrimSpace(input.Description), "NewAPI transfer in"),
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Model(&transfer).Updates(map[string]interface{}{
			"status":                 domainbilling.ExternalTransferStatusCredited,
			"balance_transaction_id": transaction.ID,
			"completed_at":           now,
		}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("id = ?", transfer.ID).First(&transfer).Error; err != nil {
			return translateError(err)
		}
		result = toDomainExternalTransfer(transfer)
		domainTx := toDomainBalanceTransaction(transaction)
		transactionResult = &domainTx
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &result, transactionResult, nil
}

// ListExternalTransfers 分页查询外部划转记录。
func (r *Repo) ListExternalTransfers(ctx context.Context, filter repository.ExternalTransferListFilter, offset int, limit int) ([]domainbilling.ExternalTransfer, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	items := make([]model.ExternalTransfer, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.ExternalTransfer{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if platform := strings.TrimSpace(filter.Platform); platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "created_at DESC, id DESC"
	if strings.TrimSpace(filter.Sort) == "oldest" {
		order = "created_at ASC, id ASC"
	}
	if err := query.Order(order).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.ExternalTransfer, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainExternalTransfer(item))
	}
	return results, total, nil
}

// UpdateExternalTransferStatus 更新外部划转状态。
func (r *Repo) UpdateExternalTransferStatus(ctx context.Context, idempotencyKey string, status string, failureReason string) error {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" || strings.TrimSpace(status) == "" {
		return repository.ErrInvalidInput
	}
	updates := map[string]interface{}{
		"status":         strings.TrimSpace(status),
		"failure_reason": strings.TrimSpace(failureReason),
	}
	if status == domainbilling.ExternalTransferStatusRolledBack || status == domainbilling.ExternalTransferStatusCredited {
		now := time.Now().UTC()
		updates["completed_at"] = now
	}
	result := r.db.WithContext(ctx).Model(&model.ExternalTransfer{}).Where("idempotency_key = ?", key).Updates(updates)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// MarkPaymentOrderPaidAndCreditBalance 标记充值支付成功并入账余额，重复回调保持幂等。
func (r *Repo) MarkPaymentOrderPaidAndCreditBalance(
	ctx context.Context,
	orderNo string,
	externalPaymentID string,
	paidAt time.Time,
) (*domainbilling.PaymentOrder, bool, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, false, repository.ErrInvalidInput
	}
	if paidAt.IsZero() {
		paidAt = time.Now()
	}

	var result domainbilling.PaymentOrder
	credited := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		if order.Status == domainbilling.PaymentStatusPaid {
			result = toDomainPaymentOrder(order)
			return nil
		}
		if order.Status != domainbilling.PaymentStatusPending || order.OrderType != domainbilling.PaymentOrderTypeTopUp || order.CreditNanousd <= 0 {
			return repository.ErrInvalidInput
		}
		if order.ExpiredAt != nil && order.ExpiredAt.Before(paidAt) {
			if err := tx.Model(&order).Updates(map[string]interface{}{
				"status": domainbilling.PaymentStatusExpired,
			}).Error; err != nil {
				return translateError(err)
			}
			return repository.ErrInvalidInput
		}

		account, err := getOrCreateBillingAccountForUpdate(tx, order.UserID)
		if err != nil {
			return err
		}
		nextBalance := account.BalanceNanousd + order.CreditNanousd
		if err := tx.Model(account).Updates(map[string]interface{}{
			"balance_nanousd": nextBalance,
			"currency":        "USD",
			"status":          "active",
		}).Error; err != nil {
			return translateError(err)
		}
		transaction := model.BalanceTransaction{
			AccountID:           account.ID,
			UserID:              order.UserID,
			Type:                domainbilling.BalanceTransactionTypeTopUp,
			AmountNanousd:       order.CreditNanousd,
			BalanceAfterNanousd: nextBalance,
			RefType:             "payment_order",
			RefID:               order.ID,
			RefNo:               order.OrderNo,
			Description:         "按量余额充值",
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":              domainbilling.PaymentStatusPaid,
			"external_payment_id": strings.TrimSpace(externalPaymentID),
			"paid_at":             paidAt,
		}).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return translateError(err)
		}
		result = toDomainPaymentOrder(order)
		credited = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &result, credited, nil
}

// ListRedemptionCodes 分页查询后台兑换码定义。
func (r *Repo) ListRedemptionCodes(ctx context.Context, filter repository.RedemptionCodeListFilter, offset int, limit int) ([]domainbilling.RedemptionCode, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	items := make([]model.RedemptionCode, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.RedemptionCode{})
	if len(filter.Modes) > 0 {
		modes := make([]string, 0, len(filter.Modes))
		for _, mode := range filter.Modes {
			if normalized := normalizeRedemptionMode(mode); normalized != "" {
				modes = append(modes, normalized)
			}
		}
		if len(modes) == 0 {
			return []domainbilling.RedemptionCode{}, 0, nil
		}
		query = query.Where("mode IN ?", modes)
	} else if mode := strings.TrimSpace(filter.Mode); mode != "" {
		query = query.Where("mode = ?", mode)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", domainbilling.RedemptionCodeStatusDeleted)
	}
	if availability := strings.TrimSpace(filter.Availability); availability != "" {
		now := time.Now()
		switch availability {
		case "available":
			query = query.
				Where("status = ?", domainbilling.RedemptionCodeStatusActive).
				Where("(expires_at IS NULL OR expires_at > ?)", now).
				Where("(max_redemptions IS NULL OR redeemed_count < max_redemptions)")
		case "expired":
			query = query.Where("expires_at IS NOT NULL AND expires_at <= ?", now)
		case "exhausted":
			query = query.Where("max_redemptions IS NOT NULL AND redeemed_count >= max_redemptions")
		}
	}
	if keyword := strings.TrimSpace(filter.Query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(description) LIKE ? OR LOWER(code_hint) LIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.RedemptionCode, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainRedemptionCode(item))
	}
	return results, total, nil
}

// GetRedemptionCodeByID 查询单个未删除兑换码定义。
func (r *Repo) GetRedemptionCodeByID(ctx context.Context, id uint) (*domainbilling.RedemptionCode, error) {
	if id == 0 {
		return nil, repository.ErrInvalidInput
	}
	var item model.RedemptionCode
	if err := r.db.WithContext(ctx).
		Where("id = ? AND status <> ?", id, domainbilling.RedemptionCodeStatusDeleted).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainRedemptionCode(item)
	return &result, nil
}

// CreateRedemptionCode 创建兑换码定义。
func (r *Repo) CreateRedemptionCode(ctx context.Context, item *domainbilling.RedemptionCode) (*domainbilling.RedemptionCode, error) {
	if item == nil || strings.TrimSpace(item.CodeHash) == "" {
		return nil, repository.ErrInvalidInput
	}
	record := model.RedemptionCode{
		CodeHash:        strings.TrimSpace(item.CodeHash),
		CodeEncrypted:   strings.TrimSpace(item.CodeEncrypted),
		CodeHint:        strings.TrimSpace(item.CodeHint),
		Mode:            normalizeRedemptionMode(item.Mode),
		RewardType:      normalizeRedemptionRewardType(item.RewardType),
		CreditNanousd:   clampNonNegative(item.CreditNanousd),
		PlanID:          item.PlanID,
		DurationDays:    item.DurationDays,
		MaxRedemptions:  copyIntPointer(item.MaxRedemptions),
		PerUserLimit:    item.PerUserLimit,
		Status:          normalizeRedemptionStatus(item.Status),
		ExpiresAt:       item.ExpiresAt,
		Description:     strings.TrimSpace(item.Description),
		CreatedByUserID: item.CreatedByUserID,
	}
	if record.PerUserLimit <= 0 {
		record.PerUserLimit = 1
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainRedemptionCode(record)
	return &result, nil
}

// PatchRedemptionCode 更新兑换码管理字段。
func (r *Repo) PatchRedemptionCode(ctx context.Context, id uint, patch repository.RedemptionCodePatch) (*domainbilling.RedemptionCode, error) {
	if id == 0 {
		return nil, repository.ErrInvalidInput
	}
	var result domainbilling.RedemptionCode
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record model.RedemptionCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status <> ?", id, domainbilling.RedemptionCodeStatusDeleted).
			First(&record).Error; err != nil {
			return translateError(err)
		}
		updates := map[string]interface{}{}
		if patch.Status != nil {
			status := normalizeRedemptionStatus(*patch.Status)
			if status == "" {
				return repository.ErrInvalidInput
			}
			updates["status"] = status
		}
		if patch.MaxRedemptionsSet {
			if patch.MaxRedemptions != nil {
				if *patch.MaxRedemptions <= 0 || *patch.MaxRedemptions < record.RedeemedCount {
					return repository.ErrInvalidInput
				}
			}
			updates["max_redemptions"] = patch.MaxRedemptions
		}
		if patch.PerUserLimit != nil {
			if *patch.PerUserLimit <= 0 {
				return repository.ErrInvalidInput
			}
			updates["per_user_limit"] = *patch.PerUserLimit
		}
		nextMaxRedemptions := record.MaxRedemptions
		if patch.MaxRedemptionsSet {
			nextMaxRedemptions = patch.MaxRedemptions
		}
		nextPerUserLimit := record.PerUserLimit
		if patch.PerUserLimit != nil {
			nextPerUserLimit = *patch.PerUserLimit
		}
		if nextMaxRedemptions != nil && nextPerUserLimit > *nextMaxRedemptions {
			return repository.ErrInvalidInput
		}
		if patch.ExpiresAtSet {
			updates["expires_at"] = patch.ExpiresAt
		}
		if patch.Description != nil {
			updates["description"] = strings.TrimSpace(*patch.Description)
		}
		if len(updates) > 0 {
			if err := tx.Model(&record).Updates(updates).Error; err != nil {
				return translateError(err)
			}
		}
		if err := tx.Where("id = ?", id).First(&record).Error; err != nil {
			return translateError(err)
		}
		result = toDomainRedemptionCode(record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteRedemptionCode 软删除兑换码定义，保留兑换记录审计。
func (r *Repo) DeleteRedemptionCode(ctx context.Context, id uint) error {
	if id == 0 {
		return repository.ErrInvalidInput
	}
	result := r.db.WithContext(ctx).
		Model(&model.RedemptionCode{}).
		Where("id = ? AND status <> ?", id, domainbilling.RedemptionCodeStatusDeleted).
		Update("status", domainbilling.RedemptionCodeStatusDeleted)
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// RedeemCode 原子校验兑换码并写入奖励、兑换记录和次数。
func (r *Repo) RedeemCode(ctx context.Context, input repository.RedemptionApplyInput) (*repository.RedemptionApplyResult, error) {
	codeHash := strings.TrimSpace(input.CodeHash)
	if codeHash == "" || input.UserID == 0 {
		return nil, repository.ErrInvalidInput
	}
	now := input.SubscriptionAt
	if now.IsZero() {
		now = time.Now()
	}
	var result repository.RedemptionApplyResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var code model.RedemptionCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_hash = ?", codeHash).
			First(&code).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrRedemptionUnavailable
			}
			return translateError(err)
		}
		if err := validateRedeemableCode(tx, code, input.UserID, strings.TrimSpace(input.CurrentMode), now); err != nil {
			return err
		}

		redemption := model.Redemption{
			CodeID:        code.ID,
			UserID:        input.UserID,
			Mode:          code.Mode,
			RewardType:    code.RewardType,
			CreditNanousd: code.CreditNanousd,
			PlanID:        code.PlanID,
			RefNo:         strings.TrimSpace(input.RefNo),
			SnapshotJSON:  redemptionSnapshotJSON(code),
		}
		var accountDomain *domainbilling.BillingAccount
		var subscriptionDomain *domainbilling.Subscription

		switch code.RewardType {
		case domainbilling.RedemptionRewardTypeBalance:
			account, balanceTxID, applyErr := applyRedemptionBalance(tx, input.UserID, code, redemption.RefNo)
			if applyErr != nil {
				return applyErr
			}
			redemption.BalanceTransactionID = balanceTxID
			domain := toDomainBillingAccount(*account)
			accountDomain = &domain
		case domainbilling.RedemptionRewardTypeSubscription:
			subscription, applyErr := applyRedemptionSubscription(tx, input.UserID, code, now)
			if applyErr != nil {
				return applyErr
			}
			redemption.SubscriptionID = subscription.ID
			domain := toDomainSubscription(*subscription)
			subscriptionDomain = &domain
		default:
			return repository.ErrInvalidInput
		}

		if err := tx.Create(&redemption).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Model(&code).Update("redeemed_count", gorm.Expr("redeemed_count + ?", 1)).Error; err != nil {
			return translateError(err)
		}
		code.RedeemedCount++
		result = repository.RedemptionApplyResult{
			Code:         toDomainRedemptionCode(code),
			Redemption:   toDomainRedemption(redemption),
			Account:      accountDomain,
			Subscription: subscriptionDomain,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRedemptions 分页查询兑换记录。
func (r *Repo) ListRedemptions(ctx context.Context, filter repository.RedemptionListFilter, offset int, limit int) ([]domainbilling.Redemption, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	items := make([]model.Redemption, 0, limit)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Redemption{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" {
		query = query.Where("mode = ?", mode)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(ref_no) LIKE ? OR LOWER(snapshot_json) LIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "created_at DESC, id DESC"
	if strings.TrimSpace(filter.Sort) == "oldest" {
		order = "created_at ASC, id ASC"
	}
	if err := query.Order(order).Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.Redemption, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainRedemption(item))
	}
	return results, total, nil
}

// GetBillingMode 查询当前计费方式。
func (r *Repo) GetBillingMode(ctx context.Context) (string, error) {
	var item model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ?", "billing", "mode").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "self", nil
		}
		return "", translateError(err)
	}
	mode := strings.TrimSpace(item.Value)
	switch mode {
	case "self", "usage", "period":
		return mode, nil
	default:
		return "self", nil
	}
}

// GetBillingPrepaidAmountNanousd 查询按量调用前要求保留的最低预付余额。
func (r *Repo) GetBillingPrepaidAmountNanousd(ctx context.Context) (int64, error) {
	var item model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ?", "billing", "prepaid_amount_usd").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, translateError(err)
	}
	value := strings.TrimSpace(item.Value)
	if value == "" {
		return 0, nil
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil || amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, repository.ErrInvalidInput
	}
	return int64(math.Round(amount * 1000000000)), nil
}

// GetNativeToolBillingEnabled 查询模型原生工具是否按官方默认价计费。
func (r *Repo) GetNativeToolBillingEnabled(ctx context.Context) (bool, error) {
	var item model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ?", "billing", "native_tool_billing_enabled").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, translateError(err)
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(item.Value))
	if err != nil {
		return false, repository.ErrInvalidInput
	}
	return enabled, nil
}

// GetNativeToolPricingJSON 查询模型原生工具计费覆盖配置。
func (r *Repo) GetNativeToolPricingJSON(ctx context.Context) (string, error) {
	var item model.SystemSetting
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ?", "billing", "native_tool_pricing_json").
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", translateError(err)
	}
	return strings.TrimSpace(item.Value), nil
}

// GetModelPricing 查询模型计费配置。
func (r *Repo) GetModelPricing(ctx context.Context, platformModelName string) (*domainbilling.ModelPricing, error) {
	var item model.ModelPricing
	if err := r.db.WithContext(ctx).
		Where("platform_model_name = ?", strings.TrimSpace(platformModelName)).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainModelPricing(item)
	return &result, nil
}

// ListModelPricing 分页查询模型单价。
func (r *Repo) ListModelPricing(ctx context.Context, query string, offset int, limit int) ([]domainbilling.ModelPricing, int64, error) {
	items := make([]model.ModelPricing, 0)
	var total int64

	dbq := r.db.WithContext(ctx).Model(&model.ModelPricing{})
	if keyword := strings.TrimSpace(query); keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		dbq = dbq.Where("LOWER(platform_model_name) LIKE ?", like)
	}

	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := dbq.Order("platform_model_name ASC, id ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}

	results := make([]domainbilling.ModelPricing, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainModelPricing(item))
	}
	return results, total, nil
}

// UpsertModelPricing 按平台模型名保存模型单价。
func (r *Repo) UpsertModelPricing(ctx context.Context, item *domainbilling.ModelPricing) (*domainbilling.ModelPricing, error) {
	if item == nil {
		return nil, repository.ErrInvalidInput
	}
	platformModelName := strings.TrimSpace(item.PlatformModelName)
	if platformModelName == "" {
		return nil, repository.ErrInvalidInput
	}

	var record model.ModelPricing
	err := r.db.WithContext(ctx).Where("platform_model_name = ?", platformModelName).First(&record).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, translateError(err)
	}

	updates := map[string]interface{}{
		"platform_model_name":              platformModelName,
		"currency":                         normalizeCurrency(item.Currency),
		"is_free":                          item.IsFree,
		"pricing_mode":                     normalizePricingMode(item.PricingMode),
		"input_nanousd_per_m_tokens":       clampNonNegative(item.InputNanousdPerMTokens),
		"cache_read_nanousd_per_m_tokens":  clampNonNegative(item.CacheReadNanousdPerMTokens),
		"cache_write_nanousd_per_m_tokens": clampNonNegative(item.CacheWriteNanousdPerMTokens),
		"output_nanousd_per_m_tokens":      clampNonNegative(item.OutputNanousdPerMTokens),
		"call_nanousd_per_call":            clampNonNegative(item.CallNanousdPerCall),
		"duration_nanousd_per_second":      clampNonNegative(item.DurationNanousdPerSecond),
		"tiered_pricing_json":              strings.TrimSpace(item.TieredPricingJSON),
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = model.ModelPricing{
			PlatformModelName: platformModelName,
		}
		if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
			return nil, translateError(err)
		}
	}
	if err := r.db.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		return nil, translateError(err)
	}
	if err := r.db.WithContext(ctx).Where("platform_model_name = ?", platformModelName).First(&record).Error; err != nil {
		return nil, translateError(err)
	}
	result := toDomainModelPricing(record)
	return &result, nil
}

// ListUsageByUser 分页查询账本。
func (r *Repo) ListUsageByUser(ctx context.Context, userID uint, filter repository.UsageListFilter, offset int, limit int) ([]domainbilling.UsageLedger, int64, error) {
	items := make([]model.UsageLedger, 0)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.UsageLedger{}).Where("user_id = ?", userID)
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(platform_model_name) LIKE ?", like)
	}
	switch strings.TrimSpace(filter.Status) {
	case "free":
		query = query.Where("is_free_model = ?", true)
	case "billable":
		query = query.Where("is_free_model = ?", false)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "usage_date DESC, id DESC"
	switch strings.TrimSpace(filter.Sort) {
	case "oldest":
		order = "usage_date ASC, id ASC"
	case "tokens_desc":
		order = "(input_tokens + cache_read_tokens + cache_write_tokens + output_tokens + reasoning_tokens) DESC, id DESC"
	case "cost_desc":
		order = "billed_nanousd DESC, id DESC"
	case "latency_desc":
		order = "latency_ms DESC, id DESC"
	}
	if err := query.
		Order(order).
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.UsageLedger, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainUsageLedger(item))
	}
	return results, total, nil
}

// ListUsageLogs 分页查询管理员调用日志。
func (r *Repo) ListUsageLogs(ctx context.Context, filter repository.UsageLogListFilter, offset int, limit int) ([]domainbilling.UsageLedger, int64, error) {
	items := make([]model.UsageLedger, 0)
	var total int64
	query := r.db.WithContext(ctx).Model(&model.UsageLedger{})
	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(platform_model_name) LIKE ? OR LOWER(upstream_model_name) LIKE ? OR LOWER(upstream_name) LIKE ? OR LOWER(routed_binding_code) LIKE ? OR LOWER(provider_protocol) LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}
	if platformModelName := strings.TrimSpace(filter.PlatformModelName); platformModelName != "" {
		query = query.Where("platform_model_name = ?", platformModelName)
	}
	switch strings.TrimSpace(filter.BillingMode) {
	case "free":
		query = query.Where("is_free_model = ?", true)
	case "token", "call", "duration", "tiered":
		query = query.Where("is_free_model = ?", false)
		query = query.Where(r.pricingModeExpression()+" = ?", strings.TrimSpace(filter.BillingMode))
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	order := "created_at DESC, id DESC"
	switch strings.TrimSpace(filter.Sort) {
	case "created_asc":
		order = "created_at ASC, id ASC"
	case "tokens_desc":
		order = "(input_tokens + cache_read_tokens + cache_write_tokens + output_tokens + reasoning_tokens) DESC, id DESC"
	case "cost_desc":
		order = "billed_nanousd DESC, id DESC"
	case "latency_desc":
		order = "latency_ms DESC, id DESC"
	}
	if err := query.
		Order(order).
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	results := make([]domainbilling.UsageLedger, 0, len(items))
	for _, item := range items {
		results = append(results, toDomainUsageLedger(item))
	}
	return results, total, nil
}

// GetAdminDashboardStats returns usage and payment aggregates for the admin dashboard.
func (r *Repo) GetAdminDashboardStats(ctx context.Context, startAt time.Time, endAt time.Time, limit int) (*domainbilling.AdminDashboardStats, error) {
	if limit <= 0 {
		limit = 8
	}
	stats := &domainbilling.AdminDashboardStats{}
	usageSQL := `
		SELECT
			COALESCE(count(*), 0) AS record_count,
			COALESCE(count(DISTINCT user_id), 0) AS active_user_count,
			COALESCE(sum(call_count), 0) AS call_count,
			COALESCE(sum(input_tokens + cache_read_tokens + cache_write_tokens + cache_write_5m_tokens + cache_write_1h_tokens + output_tokens + reasoning_tokens), 0) AS token_count,
			COALESCE(sum(duration_seconds), 0) AS duration_seconds,
			COALESCE(sum(billed_nanousd), 0) AS billed_nanousd
		FROM billing_usage_ledgers
		WHERE deleted_at IS NULL
		  AND usage_date >= CAST(? AS date)
		  AND usage_date < CAST(? AS date)`
	if err := r.db.WithContext(ctx).Raw(usageSQL, startAt, endAt).Scan(&stats.Usage).Error; err != nil {
		return nil, translateError(err)
	}

	salesSQL := `
		SELECT
			COALESCE(count(*), 0) AS paid_order_count,
			COALESCE(sum(base_amount_cents), 0) AS base_amount_cents,
			COALESCE(sum(credit_nanousd), 0) AS credit_nanousd
		FROM billing_payment_orders
		WHERE deleted_at IS NULL
		  AND status = 'paid'
		  AND paid_at >= ?
		  AND paid_at < ?`
	if err := r.db.WithContext(ctx).Raw(salesSQL, startAt, endAt).Scan(&stats.Sales).Error; err != nil {
		return nil, translateError(err)
	}

	modelSQL := `
		SELECT
			platform_model_name,
			COALESCE(count(*), 0) AS record_count,
			COALESCE(count(DISTINCT user_id), 0) AS active_user_count,
			COALESCE(sum(call_count), 0) AS call_count,
			COALESCE(sum(input_tokens + cache_read_tokens + cache_write_tokens + cache_write_5m_tokens + cache_write_1h_tokens + output_tokens + reasoning_tokens), 0) AS token_count,
			COALESCE(sum(duration_seconds), 0) AS duration_seconds,
			COALESCE(sum(billed_nanousd), 0) AS billed_nanousd
		FROM billing_usage_ledgers
		WHERE deleted_at IS NULL
		  AND usage_date >= CAST(? AS date)
		  AND usage_date < CAST(? AS date)
		GROUP BY platform_model_name
		ORDER BY COALESCE(sum(call_count), 0) DESC,
		         COALESCE(sum(input_tokens + cache_read_tokens + cache_write_tokens + cache_write_5m_tokens + cache_write_1h_tokens + output_tokens + reasoning_tokens), 0) DESC,
		         platform_model_name ASC
		LIMIT ?`
	if err := r.db.WithContext(ctx).Raw(modelSQL, startAt, endAt, limit).Scan(&stats.TopModels).Error; err != nil {
		return nil, translateError(err)
	}
	return stats, nil
}

// GetBillingRiskSummary returns the first billing-facing risk snapshot.
func (r *Repo) GetBillingRiskSummary(ctx context.Context) (*domainbilling.RiskSummary, error) {
	stats := &domainbilling.RiskSummary{GeneratedAt: time.Now()}
	if r.sqliteDialect() {
		return stats, nil
	}
	var tableName string
	if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(to_regclass('public.fingerprint_associations')::text, '')`).Scan(&tableName).Error; err != nil {
		return nil, translateError(err)
	}
	if strings.TrimSpace(tableName) != "" {
		if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(to_regclass('public.device_fingerprints')::text, '')`).Scan(&tableName).Error; err != nil {
			return nil, translateError(err)
		}
	}
	if strings.TrimSpace(tableName) != "" {
		clusterSQL := `
			WITH association_users AS (
				SELECT
					fa.id AS association_id,
					u.id AS user_id
				FROM fingerprint_associations fa
				LEFT JOIN LATERAL jsonb_array_elements_text(
					CASE
						WHEN jsonb_typeof(fa.user_ids_json::jsonb) = 'array' THEN fa.user_ids_json::jsonb
						ELSE '[]'::jsonb
					END
				) AS user_ids(user_id_text) ON true
				JOIN identity_users u
					ON u.id = CASE
						WHEN user_ids.user_id_text ~ '^[0-9]+$' THEN user_ids.user_id_text::bigint
						ELSE NULL
					END
					AND u.deleted_at IS NULL
			),
			corroborated_associations AS (
				SELECT DISTINCT au.association_id
				FROM association_users au
				JOIN fingerprint_associations fa ON fa.id = au.association_id
				JOIN device_fingerprints df
					ON df.fingerprint_id = fa.fingerprint_id
					AND df.user_id = au.user_id
					AND df.deleted_at IS NULL
					AND COALESCE(df.ip_address, '') <> ''
				GROUP BY au.association_id, df.ip_address
				HAVING count(DISTINCT df.user_id) = (
					SELECT count(DISTINCT inner_au.user_id)
					FROM association_users inner_au
					WHERE inner_au.association_id = au.association_id
				)
				UNION
				SELECT DISTINCT au.association_id
				FROM association_users au
				JOIN fingerprint_associations fa ON fa.id = au.association_id
				JOIN device_fingerprints df
					ON df.fingerprint_id = fa.fingerprint_id
					AND df.user_id = au.user_id
					AND df.deleted_at IS NULL
					AND COALESCE(df.tls_fingerprint, '') <> ''
				GROUP BY au.association_id, df.tls_fingerprint
				HAVING count(DISTINCT df.user_id) = (
					SELECT count(DISTINCT inner_au.user_id)
					FROM association_users inner_au
					WHERE inner_au.association_id = au.association_id
				)
			),
			corroborated AS (
				SELECT fa.*
				FROM fingerprint_associations fa
				WHERE fa.deleted_at IS NULL
					AND fa.id IN (SELECT association_id FROM corroborated_associations)
			)
			SELECT
				COALESCE(count(*) FILTER (WHERE ignored_at IS NULL), 0) AS multi_account_cluster_count,
				COALESCE(count(*) FILTER (WHERE ignored_at IS NULL AND (risk_level = 'high' OR confidence_score >= 0.9)), 0) AS high_risk_cluster_count,
				COALESCE(count(*) FILTER (WHERE ignored_at IS NOT NULL), 0) AS ignored_cluster_count,
				COALESCE(count(DISTINCT fingerprint_id), 0) AS unique_fingerprint_count
			FROM corroborated`
		if err := r.db.WithContext(ctx).Raw(clusterSQL).Scan(stats).Error; err != nil {
			return nil, translateError(err)
		}
	}
	var auditTable string
	if err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(to_regclass('public.audit_logs')::text, '')`).Scan(&auditTable).Error; err != nil {
		return nil, translateError(err)
	}
	if strings.TrimSpace(auditTable) != "" {
		if err := r.db.WithContext(ctx).
			Raw(`SELECT COALESCE(count(DISTINCT NULLIF(ip, '')), 0) FROM audit_logs WHERE deleted_at IS NULL`).
			Scan(&stats.UniqueIPCount).Error; err != nil {
			return nil, translateError(err)
		}
	}
	return stats, nil
}

// ListMonthlyUsageByUser 按月份聚合用户用量。
func (r *Repo) ListMonthlyUsageByUser(ctx context.Context, userID uint, limit int) ([]domainbilling.UsageMonthlySummary, error) {
	if limit <= 0 {
		limit = 6
	}
	if limit > 24 {
		limit = 24
	}
	now := time.Now()
	endMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, 1, 0)
	startMonth := endMonth.AddDate(0, -limit, 0)

	type monthlyUsageRow struct {
		MonthKey         string `gorm:"column:month_key"`
		RecordCount      int64  `gorm:"column:record_count"`
		InputTokens      int64  `gorm:"column:input_tokens"`
		CacheReadTokens  int64  `gorm:"column:cache_read_tokens"`
		CacheWriteTokens int64  `gorm:"column:cache_write_tokens"`
		OutputTokens     int64  `gorm:"column:output_tokens"`
		ReasoningTokens  int64  `gorm:"column:reasoning_tokens"`
		CallCount        int64  `gorm:"column:call_count"`
		DurationSeconds  int64  `gorm:"column:duration_seconds"`
		AvgLatencyMS     int64  `gorm:"column:avg_latency_ms"`
		BilledNanousd    int64  `gorm:"column:billed_nanousd"`
	}

	rows := make([]monthlyUsageRow, 0, limit)
	monthKeyExpression := r.usageMonthKeyExpression()
	if err := r.db.WithContext(ctx).
		Model(&model.UsageLedger{}).
		Select(monthKeyExpression+` AS month_key,
			COUNT(*) AS record_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COALESCE(SUM(call_count), 0) AS call_count,
			COALESCE(SUM(duration_seconds), 0) AS duration_seconds,
			COALESCE(ROUND(AVG(NULLIF(latency_ms, 0))), 0) AS avg_latency_ms,
			COALESCE(SUM(billed_nanousd), 0) AS billed_nanousd
		`).
		Where("user_id = ? AND usage_date >= ? AND usage_date < ?", userID, startMonth, endMonth).
		Group(monthKeyExpression).
		Order("month_key DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, translateError(err)
	}

	results := make([]domainbilling.UsageMonthlySummary, 0, len(rows))
	for _, row := range rows {
		monthStartAt, err := time.Parse("2006-01-02", row.MonthKey)
		if err != nil {
			return nil, err
		}
		results = append(results, domainbilling.UsageMonthlySummary{
			MonthStartAt:     monthStartAt,
			RecordCount:      row.RecordCount,
			InputTokens:      row.InputTokens,
			CacheReadTokens:  row.CacheReadTokens,
			CacheWriteTokens: row.CacheWriteTokens,
			OutputTokens:     row.OutputTokens,
			ReasoningTokens:  row.ReasoningTokens,
			CallCount:        row.CallCount,
			DurationSeconds:  row.DurationSeconds,
			AvgLatencyMS:     row.AvgLatencyMS,
			BilledNanousd:    row.BilledNanousd,
		})
	}
	return results, nil
}

// GetUserCreatedAt 查询用户注册时间。
func (r *Repo) GetUserCreatedAt(ctx context.Context, userID uint) (time.Time, error) {
	var item model.User
	if err := r.db.WithContext(ctx).
		Select("created_at").
		Where("id = ?", userID).
		First(&item).Error; err != nil {
		return time.Time{}, translateError(err)
	}
	return item.CreatedAt, nil
}

// ListDailyUsageByUser 按日期聚合用户用量。
func (r *Repo) ListDailyUsageByUser(ctx context.Context, userID uint, startDate time.Time, endDate time.Time) ([]domainbilling.UsageDailySummary, error) {
	type dailyModelUsageRow struct {
		UsageDateKey      string `gorm:"column:usage_date_key"`
		PlatformModelName string `gorm:"column:platform_model_name"`
		RecordCount       int64  `gorm:"column:record_count"`
		InputTokens       int64  `gorm:"column:input_tokens"`
		CacheReadTokens   int64  `gorm:"column:cache_read_tokens"`
		CacheWriteTokens  int64  `gorm:"column:cache_write_tokens"`
		OutputTokens      int64  `gorm:"column:output_tokens"`
		ReasoningTokens   int64  `gorm:"column:reasoning_tokens"`
		CallCount         int64  `gorm:"column:call_count"`
		DurationSeconds   int64  `gorm:"column:duration_seconds"`
		AvgLatencyMS      int64  `gorm:"column:avg_latency_ms"`
		LatencyCount      int64  `gorm:"column:latency_count"`
		BilledNanousd     int64  `gorm:"column:billed_nanousd"`
	}

	modelRows := make([]dailyModelUsageRow, 0)
	dayKeyExpression := r.usageDayKeyExpression()
	if err := r.db.WithContext(ctx).
		Model(&model.UsageLedger{}).
		Select(dayKeyExpression+` AS usage_date_key,
			platform_model_name,
			COUNT(*) AS record_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COALESCE(SUM(call_count), 0) AS call_count,
			COALESCE(SUM(duration_seconds), 0) AS duration_seconds,
			COALESCE(ROUND(AVG(NULLIF(latency_ms, 0))), 0) AS avg_latency_ms,
			COUNT(NULLIF(latency_ms, 0)) AS latency_count,
			COALESCE(SUM(billed_nanousd), 0) AS billed_nanousd
		`).
		Where("user_id = ? AND usage_date >= ? AND usage_date < ?", userID, startDate, endDate).
		Group(dayKeyExpression + ", platform_model_name").
		Order("usage_date_key ASC, billed_nanousd DESC, platform_model_name ASC").
		Scan(&modelRows).Error; err != nil {
		return nil, translateError(err)
	}

	resultsByDate := make(map[string]domainbilling.UsageDailySummary)
	dateKeys := make([]string, 0)
	latencyCountsByDate := make(map[string]int64)
	modelsByDate := make(map[string][]domainbilling.UsageDailyModelSummary)
	for _, row := range modelRows {
		key := row.UsageDateKey
		usageDate, err := time.Parse("2006-01-02", key)
		if err != nil {
			return nil, err
		}
		summary, exists := resultsByDate[key]
		if !exists {
			summary = domainbilling.UsageDailySummary{UsageDate: usageDate}
			dateKeys = append(dateKeys, key)
		}
		summary.RecordCount += row.RecordCount
		summary.InputTokens += row.InputTokens
		summary.CacheReadTokens += row.CacheReadTokens
		summary.CacheWriteTokens += row.CacheWriteTokens
		summary.OutputTokens += row.OutputTokens
		summary.ReasoningTokens += row.ReasoningTokens
		summary.CallCount += row.CallCount
		summary.DurationSeconds += row.DurationSeconds
		summary.BilledNanousd += row.BilledNanousd
		if row.LatencyCount > 0 {
			currentLatencyCount := latencyCountsByDate[key]
			summary.AvgLatencyMS = weightedAverageLatency(summary.AvgLatencyMS, currentLatencyCount, row.AvgLatencyMS, row.LatencyCount)
			latencyCountsByDate[key] = currentLatencyCount + row.LatencyCount
		}
		resultsByDate[key] = summary
		modelsByDate[key] = append(modelsByDate[key], domainbilling.UsageDailyModelSummary{
			PlatformModelName: row.PlatformModelName,
			RecordCount:       row.RecordCount,
			InputTokens:       row.InputTokens,
			CacheReadTokens:   row.CacheReadTokens,
			CacheWriteTokens:  row.CacheWriteTokens,
			OutputTokens:      row.OutputTokens,
			ReasoningTokens:   row.ReasoningTokens,
			CallCount:         row.CallCount,
			DurationSeconds:   row.DurationSeconds,
			AvgLatencyMS:      row.AvgLatencyMS,
			BilledNanousd:     row.BilledNanousd,
		})
	}

	results := make([]domainbilling.UsageDailySummary, 0, len(dateKeys))
	for _, key := range dateKeys {
		summary := resultsByDate[key]
		summary.Models = modelsByDate[key]
		results = append(results, summary)
	}
	return results, nil
}

func weightedAverageLatency(currentAvg int64, currentCount int64, nextAvg int64, nextCount int64) int64 {
	totalCount := currentCount + nextCount
	if totalCount <= 0 {
		return 0
	}
	return ((currentAvg * currentCount) + (nextAvg * nextCount)) / totalCount
}

// SumBillableNanousd 统计周期内付费模型的用量金额。
func (r *Repo) SumBillableNanousd(ctx context.Context, userID uint, startAt time.Time, endAt time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.UsageLedger{}).
		Select("COALESCE(SUM(billed_nanousd), 0)").
		Where("user_id = ? AND is_free_model = ? AND billing_at >= ? AND billing_at < ?", userID, false, startAt, endAt).
		Scan(&total).Error
	if err != nil {
		return 0, translateError(err)
	}
	return total, nil
}

func toDomainModelPricing(item model.ModelPricing) domainbilling.ModelPricing {
	return domainbilling.ModelPricing{
		ID:                          item.ID,
		PlatformModelName:           item.PlatformModelName,
		Currency:                    item.Currency,
		IsFree:                      item.IsFree,
		PricingMode:                 normalizePricingMode(item.PricingMode),
		InputNanousdPerMTokens:      item.InputNanousdPerMTokens,
		CacheReadNanousdPerMTokens:  item.CacheReadNanousdPerMTokens,
		CacheWriteNanousdPerMTokens: item.CacheWriteNanousdPerMTokens,
		OutputNanousdPerMTokens:     item.OutputNanousdPerMTokens,
		CallNanousdPerCall:          item.CallNanousdPerCall,
		DurationNanousdPerSecond:    item.DurationNanousdPerSecond,
		TieredPricingJSON:           item.TieredPricingJSON,
		CreatedAt:                   item.CreatedAt,
		UpdatedAt:                   item.UpdatedAt,
	}
}

func toDomainPaymentOrder(item model.PaymentOrder) domainbilling.PaymentOrder {
	return domainbilling.PaymentOrder{
		ID:                 item.ID,
		OrderNo:            item.OrderNo,
		OrderType:          item.OrderType,
		UserID:             item.UserID,
		PlanID:             item.PlanID,
		PriceID:            item.PriceID,
		Provider:           item.Provider,
		Status:             item.Status,
		BaseCurrency:       item.BaseCurrency,
		BaseAmountCents:    item.BaseAmountCents,
		PayCurrency:        item.PayCurrency,
		PayAmountCents:     item.PayAmountCents,
		FXRate:             item.FXRate,
		CreditNanousd:      item.CreditNanousd,
		BillingInterval:    item.BillingInterval,
		Cycles:             item.Cycles,
		ExternalPaymentID:  item.ExternalPaymentID,
		ExternalCheckoutID: item.ExternalCheckoutID,
		CheckoutURL:        item.CheckoutURL,
		PaidAt:             item.PaidAt,
		ExpiredAt:          item.ExpiredAt,
		SnapshotJSON:       item.SnapshotJSON,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func toModelUsageLedger(usage *domainbilling.UsageLedger) model.UsageLedger {
	return model.UsageLedger{
		UserID:              usage.UserID,
		ConversationID:      usage.ConversationID,
		ProviderProtocol:    usage.ProviderProtocol,
		UpstreamName:        usage.UpstreamName,
		PlatformModelName:   usage.PlatformModelName,
		RoutedBindingCode:   usage.RoutedBindingCode,
		UpstreamModelName:   usage.UpstreamModelName,
		IsFreeModel:         usage.IsFreeModel,
		BillingAt:           usage.BillingAt,
		UsageDate:           usage.UsageDate,
		InputTokens:         usage.InputTokens,
		CacheReadTokens:     usage.CacheReadTokens,
		CacheWriteTokens:    usage.CacheWriteTokens,
		CacheWrite5mTokens:  usage.CacheWrite5mTokens,
		CacheWrite1hTokens:  usage.CacheWrite1hTokens,
		OutputTokens:        usage.OutputTokens,
		ReasoningTokens:     usage.ReasoningTokens,
		CallCount:           usage.CallCount,
		DurationSeconds:     usage.DurationSeconds,
		LatencyMS:           usage.LatencyMS,
		UsageSpeed:          usage.UsageSpeed,
		ServiceTier:         usage.ServiceTier,
		BilledCurrency:      usage.BilledCurrency,
		BilledNanousd:       usage.BilledNanousd,
		PricingSnapshotJSON: usage.PricingSnapshotJSON,
	}
}

func toDomainUsageLedger(item model.UsageLedger) domainbilling.UsageLedger {
	return domainbilling.UsageLedger{
		ID:                  item.ID,
		UserID:              item.UserID,
		ConversationID:      item.ConversationID,
		ProviderProtocol:    item.ProviderProtocol,
		UpstreamName:        item.UpstreamName,
		PlatformModelName:   item.PlatformModelName,
		RoutedBindingCode:   item.RoutedBindingCode,
		UpstreamModelName:   item.UpstreamModelName,
		IsFreeModel:         item.IsFreeModel,
		BillingAt:           item.BillingAt,
		UsageDate:           item.UsageDate,
		InputTokens:         item.InputTokens,
		CacheReadTokens:     item.CacheReadTokens,
		CacheWriteTokens:    item.CacheWriteTokens,
		CacheWrite5mTokens:  item.CacheWrite5mTokens,
		CacheWrite1hTokens:  item.CacheWrite1hTokens,
		OutputTokens:        item.OutputTokens,
		ReasoningTokens:     item.ReasoningTokens,
		CallCount:           item.CallCount,
		DurationSeconds:     item.DurationSeconds,
		LatencyMS:           item.LatencyMS,
		UsageSpeed:          item.UsageSpeed,
		ServiceTier:         item.ServiceTier,
		BilledCurrency:      item.BilledCurrency,
		BilledNanousd:       item.BilledNanousd,
		PricingSnapshotJSON: item.PricingSnapshotJSON,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func getOrCreateBillingAccountForUpdate(tx *gorm.DB, userID uint) (*model.BillingAccount, error) {
	if userID == 0 {
		return nil, repository.ErrInvalidInput
	}
	var account model.BillingAccount
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&account).Error
	if err == nil {
		return &account, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, translateError(err)
	}
	account = model.BillingAccount{
		UserID:         userID,
		Currency:       "USD",
		BalanceNanousd: 0,
		Status:         "active",
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&account).Error; err != nil {
		return nil, translateError(err)
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, translateError(err)
	}
	return &account, nil
}

func billingAccountByUserID(tx *gorm.DB, userID uint) (*model.BillingAccount, error) {
	var account model.BillingAccount
	if err := tx.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, translateError(err)
	}
	return &account, nil
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func toDomainBillingAccount(item model.BillingAccount) domainbilling.BillingAccount {
	return domainbilling.BillingAccount{
		ID:             item.ID,
		UserID:         item.UserID,
		Currency:       item.Currency,
		BalanceNanousd: item.BalanceNanousd,
		Status:         item.Status,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toDomainCheckInRecord(item model.CheckInRecord) domainbilling.CheckInRecord {
	return domainbilling.CheckInRecord{
		ID:                   item.ID,
		UserID:               item.UserID,
		CheckInDate:          item.CheckInDate,
		RewardNanousd:        item.RewardNanousd,
		ConsecutiveDays:      item.ConsecutiveDays,
		BalanceTransactionID: item.BalanceTransactionID,
		RefNo:                item.RefNo,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func toDomainBalanceTransaction(item model.BalanceTransaction) domainbilling.BalanceTransaction {
	return domainbilling.BalanceTransaction{
		ID:                  item.ID,
		AccountID:           item.AccountID,
		UserID:              item.UserID,
		Type:                item.Type,
		AmountNanousd:       item.AmountNanousd,
		BalanceAfterNanousd: item.BalanceAfterNanousd,
		RefType:             item.RefType,
		RefID:               item.RefID,
		RefNo:               item.RefNo,
		Description:         item.Description,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func toDomainPlan(item model.BillingPlan) domainbilling.Plan {
	return domainbilling.Plan{
		ID:                  item.ID,
		Code:                item.Code,
		Name:                item.Name,
		Description:         item.Description,
		FeatureJSON:         item.FeatureJSON,
		PeriodCreditNanousd: item.PeriodCreditNanousd,
		DiscountPercent:     item.DiscountPercent,
		SortOrder:           item.SortOrder,
		IsActive:            item.IsActive,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func toDomainPrice(item model.BillingPrice) domainbilling.Price {
	return domainbilling.Price{
		ID:               item.ID,
		PlanID:           item.PlanID,
		Code:             item.Code,
		BillingInterval:  item.BillingInterval,
		Currency:         item.Currency,
		AmountCents:      item.AmountCents,
		IsActive:         item.IsActive,
		IsDefault:        item.IsDefault,
		ExternalPriceRef: item.ExternalPriceRef,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func toDomainSubscription(item model.Subscription) domainbilling.Subscription {
	return domainbilling.Subscription{
		ID:                   item.ID,
		UserID:               item.UserID,
		PlanID:               item.PlanID,
		PriceID:              item.PriceID,
		Status:               item.Status,
		StartAt:              item.StartAt,
		CurrentPeriodStartAt: item.CurrentPeriodStartAt,
		CurrentPeriodEndAt:   item.CurrentPeriodEndAt,
		CancelAtPeriodEnd:    item.CancelAtPeriodEnd,
		CanceledAt:           item.CanceledAt,
		AutoRenew:            item.AutoRenew,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func toDomainExternalAccountLink(item model.ExternalAccountLink) domainbilling.ExternalAccountLink {
	return domainbilling.ExternalAccountLink{
		ID:                  item.ID,
		UserID:              item.UserID,
		Platform:            item.Platform,
		ExternalUserID:      item.ExternalUserID,
		ExternalDisplayName: item.ExternalDisplayName,
		LinuxDOSub:          item.LinuxDOSub,
		Status:              item.Status,
		LinkedAt:            item.LinkedAt,
		LastSyncedAt:        item.LastSyncedAt,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func toDomainExternalTransfer(item model.ExternalTransfer) domainbilling.ExternalTransfer {
	return domainbilling.ExternalTransfer{
		ID:                    item.ID,
		UserID:                item.UserID,
		LinkID:                item.LinkID,
		Platform:              item.Platform,
		Direction:             item.Direction,
		ExternalTransferID:    item.ExternalTransferID,
		IdempotencyKey:        item.IdempotencyKey,
		ExternalAmountUSD:     item.ExternalAmountUSD,
		CreditedAmountNanousd: item.CreditedAmountNanousd,
		BalanceTransactionID:  item.BalanceTransactionID,
		Status:                item.Status,
		FailureReason:         item.FailureReason,
		RequestedAt:           item.RequestedAt,
		CompletedAt:           item.CompletedAt,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func toDomainRedemptionCode(item model.RedemptionCode) domainbilling.RedemptionCode {
	return domainbilling.RedemptionCode{
		ID:              item.ID,
		CodeHash:        item.CodeHash,
		CodeEncrypted:   item.CodeEncrypted,
		CodeHint:        item.CodeHint,
		Mode:            item.Mode,
		RewardType:      item.RewardType,
		CreditNanousd:   item.CreditNanousd,
		PlanID:          item.PlanID,
		DurationDays:    item.DurationDays,
		MaxRedemptions:  copyIntPointer(item.MaxRedemptions),
		PerUserLimit:    item.PerUserLimit,
		RedeemedCount:   item.RedeemedCount,
		Status:          item.Status,
		ExpiresAt:       item.ExpiresAt,
		Description:     item.Description,
		CreatedByUserID: item.CreatedByUserID,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func toDomainRedemption(item model.Redemption) domainbilling.Redemption {
	return domainbilling.Redemption{
		ID:                   item.ID,
		CodeID:               item.CodeID,
		UserID:               item.UserID,
		Mode:                 item.Mode,
		RewardType:           item.RewardType,
		CreditNanousd:        item.CreditNanousd,
		PlanID:               item.PlanID,
		SubscriptionID:       item.SubscriptionID,
		BalanceTransactionID: item.BalanceTransactionID,
		RefNo:                item.RefNo,
		SnapshotJSON:         item.SnapshotJSON,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func validateRedeemableCode(tx *gorm.DB, code model.RedemptionCode, userID uint, currentMode string, now time.Time) error {
	if code.Status != domainbilling.RedemptionCodeStatusActive ||
		!domainbilling.RedemptionCodeModeAvailableInBillingMode(code.Mode, currentMode) ||
		(code.ExpiresAt != nil && !code.ExpiresAt.After(now)) {
		return repository.ErrRedemptionUnavailable
	}
	if code.MaxRedemptions != nil && code.RedeemedCount >= *code.MaxRedemptions {
		return repository.ErrRedemptionExhausted
	}
	perUserLimit := code.PerUserLimit
	if perUserLimit <= 0 {
		perUserLimit = 1
	}
	var userCount int64
	if err := tx.Model(&model.Redemption{}).
		Where("code_id = ? AND user_id = ?", code.ID, userID).
		Count(&userCount).Error; err != nil {
		return translateError(err)
	}
	if userCount >= int64(perUserLimit) {
		return repository.ErrRedemptionUserLimitExceeded
	}
	switch code.Mode {
	case domainbilling.RedemptionCodeModeUsage:
		if code.RewardType != domainbilling.RedemptionRewardTypeBalance || code.CreditNanousd <= 0 {
			return repository.ErrInvalidInput
		}
	case domainbilling.RedemptionCodeModePeriod:
		if code.RewardType != domainbilling.RedemptionRewardTypeSubscription || code.PlanID == 0 {
			return repository.ErrInvalidInput
		}
	default:
		return repository.ErrRedemptionUnavailable
	}
	return nil
}

func applyRedemptionBalance(tx *gorm.DB, userID uint, code model.RedemptionCode, refNo string) (*model.BillingAccount, uint, error) {
	account, err := getOrCreateBillingAccountForUpdate(tx, userID)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Model(account).Updates(map[string]interface{}{
		"balance_nanousd": gorm.Expr("balance_nanousd + ?", code.CreditNanousd),
		"currency":        "USD",
		"status":          "active",
	}).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", account.ID).First(account).Error; err != nil {
		return nil, 0, translateError(err)
	}
	transaction := model.BalanceTransaction{
		AccountID:           account.ID,
		UserID:              userID,
		Type:                domainbilling.BalanceTransactionTypeRedemption,
		AmountNanousd:       code.CreditNanousd,
		BalanceAfterNanousd: account.BalanceNanousd,
		RefType:             "redemption_code",
		RefID:               code.ID,
		RefNo:               strings.TrimSpace(refNo),
		Description:         firstNonEmpty(strings.TrimSpace(code.Description), "兑换码入账"),
	}
	if err := tx.Create(&transaction).Error; err != nil {
		return nil, 0, translateError(err)
	}
	return account, transaction.ID, nil
}

func applyRedemptionSubscription(tx *gorm.DB, userID uint, code model.RedemptionCode, now time.Time) (*model.Subscription, error) {
	var plan model.BillingPlan
	if err := tx.Where("id = ? AND is_active = ?", code.PlanID, true).First(&plan).Error; err != nil {
		return nil, translateError(err)
	}
	price, err := activeDefaultPriceForPlan(tx, plan.ID)
	if err != nil {
		return nil, err
	}
	if code.DurationDays <= 0 {
		return nil, repository.ErrInvalidInput
	}
	duration := time.Duration(code.DurationDays) * 24 * time.Hour
	return grantSubscriptionOnTimeline(tx, subscriptionTimelineGrantRequest{
		UserID:   userID,
		Plan:     plan,
		Price:    *price,
		StartAt:  now,
		Duration: duration,
		NewGrant: true,
	})
}

func activeDefaultPriceForPlan(tx *gorm.DB, planID uint) (*model.BillingPrice, error) {
	var price model.BillingPrice
	err := tx.Where("plan_id = ? AND is_active = ? AND is_default = ?", planID, true, true).
		First(&price).Error
	if err == nil {
		return &price, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, translateError(err)
	}
	if err := tx.Where("plan_id = ? AND is_active = ?", planID, true).
		Order("amount_cents ASC, id ASC").
		First(&price).Error; err != nil {
		return nil, translateError(err)
	}
	return &price, nil
}

type subscriptionTimelineGrantRequest struct {
	UserID            uint
	Plan              model.BillingPlan
	Price             model.BillingPrice
	StartAt           time.Time
	Duration          time.Duration
	CancelAtPeriodEnd bool
	AutoRenew         bool
	NewGrant          bool
}

func grantSubscriptionOnTimeline(tx *gorm.DB, input subscriptionTimelineGrantRequest) (*model.Subscription, error) {
	if input.UserID == 0 || input.Plan.ID == 0 || input.Price.ID == 0 || input.Duration <= 0 {
		return nil, repository.ErrInvalidInput
	}
	now := input.StartAt
	if now.IsZero() {
		now = time.Now()
	}
	var existing []model.Subscription
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND status = ? AND current_period_end_at IS NOT NULL AND current_period_end_at > ?", input.UserID, "active", now).
		Order("current_period_start_at ASC, current_period_end_at ASC, id ASC").
		Find(&existing).Error; err != nil {
		return nil, translateError(err)
	}
	plans, err := billingPlansForTimeline(tx, appendSubscriptionPlanID(existing, input.Plan.ID))
	if err != nil {
		return nil, err
	}

	segments := make([]subscriptionTimelineSegment, 0, len(existing)+1)
	for _, item := range existing {
		if item.CurrentPeriodEndAt == nil || !item.CurrentPeriodEndAt.After(now) {
			continue
		}
		startAt := item.CurrentPeriodStartAt
		if startAt.Before(now) {
			startAt = now
		}
		if !item.CurrentPeriodEndAt.After(startAt) {
			continue
		}
		segmentPlan, ok := plans[item.PlanID]
		if !ok {
			return nil, repository.ErrInvalidInput
		}
		segments = append(segments, subscriptionTimelineSegment{
			SubscriptionID:    item.ID,
			PlanID:            item.PlanID,
			PriceID:           item.PriceID,
			Rank:              subscriptionPlanRank(segmentPlan),
			StartAt:           startAt,
			EndAt:             *item.CurrentPeriodEndAt,
			CancelAtPeriodEnd: item.CancelAtPeriodEnd,
			AutoRenew:         item.AutoRenew,
		})
	}

	segments, err = buildSubscriptionTimeline(segments, subscriptionTimelineGrant{
		PlanID:            input.Plan.ID,
		PriceID:           input.Price.ID,
		Rank:              subscriptionPlanRank(input.Plan),
		StartAt:           now,
		Duration:          input.Duration,
		CancelAtPeriodEnd: input.CancelAtPeriodEnd,
		AutoRenew:         input.AutoRenew,
		NewGrant:          input.NewGrant,
	})
	if err != nil {
		return nil, err
	}
	return applySubscriptionTimeline(tx, input.UserID, existing, segments, now, input.Plan.ID)
}

func applySubscriptionTimeline(
	tx *gorm.DB,
	userID uint,
	existing []model.Subscription,
	segments []subscriptionTimelineSegment,
	now time.Time,
	grantPlanID uint,
) (*model.Subscription, error) {
	targetsBySubscriptionID := make(map[uint][]subscriptionTimelineSegment, len(existing))
	newSegments := make([]subscriptionTimelineSegment, 0)
	for _, segment := range normalizeSubscriptionTimeline(segments) {
		if segment.SubscriptionID > 0 {
			targetsBySubscriptionID[segment.SubscriptionID] = append(targetsBySubscriptionID[segment.SubscriptionID], segment)
			continue
		}
		newSegments = append(newSegments, segment)
	}

	var granted *model.Subscription
	for _, item := range existing {
		targets := targetsBySubscriptionID[item.ID]
		if len(targets) == 0 {
			if err := expireSubscriptionForTimeline(tx, item, now); err != nil {
				return nil, err
			}
			continue
		}

		reusableIndex := reusableSubscriptionSegmentIndex(item, targets, now)
		if reusableIndex < 0 {
			if err := expireSubscriptionForTimeline(tx, item, now); err != nil {
				return nil, err
			}
		}
		for index, target := range targets {
			var record *model.Subscription
			var err error
			if index == reusableIndex {
				record, err = updateSubscriptionSegment(tx, item, target, now)
			} else {
				record, err = createSubscriptionSegment(tx, userID, target)
			}
			if err != nil {
				return nil, err
			}
			if granted == nil && target.NewGrant && target.PlanID == grantPlanID {
				granted = record
			}
		}
	}

	for _, target := range newSegments {
		record, err := createSubscriptionSegment(tx, userID, target)
		if err != nil {
			return nil, err
		}
		if granted == nil && target.NewGrant && target.PlanID == grantPlanID {
			granted = record
		}
	}
	if granted == nil {
		return nil, repository.ErrInvalidInput
	}
	return granted, nil
}

func reusableSubscriptionSegmentIndex(item model.Subscription, targets []subscriptionTimelineSegment, now time.Time) int {
	for index, target := range targets {
		if !target.EndAt.After(target.StartAt) {
			continue
		}
		if item.CurrentPeriodStartAt.Before(now) && target.StartAt.After(now) {
			continue
		}
		return index
	}
	return -1
}

func expireSubscriptionForTimeline(tx *gorm.DB, item model.Subscription, now time.Time) error {
	endAt := now
	if item.CurrentPeriodStartAt.After(endAt) {
		endAt = item.CurrentPeriodStartAt
	}
	if item.CurrentPeriodEndAt != nil && item.CurrentPeriodEndAt.Before(endAt) {
		endAt = *item.CurrentPeriodEndAt
	}
	return translateError(tx.Model(&item).Updates(map[string]interface{}{
		"status":                "expired",
		"auto_renew":            false,
		"cancel_at_period_end":  false,
		"current_period_end_at": endAt,
	}).Error)
}

func updateSubscriptionSegment(tx *gorm.DB, item model.Subscription, segment subscriptionTimelineSegment, now time.Time) (*model.Subscription, error) {
	startAt := segment.StartAt
	if item.CurrentPeriodStartAt.Before(now) && !segment.StartAt.After(now) {
		startAt = item.CurrentPeriodStartAt
	}
	recordStartAt := startAt
	if item.StartAt.Before(startAt) && !segment.StartAt.After(now) {
		recordStartAt = item.StartAt
	}
	endAt := segment.EndAt
	if err := tx.Model(&item).Updates(map[string]interface{}{
		"plan_id":                 segment.PlanID,
		"price_id":                segment.PriceID,
		"status":                  "active",
		"start_at":                recordStartAt,
		"current_period_start_at": startAt,
		"current_period_end_at":   endAt,
		"cancel_at_period_end":    segment.CancelAtPeriodEnd,
		"canceled_at":             nil,
		"auto_renew":              segment.AutoRenew,
	}).Error; err != nil {
		return nil, translateError(err)
	}
	var updated model.Subscription
	if err := tx.Where("id = ?", item.ID).First(&updated).Error; err != nil {
		return nil, translateError(err)
	}
	return &updated, nil
}

func createSubscriptionSegment(tx *gorm.DB, userID uint, segment subscriptionTimelineSegment) (*model.Subscription, error) {
	if userID == 0 || segment.PlanID == 0 || segment.PriceID == 0 || !segment.EndAt.After(segment.StartAt) {
		return nil, repository.ErrInvalidInput
	}
	endAt := segment.EndAt
	record := model.Subscription{
		UserID:               userID,
		PlanID:               segment.PlanID,
		PriceID:              segment.PriceID,
		Status:               "active",
		StartAt:              segment.StartAt,
		CurrentPeriodStartAt: segment.StartAt,
		CurrentPeriodEndAt:   &endAt,
		CancelAtPeriodEnd:    segment.CancelAtPeriodEnd,
		AutoRenew:            segment.AutoRenew,
	}
	if err := tx.Create(&record).Error; err != nil {
		return nil, translateError(err)
	}
	return &record, nil
}

type subscriptionTimelineSegment struct {
	SubscriptionID    uint
	PlanID            uint
	PriceID           uint
	Rank              int
	StartAt           time.Time
	EndAt             time.Time
	CancelAtPeriodEnd bool
	AutoRenew         bool
	NewGrant          bool
}

type subscriptionTimelineGrant struct {
	SubscriptionID    uint
	PlanID            uint
	PriceID           uint
	Rank              int
	StartAt           time.Time
	Duration          time.Duration
	CancelAtPeriodEnd bool
	AutoRenew         bool
	NewGrant          bool
}

func buildSubscriptionTimeline(existing []subscriptionTimelineSegment, grant subscriptionTimelineGrant) ([]subscriptionTimelineSegment, error) {
	if grant.Duration <= 0 {
		return nil, repository.ErrInvalidInput
	}
	segments := normalizeSubscriptionTimeline(existing)
	queue := []subscriptionTimelineGrant{grant}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		var displaced []subscriptionTimelineGrant
		var err error
		segments, displaced, err = placeSubscriptionTimelineGrant(segments, current)
		if err != nil {
			return nil, err
		}
		queue = append(displaced, queue...)
	}
	return normalizeSubscriptionTimeline(segments), nil
}

func placeSubscriptionTimelineGrant(existing []subscriptionTimelineSegment, grant subscriptionTimelineGrant) ([]subscriptionTimelineSegment, []subscriptionTimelineGrant, error) {
	if grant.Duration <= 0 {
		return existing, nil, repository.ErrInvalidInput
	}
	segments := normalizeSubscriptionTimeline(existing)
	remaining := grant.Duration
	cursor := grant.StartAt
	displaced := make([]subscriptionTimelineGrant, 0)
	for remaining > 0 {
		blocker, ok := nextSubscriptionTimelineBlocker(segments, grant.Rank, cursor)
		if ok && !blocker.StartAt.After(cursor) {
			cursor = blocker.EndAt
			continue
		}
		endAt := cursor.Add(remaining)
		if ok && blocker.StartAt.Before(endAt) {
			endAt = blocker.StartAt
		}
		if !endAt.After(cursor) {
			cursor = blocker.EndAt
			continue
		}
		allocated := subscriptionTimelineSegment{
			SubscriptionID:    grant.SubscriptionID,
			PlanID:            grant.PlanID,
			PriceID:           grant.PriceID,
			Rank:              grant.Rank,
			StartAt:           cursor,
			EndAt:             endAt,
			CancelAtPeriodEnd: grant.CancelAtPeriodEnd,
			AutoRenew:         grant.AutoRenew,
			NewGrant:          grant.NewGrant,
		}
		var moved []subscriptionTimelineGrant
		segments, moved = insertSubscriptionTimelineSegment(segments, allocated)
		displaced = append(displaced, moved...)
		remaining -= endAt.Sub(cursor)
		cursor = endAt
	}
	return normalizeSubscriptionTimeline(segments), displaced, nil
}

func nextSubscriptionTimelineBlocker(segments []subscriptionTimelineSegment, rank int, cursor time.Time) (subscriptionTimelineSegment, bool) {
	var result subscriptionTimelineSegment
	found := false
	for _, item := range segments {
		if item.Rank < rank || !item.EndAt.After(cursor) {
			continue
		}
		if !found || item.StartAt.Before(result.StartAt) || (item.StartAt.Equal(result.StartAt) && item.EndAt.Before(result.EndAt)) {
			result = item
			found = true
		}
	}
	return result, found
}

func insertSubscriptionTimelineSegment(segments []subscriptionTimelineSegment, allocated subscriptionTimelineSegment) ([]subscriptionTimelineSegment, []subscriptionTimelineGrant) {
	next := make([]subscriptionTimelineSegment, 0, len(segments)+1)
	displaced := make([]subscriptionTimelineGrant, 0)
	for _, item := range segments {
		if !subscriptionTimelineOverlaps(item, allocated) || item.Rank >= allocated.Rank {
			next = append(next, item)
			continue
		}
		overlapStart := maxTime(item.StartAt, allocated.StartAt)
		overlapEnd := minTime(item.EndAt, allocated.EndAt)
		if item.StartAt.Before(overlapStart) {
			left := item
			left.EndAt = overlapStart
			next = append(next, left)
		}
		if item.EndAt.After(overlapEnd) {
			right := item
			right.StartAt = overlapEnd
			next = append(next, right)
		}
		if overlapEnd.After(overlapStart) {
			displaced = append(displaced, subscriptionTimelineGrant{
				SubscriptionID:    item.SubscriptionID,
				PlanID:            item.PlanID,
				PriceID:           item.PriceID,
				Rank:              item.Rank,
				StartAt:           maxTime(allocated.EndAt, item.EndAt),
				Duration:          overlapEnd.Sub(overlapStart),
				CancelAtPeriodEnd: item.CancelAtPeriodEnd,
				AutoRenew:         item.AutoRenew,
				NewGrant:          item.NewGrant,
			})
		}
	}
	next = append(next, allocated)
	return normalizeSubscriptionTimeline(next), displaced
}

func normalizeSubscriptionTimeline(segments []subscriptionTimelineSegment) []subscriptionTimelineSegment {
	clean := make([]subscriptionTimelineSegment, 0, len(segments))
	for _, item := range segments {
		if item.PlanID == 0 || item.PriceID == 0 || !item.EndAt.After(item.StartAt) {
			continue
		}
		clean = append(clean, item)
	}
	sort.SliceStable(clean, func(i, j int) bool {
		if clean[i].StartAt.Equal(clean[j].StartAt) {
			if clean[i].Rank == clean[j].Rank {
				if clean[i].EndAt.Equal(clean[j].EndAt) {
					return clean[i].PlanID < clean[j].PlanID
				}
				return clean[i].EndAt.Before(clean[j].EndAt)
			}
			return clean[i].Rank > clean[j].Rank
		}
		return clean[i].StartAt.Before(clean[j].StartAt)
	})
	merged := make([]subscriptionTimelineSegment, 0, len(clean))
	for _, item := range clean {
		lastIndex := len(merged) - 1
		if lastIndex >= 0 {
			last := &merged[lastIndex]
			if last.PlanID == item.PlanID &&
				last.PriceID == item.PriceID &&
				last.Rank == item.Rank &&
				last.SubscriptionID == item.SubscriptionID &&
				last.CancelAtPeriodEnd == item.CancelAtPeriodEnd &&
				last.AutoRenew == item.AutoRenew &&
				last.NewGrant == item.NewGrant &&
				!last.EndAt.Before(item.StartAt) {
				if item.EndAt.After(last.EndAt) {
					last.EndAt = item.EndAt
				}
				continue
			}
		}
		merged = append(merged, item)
	}
	return merged
}

func subscriptionTimelineOverlaps(left subscriptionTimelineSegment, right subscriptionTimelineSegment) bool {
	return left.StartAt.Before(right.EndAt) && right.StartAt.Before(left.EndAt)
}

func minTime(left time.Time, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func maxTime(left time.Time, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func appendSubscriptionPlanID(items []model.Subscription, planID uint) []uint {
	ids := make([]uint, 0, len(items)+1)
	seen := make(map[uint]struct{}, len(items)+1)
	if planID > 0 {
		ids = append(ids, planID)
		seen[planID] = struct{}{}
	}
	for _, item := range items {
		if item.PlanID == 0 {
			continue
		}
		if _, ok := seen[item.PlanID]; ok {
			continue
		}
		ids = append(ids, item.PlanID)
		seen[item.PlanID] = struct{}{}
	}
	return ids
}

func billingPlansForTimeline(tx *gorm.DB, planIDs []uint) (map[uint]model.BillingPlan, error) {
	results := make(map[uint]model.BillingPlan, len(planIDs))
	if len(planIDs) == 0 {
		return results, nil
	}
	var plans []model.BillingPlan
	if err := tx.Where("id IN ?", planIDs).Find(&plans).Error; err != nil {
		return nil, translateError(err)
	}
	for _, item := range plans {
		results[item.ID] = item
	}
	if len(results) != len(planIDs) {
		return nil, repository.ErrInvalidInput
	}
	return results, nil
}

func subscriptionPlanRank(plan model.BillingPlan) int {
	if strings.TrimSpace(plan.Code) == "free" {
		return 0
	}
	if plan.SortOrder > 0 {
		return plan.SortOrder
	}
	return int(plan.ID)
}

func redemptionSnapshotJSON(code model.RedemptionCode) string {
	payload := map[string]interface{}{
		"code_id":        code.ID,
		"mode":           code.Mode,
		"reward_type":    code.RewardType,
		"credit_nanousd": code.CreditNanousd,
		"plan_id":        code.PlanID,
		"duration_days":  code.DurationDays,
		"description":    strings.TrimSpace(code.Description),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func normalizeOrderType(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.PaymentOrderTypeTopUp:
		return domainbilling.PaymentOrderTypeTopUp
	default:
		return domainbilling.PaymentOrderTypeSubscription
	}
}

func normalizeRedemptionMode(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.RedemptionCodeModePeriod:
		return domainbilling.RedemptionCodeModePeriod
	default:
		return domainbilling.RedemptionCodeModeUsage
	}
}

func normalizeRedemptionRewardType(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.RedemptionRewardTypeSubscription:
		return domainbilling.RedemptionRewardTypeSubscription
	default:
		return domainbilling.RedemptionRewardTypeBalance
	}
}

func normalizeRedemptionStatus(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.RedemptionCodeStatusInactive:
		return domainbilling.RedemptionCodeStatusInactive
	case domainbilling.RedemptionCodeStatusDeleted:
		return domainbilling.RedemptionCodeStatusDeleted
	default:
		return domainbilling.RedemptionCodeStatusActive
	}
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeCurrency(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "USD"
	}
	return normalized
}

func normalizePricingMode(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.PricingModeCall:
		return domainbilling.PricingModeCall
	case domainbilling.PricingModeDuration:
		return domainbilling.PricingModeDuration
	case domainbilling.PricingModeTiered:
		return domainbilling.PricingModeTiered
	default:
		return domainbilling.PricingModeToken
	}
}

func clampNonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizeInterval(value string) string {
	switch strings.TrimSpace(value) {
	case domainbilling.IntervalYear:
		return domainbilling.IntervalYear
	case domainbilling.IntervalLifetime:
		return domainbilling.IntervalLifetime
	default:
		return domainbilling.IntervalMonth
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
