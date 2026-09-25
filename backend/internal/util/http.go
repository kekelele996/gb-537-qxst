package util

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CodeValidation       = "VALIDATION_FAILED"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeStateTransition  = "INVALID_STATE_TRANSITION"
	CodeIdempotency      = "IDEMPOTENCY_CONFLICT"
	CodeReviewerConflict = "REVIEWER_SEPARATION_REQUIRED"
	CodeInternal         = "INTERNAL_ERROR"
)

type APIError struct {
	Status        int
	Code, Message string
	Cause         error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}
func (e *APIError) Unwrap() error { return e.Cause }
func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}
func WrapError(status int, code, message string, cause error) *APIError {
	return &APIError{Status: status, Code: code, Message: message, Cause: cause}
}
func NotFound(entity string) *APIError {
	return NewError(http.StatusNotFound, CodeNotFound, entity+" was not found")
}

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Code: "OK", Message: "success", Data: data, RequestID: RequestID(c)})
}
func Fail(c *gin.Context, err error) {
	apiErr := &APIError{}
	if !errors.As(err, &apiErr) {
		apiErr = WrapError(http.StatusInternalServerError, CodeInternal, "request could not be completed", err)
	}
	c.Error(apiErr)
	c.AbortWithStatusJSON(apiErr.Status, Envelope{Code: apiErr.Code, Message: apiErr.Message, RequestID: RequestID(c)})
}

type Actor struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Team        string `json:"team"`
	Role        string `json:"role"`
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	id, _ := value.(string)
	return id
}
func ParseUintParam(c *gin.Context, key string) (uint, error) {
	raw := c.Param(key)
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		return 0, NewError(http.StatusBadRequest, CodeValidation, key+" must be a positive integer")
	}
	return uint(parsed), nil
}
func Pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 100
	}
	if size > 200 {
		size = 200
	}
	return page, size
}
func CanonicalJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode canonical JSON: %w", err)
	}
	return string(raw), nil
}
func HashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func HashCanonical(value any) (string, string, error) {
	raw, err := CanonicalJSON(value)
	if err != nil {
		return "", "", err
	}
	return HashString(raw), raw, nil
}
func CompactText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > max {
		return value[:max]
	}
	return value
}
