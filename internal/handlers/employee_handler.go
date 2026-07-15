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

// EmployeeHandler adapts HTTP requests to services.EmployeeService calls.
// It owns request parsing/validation/response-shaping only; all business
// logic (caching, uniqueness checks, persistence) lives in the service.
type EmployeeHandler struct {
	svc *services.EmployeeService
}

func NewEmployeeHandler(svc *services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

// ListEmployees handles GET /api/v1/employees?page=1&page_size=50.
// Reads from Redis first; on a cache miss it falls back to MySQL and
// repopulates the cache, per the assignment's caching requirement.
// Pagination is applied in-memory on top of the cached/DB result so the
// cache still represents "the full imported dataset" as a single entity.
func (h *EmployeeHandler) ListEmployees(c *gin.Context) {
	employees, source, appErr := h.svc.List(c.Request.Context())
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	page := parsePositiveIntQuery(c, "page", 1)
	pageSize := parsePositiveIntQuery(c, "page_size", 50)
	if pageSize > 500 {
		pageSize = 500 // guard against accidentally huge responses
	}

	total := len(employees)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paged := employees[start:end]

	response.OKWithMeta(c, "employees retrieved", paged, gin.H{
		"source":      source, // "redis" or "mysql", handy for debugging/demo
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
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

// CreateEmployee handles POST /api/v1/employees for manually adding a
// single record (independent of the bulk Excel import flow). The payload
// is validated field-by-field via internal/validation before it ever
// reaches the service layer, and a duplicate email is rejected with a
// 409 Conflict rather than a generic error.
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
//
// PUT is a full replacement: the request body must represent the complete
// desired state of the resource (same validation rules as Create), and
// any field the client omits is written as its zero value — exactly as
// PUT semantics require. Use PatchEmployee instead to change only a
// subset of fields.
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
//
// PATCH is a partial update: only fields present in the request body are
// validated and changed. Every field omitted from the body is left
// exactly as it was — this is what makes PATCH an independent method from
// ReplaceEmployee/PUT rather than an alias for it.
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
// single-employee route (GET/PUT/PATCH/DELETE), so each handler doesn't
// repeat the same parse-or-400 boilerplate.
func parseIDParam(c *gin.Context) (uint, *apperrors.AppError) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, apperrors.NewBadRequest("employee id must be a positive integer")
	}
	return uint(id), nil
}

// parsePositiveIntQuery reads an integer query param (page, page_size),
// falling back to fallback when the param is absent or not a valid
// integer, rather than rejecting the request outright — pagination
// params are a convenience, not something worth a 400 over.
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
