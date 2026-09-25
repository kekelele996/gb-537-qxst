package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/util"
)

func RequirePermission(permission constants.Permission) gin.HandlerFunc {
	return RequireAnyPermission(permission)
}

func RequireAnyPermission(permissions ...constants.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := Actor(c)
		if !ok {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication context is missing"))
			return
		}
		for _, permission := range permissions {
			if constants.HasPermission(constants.Role(actor.Role), permission) {
				c.Next()
				return
			}
		}
		util.Fail(c, util.NewError(http.StatusForbidden, util.CodeForbidden, "role does not have permission for this action"))
	}
}
