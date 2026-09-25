package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/constants"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/service"
	"pki-certificate-rollover-impact/backend/internal/util"
)

type RolloverScenarioHandler struct {
	service *service.RolloverScenarioService
}

func NewRolloverScenarioHandler(value *service.RolloverScenarioService) *RolloverScenarioHandler {
	return &RolloverScenarioHandler{service: value}
}
func (h *RolloverScenarioHandler) List(c *gin.Context) {
	creator, ok := optionalUint(c, "created_by")
	if !ok {
		return
	}
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.RolloverScenarioQuery{State: c.Query("state"), CreatedBy: creator, Page: page, PageSize: size})
	respond(c, http.StatusOK, result, err)
}
func (h *RolloverScenarioHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *RolloverScenarioHandler) Create(c *gin.Context) {
	var request dto.CreateRolloverScenarioRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *RolloverScenarioHandler) Simulate(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, reused, serviceErr := h.service.Simulate(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), mustActor(c), util.RequestID(c))
	status := http.StatusCreated
	if reused {
		status = http.StatusOK
	}
	respond(c, status, result, serviceErr)
}
func (h *RolloverScenarioHandler) Transition(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.RolloverScenarioTransitionRequest
	if !bindJSON(c, &request) {
		return
	}
	actor := mustActor(c)
	permission := constants.PermissionScenarioWrite
	if request.ToState == string(constants.ScenarioVerified) {
		permission = constants.PermissionScenarioVerify
	}
	if !constants.HasPermission(constants.Role(actor.Role), permission) {
		if request.ToState == string(constants.ScenarioVerified) {
			scenario, scenarioErr := h.service.Get(c.Request.Context(), id)
			if scenarioErr != nil {
				respond(c, http.StatusOK, nil, scenarioErr)
				return
			}
			if scenario.CreatedBy == actor.UserID {
				util.Fail(c, util.NewError(http.StatusConflict, util.CodeReviewerConflict, "scenario creator cannot verify their own simulation"))
				return
			}
		}
		util.Fail(c, util.NewError(http.StatusForbidden, util.CodeForbidden, "role cannot perform this scenario transition"))
		return
	}
	result, serviceErr := h.service.Transition(c.Request.Context(), id, request, actor, util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *RolloverScenarioHandler) Replay(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Replay(c.Request.Context(), id, mustActor(c), util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *RolloverScenarioHandler) Compare(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	otherID, err := util.ParseUintParam(c, "other_id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Compare(c.Request.Context(), id, otherID)
	respond(c, http.StatusOK, result, serviceErr)
}
