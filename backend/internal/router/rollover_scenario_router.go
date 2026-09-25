package router

import (
	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/handler"
	"pki-certificate-rollover-impact/backend/internal/middleware"
)

func RegisterRolloverScenarioRoutes(api *gin.RouterGroup, h *handler.RolloverScenarioHandler, limit *middleware.RateLimiter) {
	group := api.Group("/rollover-scenarios")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.POST("", middleware.RequirePermission(constants.PermissionScenarioWrite), h.Create)
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)
	group.POST("/:id/simulate", middleware.RequirePermission(constants.PermissionScenarioRun), limit.Middleware("rollover-simulation"), h.Simulate)
	group.POST("/:id/transition", middleware.RequireAnyPermission(constants.PermissionScenarioWrite, constants.PermissionScenarioVerify), h.Transition)
	group.POST("/:id/replay", middleware.RequirePermission(constants.PermissionScenarioRun), limit.Middleware("rollover-replay"), h.Replay)
	group.GET("/:id/compare/:other_id", middleware.RequirePermission(constants.PermissionRead), h.Compare)
}
