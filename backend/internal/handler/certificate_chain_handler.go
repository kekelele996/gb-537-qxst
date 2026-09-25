package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"pki-certificate-rollover-impact/backend/internal/dto"
	"pki-certificate-rollover-impact/backend/internal/service"
	"pki-certificate-rollover-impact/backend/internal/util"
)

type CertificateChainHandler struct {
	service *service.CertificateChainService
}

func NewCertificateChainHandler(value *service.CertificateChainService) *CertificateChainHandler {
	return &CertificateChainHandler{service: value}
}
func (h *CertificateChainHandler) List(c *gin.Context) {
	anchorID, ok := optionalUint(c, "trust_anchor_id")
	if !ok {
		return
	}
	page, size := util.Pagination(c)
	result, err := h.service.List(c.Request.Context(), dto.CertificateChainQuery{Search: c.Query("search"), TrustAnchorID: anchorID, State: c.Query("state"), Page: page, PageSize: size})
	respond(c, http.StatusOK, result, err)
}
func (h *CertificateChainHandler) Get(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, serviceErr := h.service.Get(c.Request.Context(), id)
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *CertificateChainHandler) Import(c *gin.Context) {
	var request dto.ImportCertificateChainRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Import(c.Request.Context(), request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusCreated, result, err)
}
func (h *CertificateChainHandler) Transition(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.CertificateChainTransitionRequest
	if !bindJSON(c, &request) {
		return
	}
	result, serviceErr := h.service.Transition(c.Request.Context(), id, request, mustActor(c), util.RequestID(c))
	respond(c, http.StatusOK, result, serviceErr)
}
func (h *CertificateChainHandler) Compare(c *gin.Context) {
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
