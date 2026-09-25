package router

import (
	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/handler"
	"pki-certificate-rollover-impact/backend/internal/middleware"
)

func RegisterTrustAnchorRoutes(api *gin.RouterGroup, h *handler.TrustAnchorHandler, limit *middleware.RateLimiter) {
	group := api.Group("/trust-anchors")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.POST("", middleware.RequirePermission(constants.PermissionAnchorWrite), limit.Middleware("trust-anchor-import"), h.Import)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.POST("/:id/lifecycle", middleware.RequirePermission(constants.PermissionAnchorWrite), h.Lifecycle)
}
