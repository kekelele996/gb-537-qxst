package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/middleware"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/service"
	"pki-certificate-rollover-impact/backend/internal/util"
)

func respond(c *gin.Context, status int, data any, err error) {
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Success(c, status, data)
}
func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, "request body is malformed or incomplete"))
		return false
	}
	return true
}
func mustActor(c *gin.Context) util.Actor {
	actor, ok := middleware.Actor(c)
	if !ok {
		util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "authentication context is missing"))
	}
	return actor
}
func optionalUint(c *gin.Context, key string) (uint, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, key+" must be a positive integer"))
		return 0, false
	}
	return uint(value), true
}
func optionalTime(c *gin.Context, key string) (*time.Time, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, key+" must use RFC3339"))
		return nil, false
	}
	value = value.UTC()
	return &value, true
}

type TrustAnchorHandler struct{ service *service.TrustAnchorService }

func NewTrustAnchorHandler(value *service.TrustAnchorService) *TrustAnchorHandler {
	return &TrustAnchorHandler{service: value}
}
func (h *TrustAnchorHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	query := dto.TrustAnchorQuery{Search: c.Query("search"), State: c.Query("state"), Page: page, PageSize: size}
	if raw := c.Query("archived"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, "archived must be true or false"))
			return
		}
		query.Archived = &value
	}
	result, err := h.service.List(c.Request.Context(), query)
	respond(c, http.StatusOK, result, err)
}
func (h *TrustAnchorHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *TrustAnchorHandler) Import(c *gin.Context) {
	var request dto.ImportTrustAnchorRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Import(c.Request.Context(), request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *TrustAnchorHandler) Lifecycle(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.TrustAnchorLifecycleRequest
	if !bindJSON(c, &request) {
		return
	}
	result, serviceErr := h.service.Lifecycle(c.Request.Context(), id, request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}

type AuditHandler struct{ service *service.AuditService }

func NewAuditHandler(value *service.AuditService) *AuditHandler { return &AuditHandler{service: value} }
func (h *AuditHandler) List(c *gin.Context) {
	page, size := util.Pagination(c)
	from, ok := optionalTime(c, "from")
	if !ok {
		return
	}
	to, ok := optionalTime(c, "to")
	if !ok {
		return
	}
	result, err := h.service.List(c.Request.Context(), repository.AuditQuery{EntityType: c.Query("entity_type"), RequestID: c.Query("request_id"), Actor: c.Query("actor"), Action: c.Query("action"), From: from, To: to, Page: page, PageSize: size})
	respond(c, http.StatusOK, result, err)
}
