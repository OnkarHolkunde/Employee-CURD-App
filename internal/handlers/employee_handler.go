package handlers

import (
	"strconv"

	"excel-crud-app/internal/apperrors"
	"excel-crud-app/internal/models"
	"excel-crud-app/internal/response"
	"excel-crud-app/internal/services"
	"excel-crud-app/internal/validation"

	"github.com/gin-gonic/gin"
)

type EmployeeHandler struct {
	svc *services.EmployeeService
}

func NewEmployeeHandler(svc *services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

// ListEmployees handles GET /api/v1/employees
// Reads from Redis first; on a cache miss it falls back to (and repopulates
// from) MySQL, which is queried using LIMIT/OFFSET for the given page.
func (h *EmployeeHandler) ListEmployees(c *gin.Context) {
	start := parseNonNegativeIntQuery(c, "start", 0)
	limit := min(parsePositiveIntQuery(c, "limit", 50), 500) // guard against accidentally huge responses

	employees, total, source, appErr := h.svc.List(c.Request.Context(), start, limit)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.OKWithMeta(c, "employees retrieved", employees, gin.H{
		"source": source, // "redis" or "mysql", handy for debugging/demo
		"total":  total,
		"start":  start,
		"limit":  limit,
	})
}

// GetEmployee handles GET /api/v1/employees/:id.
func (h *EmployeeHandler) GetEmployee(c *gin.Context) {
	id, appErr := parseIDParam(c)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	emp, source, appErr := h.svc.GetByID(c.Request.Context(), id)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.OKWithMeta(c, "employee retrieved", emp, gin.H{"source": source})
}

// CreateEmployee handles POST /api/v1/employees for manually adding a single record
func (h *EmployeeHandler) CreateEmployee(c *gin.Context) {
	var emp models.Employee
	if appErr := apperrors.BindEmployeeJSON(c, &emp); appErr != nil {
		response.Error(c, appErr)
		return
	}

	if fieldErrs := validation.ValidateEmployeeFull(&emp); len(fieldErrs) > 0 {
		response.Error(c, apperrors.NewValidation(fieldErrs))
		return
	}

	if appErr := h.svc.Create(c.Request.Context(), &emp); appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.Created(c, "employee created", emp)
}

// ReplaceEmployee handles PUT /api/v1/employees/:id.
// PUT is a full replacement: the request body must represent the complete desired state of the resource
func (h *EmployeeHandler) ReplaceEmployee(c *gin.Context) {
	id, appErr := parseIDParam(c)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var emp models.Employee
	if appErr := apperrors.BindEmployeeJSON(c, &emp); appErr != nil {
		response.Error(c, appErr)
		return
	}

	if fieldErrs := validation.ValidateEmployeeFull(&emp); len(fieldErrs) > 0 {
		response.Error(c, apperrors.NewValidation(fieldErrs))
		return
	}

	updated, appErr := h.svc.Replace(c.Request.Context(), id, &emp)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.OK(c, "employee replaced", updated)
}

// PatchEmployee handles PATCH /api/v1/employees/:id.
// PATCH is a partial update: only fields present in the request body are validated and changed.
func (h *EmployeeHandler) PatchEmployee(c *gin.Context) {
	id, appErr := parseIDParam(c)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	var input models.EmployeeUpdateInput
	if appErr := apperrors.BindEmployeeJSON(c, &input); appErr != nil {
		response.Error(c, appErr)
		return
	}

	if fieldErrs := validation.ValidateEmployeePatch(&input); len(fieldErrs) > 0 {
		response.Error(c, apperrors.NewValidation(fieldErrs))
		return
	}

	updated, appErr := h.svc.Patch(c.Request.Context(), id, input)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.OK(c, "employee updated", updated)
}

// DeleteEmployee handles DELETE /api/v1/employees/:id.
func (h *EmployeeHandler) DeleteEmployee(c *gin.Context) {
	id, appErr := parseIDParam(c)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	if appErr := h.svc.Delete(c.Request.Context(), id); appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.OK(c, "employee deleted", nil)
}

// parseIDParam extracts and validates the :id path param shared by every
// single-employee route (GET/PUT/PATCH/DELETE)
func parseIDParam(c *gin.Context) (uint, *apperrors.AppError) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, apperrors.NewBadRequest("employee id must be a positive integer")
	}
	return uint(id), nil
}

// parsePositiveIntQuery reads an integer query param
func parsePositiveIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return fallback
	}
	return v
}

// parseNonNegativeIntQuery reads an integer query param, allowing zero
// (used for "start", where 0 is a valid offset).
func parseNonNegativeIntQuery(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}
