package billing

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes 注册计费公开回调路由。
func (m *Module) RegisterPublicRoutes(publicGroup *gin.RouterGroup) {
	publicGroup.POST("/billing/payments/stripe/webhook", m.Handler.StripeWebhook)
	publicGroup.GET("/billing/payments/epay/notify", m.Handler.EPayNotify)
	publicGroup.POST("/billing/payments/epay/notify", m.Handler.EPayNotify)
	publicGroup.GET("/public/billing/plans", m.Handler.ListPlans)
}

// RegisterRoutes 注册计费域路由。
func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	authRequired.GET("/billing/config", m.Handler.GetBillingConfig)
	authRequired.GET("/billing/account", m.Handler.GetBillingAccount)
	authRequired.GET("/billing/checkin", m.Handler.GetCheckInStatus)
	authRequired.POST("/billing/checkin", m.Handler.ClaimDailyCheckIn)
	authRequired.GET("/billing/overview", m.Handler.GetBillingOverview)
	authRequired.GET("/billing/plans", m.Handler.ListPlans)
	authRequired.POST("/billing/subscriptions", m.Handler.Subscribe)
	authRequired.POST("/billing/payments/checkout", m.Handler.CreateCheckout)
	authRequired.GET("/billing/payments", m.Handler.ListPaymentOrders)
	authRequired.GET("/billing/payments/:order_no", m.Handler.GetPaymentOrder)
	authRequired.POST("/billing/redemptions", m.Handler.RedeemCode)
	authRequired.GET("/billing/redemptions", m.Handler.ListRedemptions)
	authRequired.GET("/billing/balance-transactions", m.Handler.ListUserBalanceTransactions)
	authRequired.POST("/billing/external/newapi/link", m.Handler.LinkNewAPIAccount)
	authRequired.GET("/billing/external/newapi/balance", m.Handler.GetNewAPIBalance)
	authRequired.POST("/billing/external/newapi/transfer", m.Handler.TransferFromNewAPI)
	authRequired.GET("/billing/external/newapi/transfers", m.Handler.ListNewAPITransfers)
	authRequired.GET("/billing/usage", m.Handler.ListUsage)
	authRequired.GET("/billing/usage.csv", m.Handler.ExportUsageCSV)
	authRequired.GET("/billing/usage/monthly", m.Handler.ListMonthlyUsage)
	authRequired.GET("/billing/usage/daily", m.Handler.ListDailyUsage)
}

// RegisterAdminRoutes 注册管理员侧计费路由。
func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/billing/config", m.Handler.GetBillingConfig)
	adminGroup.PATCH("/billing/config", m.Handler.PatchBillingConfig)
	adminGroup.GET("/billing/plans", m.Handler.ListPlans)
	adminGroup.POST("/billing/plans", m.Handler.CreatePlan)
	adminGroup.PATCH("/billing/plans/:id", m.Handler.UpdatePlan)
	adminGroup.DELETE("/billing/plans/:id", m.Handler.DeletePlan)
	adminGroup.PATCH("/billing/accounts/:user_id/balance", m.Handler.UpdateBillingAccountBalance)
	adminGroup.POST("/billing/accounts/:user_id/balance-delta", m.Handler.AdjustBillingAccountBalance)
	adminGroup.GET("/billing/checkin", m.Handler.GetAdminCheckIn)
	adminGroup.PATCH("/billing/checkin/config", m.Handler.PatchAdminCheckInConfig)
	adminGroup.GET("/checkin", m.Handler.GetAdminCheckIn)
	adminGroup.GET("/checkin/stats", m.Handler.GetAdminCheckIn)
	adminGroup.PATCH("/checkin/config", m.Handler.PatchAdminCheckInConfig)
	adminGroup.GET("/billing/balance-transactions", m.Handler.ListAdminBalanceTransactions)
	adminGroup.GET("/billing/payment-orders", m.Handler.ListAdminPaymentOrders)
	adminGroup.POST("/billing/payment-orders/:order_no/actions", m.Handler.ApplyPaymentOrderAction)
	adminGroup.GET("/billing/usage.csv", m.Handler.ExportAdminUsageCSV)
	adminGroup.GET("/billing/risk-summary", m.Handler.GetBillingRiskSummary)
	adminGroup.GET("/billing/redemption-codes", m.Handler.ListRedemptionCodes)
	adminGroup.POST("/billing/redemption-codes", m.Handler.CreateRedemptionCodes)
	adminGroup.POST("/billing/redemption-codes/batch-delete", m.Handler.BatchDeleteRedemptionCodes)
	adminGroup.GET("/billing/redemption-codes/:id/code", m.Handler.RevealRedemptionCode)
	adminGroup.PATCH("/billing/redemption-codes/:id", m.Handler.PatchRedemptionCode)
	adminGroup.DELETE("/billing/redemption-codes/:id", m.Handler.DeleteRedemptionCode)
	adminGroup.GET("/billing/model-prices", m.Handler.ListModelPricing)
	adminGroup.PUT("/billing/model-prices", m.Handler.UpsertModelPricing)
	adminGroup.GET("/billing/official-pricing/openrouter", m.Handler.GetOpenRouterOfficialPricing)
}
