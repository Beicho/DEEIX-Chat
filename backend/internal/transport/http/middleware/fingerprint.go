package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const FingerprintIDHeader = "X-DEEIX-Fingerprint-ID"

type FingerprintRecorder interface {
	TouchFingerprint(ctx context.Context, userID uint, fingerprintID string, ipAddress string, userAgent string)
}

func FingerprintMiddleware(recorder FingerprintRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		fingerprintID := strings.TrimSpace(c.GetHeader(FingerprintIDHeader))
		if recorder != nil && fingerprintID != "" {
			userID := MustUserID(c)
			ipAddress := c.ClientIP()
			userAgent := c.Request.UserAgent()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				recorder.TouchFingerprint(ctx, userID, fingerprintID, ipAddress, userAgent)
			}()
		}
		c.Next()
	}
}
