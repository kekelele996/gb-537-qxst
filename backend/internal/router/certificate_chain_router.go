package router

import (
	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/handler"
	"pki-certificate-rollover-impact/backend/internal/middleware"
)

func RegisterCertificateChainRoutes(api *gin.RouterGroup, h *handler.CertificateChainHandler, limit *middleware.RateLimiter) {
	group := api.Group("/certificate-chains")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.POST("", middleware.RequirePermission(constants.PermissionChainWrite), limit.Middleware("certificate-chain-import"), h.Import)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.POST("/:id/transition", middleware.RequirePermission(constants.PermissionChainWrite), h.Transition)
	group.GET("/:id/compare/:other_id", middleware.RequirePermission(constants.PermissionRead), h.Compare)
}
