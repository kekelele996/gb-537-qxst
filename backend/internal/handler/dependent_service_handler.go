package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/service"
	"pki-certificate-rollover-impact/backend/internal/util"
)

type DependentServiceHandler struct {
	service *service.DependentServiceService
}

func NewDependentServiceHandler(value *service.DependentServiceService) *DependentServiceHandler {
	return &DependentServiceHandler{service: value}
}
func (h *DependentServiceHandler) List(c *gin.Context) {
	chainID, ok := optionalUint(c, "chain_id")
	if !ok {
		return
	}
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.DependentServiceQuery{Search: c.Query("search"), Environment: c.Query("environment"), State: c.Query("state"), ChainID: chainID, Page: page, PageSize: size})
	respond(c, http.StatusOK, result, err)
}
func (h *DependentServiceHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *DependentServiceHandler) Create(c *gin.Context) {
	var request dto.CreateDependentServiceRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Create(c.Request.Context(), request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *DependentServiceHandler) Update(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateDependentServiceRequest
	if !bindJSON(c, &request) {
		return
	}
	result, serviceErr := h.service.Update(c.Request.Context(), id, request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *DependentServiceHandler) Deactivate(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Deactivate(c.Request.Context(), id, mustActor(c), util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}
