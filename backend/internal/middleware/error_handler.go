package middleware

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"pki-certificate-rollover-impact/backend/internal/util"
)

func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, entry := range c.Errors {
			if apiErr, ok := entry.Err.(*util.APIError); ok {
				level := slog.LevelWarn
				if apiErr.Status >= 500 {
					level = slog.LevelError
				}
				logger.Log(c.Request.Context(), level, "request failed", "request_id", util.RequestID(c), "method", c.Request.Method, "path", c.Request.URL.Path, "status", apiErr.Status, "code", apiErr.Code, "error", apiErr.Message)
			}
		}
	}
}
