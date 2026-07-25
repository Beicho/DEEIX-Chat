package middleware

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	domainuser "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/user"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// GenerationSlotStore 提供并发槽位的占用与释放能力。
type GenerationSlotStore interface {
	AcquireConcurrencySlot(ctx context.Context, key string, limit int, ttl time.Duration) (bool, error)
	ReleaseConcurrencySlot(ctx context.Context, key string) error
}

// generationConcurrencyTTL 兜底释放时间，防止连接异常中断导致槽位泄漏。
const generationConcurrencyTTL = 15 * time.Minute

var generationConcurrencyRoutes = []*regexp.Regexp{
	regexp.MustCompile(`^/api/v1/conversations/[^/]+/messages$`),
	regexp.MustCompile(`^/api/v1/conversations/[^/]+/messages/stream$`),
	regexp.MustCompile(`^/api/v1/conversations/[^/]+/media/images/generations/stream$`),
	regexp.MustCompile(`^/api/v1/conversations/[^/]+/media/images/edits/stream$`),
	regexp.MustCompile(`^/api/v1/conversations/[^/]+/media/videos/generations/stream$`),
}

// GenerationConcurrencyLimit 限制单个用户同时进行的生成任务数量。
func GenerationConcurrencyLimit(store GenerationSlotStore, runtime *config.Runtime) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		limit := generationConcurrencyLimit(runtime)
		if limit <= 0 {
			c.Next()
			return
		}
		if !isGenerationConcurrencyRoute(c.Request.URL.Path) {
			c.Next()
			return
		}
		role, hasRole := c.Get(ContextKeyUserRole)
		if roleStr, ok := role.(string); hasRole && ok && domainuser.IsAdminRole(roleStr) {
			c.Next()
			return
		}
		userIDValue, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Next()
			return
		}
		userID, ok := rateLimitUserID(userIDValue)
		if !ok {
			c.Next()
			return
		}

		key := fmt.Sprintf("generation:concurrency:user:%d", userID)
		acquired, err := store.AcquireConcurrencySlot(c.Request.Context(), key, limit, generationConcurrencyTTL)
		if err != nil {
			// 存储异常时放行，避免风控组件故障直接阻断正常聊天。
			c.Next()
			return
		}
		if !acquired {
			c.Header("Retry-After", "10")
			response.Error(c, http.StatusTooManyRequests, "concurrent generation limit exceeded")
			c.Abort()
			return
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = store.ReleaseConcurrencySlot(releaseCtx, key)
		}()
		c.Next()
	}
}

func generationConcurrencyLimit(runtime *config.Runtime) int {
	if runtime == nil {
		return 0
	}
	return runtime.Snapshot().RiskMaxConcurrentGenerations
}

func isGenerationConcurrencyRoute(path string) bool {
	trimmed := strings.TrimSpace(path)
	for _, pattern := range generationConcurrencyRoutes {
		if pattern.MatchString(trimmed) {
			return true
		}
	}
	return false
}
