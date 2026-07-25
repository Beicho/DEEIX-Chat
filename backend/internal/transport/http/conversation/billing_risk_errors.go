package conversation

import (
	"errors"
	"net/http"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// writeRiskControlError 将资损风控拒绝映射为用户可理解的 402 响应。
// 返回 true 表示已写出响应，调用方应直接返回。
func writeRiskControlError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, billing.ErrCallCostNotCovered):
		response.Error(c, http.StatusPaymentRequired, "call cost is not covered by available credit")
	case errors.Is(err, billing.ErrNewUserCooldown):
		response.Error(c, http.StatusForbidden, "new user cooldown is active")
	case errors.Is(err, billing.ErrDailySpendLimitExceeded):
		response.Error(c, http.StatusTooManyRequests, "daily spend limit exceeded")
	case errors.Is(err, billing.ErrConcurrentGenerationLimit):
		response.Error(c, http.StatusTooManyRequests, "concurrent generation limit exceeded")
	default:
		return false
	}
	return true
}
