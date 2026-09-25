package router

import (
	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/handler"
	"pki-certificate-rollover-impact/backend/internal/middleware"
)

func RegisterDependentServiceRoutes(api *gin.RouterGroup, h *handler.DependentServiceHandler) {
	group := api.Group("/dependent-services")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.POST("", middleware.RequirePermission(constants.PermissionDependencyWrite), h.Create)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.PUT("/:id", middleware.RequirePermission(constants.PermissionDependencyWrite), h.Update)
	group.POST("/:id/deactivate", middleware.RequirePermission(constants.PermissionDependencyWrite), h.Deactivate)
}
