// Package apperrors defines every error the API can return to a client,
// so raw driver/library errors never leak to a consumer.
package apperrors

import "net/http"

// ErrorCode is a stable, machine-readable identifier a client can branch
// on, independent of the (human-readable) Message text.
type ErrorCode string

const (
	CodeValidation     ErrorCode = "VALIDATION_ERROR"
	CodeDuplicateEmail ErrorCode = "DUPLICATE_EMAIL"
	CodeNotFound       ErrorCode = "NOT_FOUND"
	CodeBadRequest     ErrorCode = "BAD_REQUEST"
	CodeInvalidFile    ErrorCode = "INVALID_FILE"
	CodeFileTooLarge   ErrorCode = "FILE_TOO_LARGE"
	CodeInternal       ErrorCode = "INTERNAL_ERROR"
)

// FieldError attaches a validation problem to the specific field that
// caused it, so a client can highlight the right form input.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the only error type that should ever reach the HTTP layer.
type AppError struct {
	Code       ErrorCode    `json:"code"`
	Message    string       `json:"message"`
	Fields     []FieldError `json:"fields,omitempty"`
	HTTPStatus int          `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

// NewValidation reports one or more field-level validation failures.
func NewValidation(fields []FieldError) *AppError {
	return &AppError{
		Code:       CodeValidation,
		Message:    "one or more fields failed validation",
		Fields:     fields,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

// NewDuplicateEmail reports that the given email is already in use by
// another employee record.
func NewDuplicateEmail(email string) *AppError {
	return &AppError{
		Code:    CodeDuplicateEmail,
		Message: "an employee with this email address already exists",
		Fields: []FieldError{
			{Field: "email", Message: "this email address is already in use by another employee"},
		},
		HTTPStatus: http.StatusConflict,
	}
}

// NewNotFound reports that the requested resource does not exist.
func NewNotFound(resource string) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    resource + " not found",
		HTTPStatus: http.StatusNotFound,
	}
}

// NewBadRequest reports a malformed request (bad path param, bad JSON
// body, missing required multipart field, etc.).
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:       CodeBadRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewInvalidFile reports that an uploaded file doesn't meet the expected
// format/schema (wrong extension, bad/missing headers, etc.).
func NewInvalidFile(message string) *AppError {
	return &AppError{
		Code:       CodeInvalidFile,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewFileTooLarge reports that an uploaded file exceeds the size limit.
func NewFileTooLarge(message string) *AppError {
	return &AppError{
		Code:       CodeFileTooLarge,
		Message:    message,
		HTTPStatus: http.StatusRequestEntityTooLarge,
	}
}

// NewInternal is the ONLY error ever shown for an unexpected failure
// (a DB outage, a driver error, a bug). Its message is intentionally
// generic and fixed — the real cause is logged server-side, never
// forwarded to the client.
func NewInternal() *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    "an unexpected error occurred while processing your request",
		HTTPStatus: http.StatusInternalServerError,
	}
}
