package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"excel-crud-app/internal/apperrors"
	"excel-crud-app/internal/database"
	"excel-crud-app/internal/models"

	"gorm.io/gorm"
)

const (
	employeeKeyPrefix = "employee:"
	listVersionKey    = "employees:list:version"
	batchInsertSize   = 500
)

// employeeListPage is what gets cached in Redis for a given (version, start,
// limit) triple: the page of employees plus the total row count.
type employeeListPage struct {
	Employees []models.Employee `json:"employees"`
	Total     int64             `json:"total"`
}

// listCacheKey scopes a cached page to the current list version so a single
// InvalidateListCache call invalidates every cached page at once: bumping
// the version makes all previously cached keys unreachable (they just expire
// off later via cacheTTL).
func listCacheKey(version int64, start, limit int) string {
	return fmt.Sprintf("employees:list:v%d:s%d:l%d", version, start, limit)
}

// EmployeeService centralizes all MySQL + Redis logic for employees.
type EmployeeService struct {
	db       *gorm.DB
	cacheTTL time.Duration
}

func NewEmployeeService(cacheTTLSeconds int) *EmployeeService {
	return &EmployeeService{
		db:       database.DB,
		cacheTTL: time.Duration(cacheTTLSeconds) * time.Second,
	}
}

func employeeCacheKey(id uint) string {
	return employeeKeyPrefix + itoa64(id)
}

// itoa64 avoids pulling in strconv for this one uint->string conversion.
func itoa64(id uint) string {
	if id == 0 {
		return "0"
	}
	digits := ""
	for id > 0 {
		digits = string(rune('0'+id%10)) + digits
		id /= 10
	}
	return digits
}

// BulkInsert writes parsed employees to MySQL in batches and invalidates
// the list cache. Returns a plain error since it's only called from the
// background import job, not an HTTP handler.
func (s *EmployeeService) BulkInsert(ctx context.Context, employees []models.Employee) (int, error) {
	if len(employees) == 0 {
		return 0, nil
	}

	if err := s.db.WithContext(ctx).CreateInBatches(&employees, batchInsertSize).Error; err != nil {
		return 0, err
	}

	// Best-effort cache invalidation; a stale cache will simply expire
	// within cacheTTL even if this fails, so we don't treat it as fatal.
	_ = s.InvalidateListCache(ctx)

	return len(employees), nil
}

// List returns a page of employees ordered by id, along with the total
// count of (non-deleted) employees in the database.
// It checks the Redis cache first and falls back to (and repopulates from)
// MySQL on a cache miss.
func (s *EmployeeService) List(ctx context.Context, start, limit int) ([]models.Employee, int64, string, *apperrors.AppError) {
	cacheKey := listCacheKey(s.currentListVersion(ctx), start, limit)

	cached, err := database.RDB.Get(ctx, cacheKey).Result()
	if err == nil {
		var page employeeListPage
		if jsonErr := json.Unmarshal([]byte(cached), &page); jsonErr == nil {
			return page.Employees, page.Total, "redis", nil
		}
		// Corrupt cache entry: fall through and rebuild from MySQL.
	}

	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Employee{}).Where("is_deleted = ?", false).Count(&total).Error; err != nil {
		slog.Error("list employees: mysql count failed", "error", err)
		return nil, 0, "", apperrors.NewInternal()
	}

	var employees []models.Employee
	if err := s.db.WithContext(ctx).Where("is_deleted = ?", false).Order("id asc").
		Offset(start).Limit(limit).Find(&employees).Error; err != nil {
		slog.Error("list employees: mysql query failed", "error", err)
		return nil, 0, "", apperrors.NewInternal()
	}

	if payload, jsonErr := json.Marshal(employeeListPage{Employees: employees, Total: total}); jsonErr == nil {
		// Best-effort re-cache; a Redis hiccup shouldn't fail the read.
		_ = database.RDB.Set(ctx, cacheKey, payload, s.cacheTTL).Err()
	}

	return employees, total, "mysql", nil
}

// currentListVersion reads the list cache version, defaulting to 0 if it
// has never been set (e.g. right after startup).
func (s *EmployeeService) currentListVersion(ctx context.Context) int64 {
	v, err := database.RDB.Get(ctx, listVersionKey).Int64()
	if err != nil {
		return 0
	}
	return v
}

// InvalidateListCache bumps the list cache version, which orphans every
// previously cached page (they become unreachable and simply expire off
// later via cacheTTL) without needing to scan/delete individual keys.
func (s *EmployeeService) InvalidateListCache(ctx context.Context) error {
	return database.RDB.Incr(ctx, listVersionKey).Err()
}

// GetByID fetches a single employee, checking the per-record Redis cache
// first before falling back to MySQL.
func (s *EmployeeService) GetByID(ctx context.Context, id uint) (*models.Employee, string, *apperrors.AppError) {
	key := employeeCacheKey(id)

	cached, err := database.RDB.Get(ctx, key).Result()
	if err == nil {
		var emp models.Employee
		if jsonErr := json.Unmarshal([]byte(cached), &emp); jsonErr == nil {
			return &emp, "redis", nil
		}
	}

	var emp models.Employee
	if err := s.db.WithContext(ctx).Where("is_deleted = ?", false).First(&emp, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", apperrors.NewNotFound("employee")
		}
		slog.Error("get employee: mysql query failed", "id", id, "error", err)
		return nil, "", apperrors.NewInternal()
	}

	if payload, jsonErr := json.Marshal(emp); jsonErr == nil {
		_ = database.RDB.Set(ctx, key, payload, s.cacheTTL).Err()
	}

	return &emp, "mysql", nil
}

