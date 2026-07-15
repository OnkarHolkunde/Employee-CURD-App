package apperrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

// BindEmployeeJSON binds the request body into obj, returning a specific
// field-level *AppError (e.g. "phone must be a string") on failure
func BindEmployeeJSON(c *gin.Context, obj interface{}) *AppError {
	if err := c.ShouldBindJSON(obj); err != nil {
		return jsonBindError(err)
	}
	return nil
}

func jsonBindError(err error) *AppError {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return NewValidation([]FieldError{
			{
				Field:   typeErr.Field,
				Message: fmt.Sprintf("%s must be a %s, got a %s", typeErr.Field, typeErr.Type.String(), typeErr.Value),
			},
		})
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.ErrUnexpectedEOF) {
		return NewBadRequest("request body is not valid JSON")
	}

	if errors.Is(err, io.EOF) {
		return NewBadRequest("request body must not be empty")
	}

	return NewBadRequest("request body must be valid JSON matching the employee schema")
}
