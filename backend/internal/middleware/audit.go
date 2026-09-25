package middleware

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"pki-certificate-rollover-impact/backend/internal/util"
	"time"
)

func Audit(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Header("Cache-Control", "no-store")
		c.Next()
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			actor, _ := Actor(c)
			logger.Info("mutating request completed", "request_id", util.RequestID(c), "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(), "actor", actor.Username, "role", actor.Role, "duration_ms", time.Since(started).Milliseconds())
		}
	}
}