// Create inserts a single new employee record, rejecting a duplicate
// email with a 409 Conflict.
func (s *EmployeeService) Create(ctx context.Context, emp *models.Employee) *apperrors.AppError {
	if inUse, appErr := s.emailInUse(ctx, emp.Email, 0); appErr != nil {
		return appErr
	} else if inUse {
		return apperrors.NewDuplicateEmail(emp.Email)
	}

	if err := s.db.WithContext(ctx).Create(emp).Error; err != nil {
		slog.Error("create employee: mysql insert failed", "error", err)
		return apperrors.NewInternal()
	}

	_ = s.InvalidateListCache(ctx)
	return nil
}

// Replace implements full PUT semantics: replacement overwrites the whole
// record, so a field omitted from the request is written as its zero value.
func (s *EmployeeService) Replace(ctx context.Context, id uint, replacement *models.Employee) (*models.Employee, *apperrors.AppError) {
	var existing models.Employee
	if err := s.db.WithContext(ctx).Where("is_deleted = ?", false).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("employee")
		}
		slog.Error("replace employee: mysql lookup failed", "id", id, "error", err)
		return nil, apperrors.NewInternal()
	}

	if inUse, appErr := s.emailInUse(ctx, replacement.Email, id); appErr != nil {
		return nil, appErr
	} else if inUse {
		return nil, apperrors.NewDuplicateEmail(replacement.Email)
	}

	// Preserve identity/audit fields; everything else is fully replaced.
	replacement.ID = existing.ID
	replacement.CreatedAt = existing.CreatedAt

	if err := s.db.WithContext(ctx).Save(replacement).Error; err != nil {
		slog.Error("replace employee: mysql save failed", "id", id, "error", err)
		return nil, apperrors.NewInternal()
	}

	s.refreshCaches(ctx, *replacement)

	return replacement, nil
}

// Patch implements partial-update (PATCH) semantics: only non-nil fields
// in input are changed; everything else is left untouched.
func (s *EmployeeService) Patch(ctx context.Context, id uint, input models.EmployeeUpdateInput) (*models.Employee, *apperrors.AppError) {
	var emp models.Employee
	if err := s.db.WithContext(ctx).Where("is_deleted = ?", false).First(&emp, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NewNotFound("employee")
		}
		slog.Error("patch employee: mysql lookup failed", "id", id, "error", err)
		return nil, apperrors.NewInternal()
	}

	if input.Email != nil && *input.Email != "" && !strings.EqualFold(*input.Email, emp.Email) {
		if inUse, appErr := s.emailInUse(ctx, *input.Email, id); appErr != nil {
			return nil, appErr
		} else if inUse {
			return nil, apperrors.NewDuplicateEmail(*input.Email)
		}
	}

	applyPatch(&emp, input)

	if err := s.db.WithContext(ctx).Save(&emp).Error; err != nil {
		slog.Error("patch employee: mysql save failed", "id", id, "error", err)
		return nil, apperrors.NewInternal()
	}

	s.refreshCaches(ctx, emp)

	return &emp, nil
}

// Delete soft-deletes an employee: the row stays in MySQL with
// is_deleted set to true
func (s *EmployeeService) Delete(ctx context.Context, id uint) *apperrors.AppError {
	result := s.db.WithContext(ctx).Model(&models.Employee{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("is_deleted", true)
	if result.Error != nil {
		slog.Error("delete employee: mysql soft-delete failed", "id", id, "error", result.Error)
		return apperrors.NewInternal()
	}
	if result.RowsAffected == 0 {
		return apperrors.NewNotFound("employee")
	}

	_ = database.RDB.Del(ctx, employeeCacheKey(id)).Err()
	_ = s.InvalidateListCache(ctx)

	return nil
}

// refreshCaches writes the just-updated record back into the per-record
// cache and invalidates the list cache.
func (s *EmployeeService) refreshCaches(ctx context.Context, emp models.Employee) {
	if payload, jsonErr := json.Marshal(emp); jsonErr == nil {
		_ = database.RDB.Set(ctx, employeeCacheKey(emp.ID), payload, s.cacheTTL).Err()
	}
	_ = s.InvalidateListCache(ctx)
}

// emailInUse reports whether `email` already belongs to a different
// employee record
func (s *EmployeeService) emailInUse(ctx context.Context, email string, excludeID uint) (bool, *apperrors.AppError) {
	email = strings.TrimSpace(email)
	if email == "" {
		return false, nil
	}

	query := s.db.WithContext(ctx).Model(&models.Employee{}).
		Where("LOWER(email) = LOWER(?)", email).
		Where("is_deleted = ?", false)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		slog.Error("email uniqueness check: mysql query failed", "error", err)
		return false, apperrors.NewInternal()
	}

	return count > 0, nil
}

// applyPatch copies every non-nil field from input onto emp in place.
func applyPatch(emp *models.Employee, input models.EmployeeUpdateInput) {
	if input.FirstName != nil {
		emp.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		emp.LastName = *input.LastName
	}
	if input.CompanyName != nil {
		emp.CompanyName = *input.CompanyName
	}
	if input.Address != nil {
		emp.Address = *input.Address
	}
	if input.City != nil {
		emp.City = *input.City
	}
	if input.County != nil {
		emp.County = *input.County
	}
	if input.Postal != nil {
		emp.Postal = *input.Postal
	}
	if input.Phone != nil {
		emp.Phone = *input.Phone
	}
	if input.Email != nil {
		emp.Email = *input.Email
	}
	if input.Web != nil {
		emp.Web = *input.Web
	}
}
