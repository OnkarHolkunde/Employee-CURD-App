// Package response provides a consistent JSON envelope for every API
// response, success or error.
package response

import (
	"net/http"

	"excel-crud-app/internal/apperrors"

	"github.com/gin-gonic/gin"
)

// Envelope is the single response shape every endpoint returns.
type Envelope struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    interface{}            `json:"data,omitempty"`
	Meta    interface{}            `json:"meta,omitempty"`
	Code    apperrors.ErrorCode    `json:"code,omitempty"`
	Fields  []apperrors.FieldError `json:"fields,omitempty"`
}

// OK writes a 200 success envelope.
func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data})
}

// OKWithMeta is OK plus a Meta payload (e.g. pagination info).
func OKWithMeta(c *gin.Context, message string, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data, Meta: meta})
}

// Created writes a 201 success envelope for a newly created resource.
func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

// Accepted writes a 202 success envelope for queued-but-not-done work.
func Accepted(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusAccepted, Envelope{Success: true, Message: message, Data: data})
}

// Error writes an error response entirely from an *apperrors.AppError.
func Error(c *gin.Context, err *apperrors.AppError) {
	if err == nil {
		err = apperrors.NewInternal()
	}
	c.JSON(err.HTTPStatus, Envelope{
		Success: false,
		Message: err.Message,
		Code:    err.Code,
		Fields:  err.Fields,
	})
}
