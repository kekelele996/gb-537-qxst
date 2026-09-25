package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/util"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("request panic recovered", "request_id", util.RequestID(c), "method", c.Request.Method, "path", c.Request.URL.Path, "error", fmt.Sprint(recovered))
		util.Fail(c, util.NewError(http.StatusInternalServerError, util.CodeInternal, "request could not be completed"))
	})
}
