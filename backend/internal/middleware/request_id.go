package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/gin-gonic/gin"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !requestIDPattern.MatchString(requestID) {
			buffer := make([]byte, 16)
			if _, err := rand.Read(buffer); err != nil {
				requestID = "request-id-unavailable"
			} else {
				requestID = hex.EncodeToString(buffer)
			}
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
