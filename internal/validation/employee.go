// Package validation holds field-level validation rules for the Employee
// resource, returning apperrors.FieldError with our own messages.
package validation

import (
	"net/mail"
	"strings"

	"excel-crud-app/internal/apperrors"
	"excel-crud-app/internal/models"
)

const (
	maxNameLength    = 100
	maxCompanyLength = 150
	maxAddressLength = 255
	maxCityLength    = 100
	maxCountyLength  = 100
	maxPostalLength  = 20
	maxPhoneLength   = 30
	maxEmailLength   = 150
	maxWebLength     = 255
)

// ValidateEmployeeFull validates a complete Employee record (POST/PUT).
// first_name and last_name are required; everything else is optional.
func ValidateEmployeeFull(emp *models.Employee) []apperrors.FieldError {
	var errs []apperrors.FieldError

	if strings.TrimSpace(emp.FirstName) == "" {
		errs = append(errs, apperrors.FieldError{Field: "first_name", Message: "first_name is required"})
	} else if len(emp.FirstName) > maxNameLength {
		errs = append(errs, apperrors.FieldError{Field: "first_name", Message: "first_name must be at most 100 characters"})
	}

	if strings.TrimSpace(emp.LastName) == "" {
		errs = append(errs, apperrors.FieldError{Field: "last_name", Message: "last_name is required"})
	} else if len(emp.LastName) > maxNameLength {
		errs = append(errs, apperrors.FieldError{Field: "last_name", Message: "last_name must be at most 100 characters"})
	}

	if emp.Email != "" {
		errs = append(errs, validateEmailField(emp.Email)...)
	}

	errs = append(errs, validateOptionalLength("company_name", emp.CompanyName, maxCompanyLength)...)
	errs = append(errs, validateOptionalLength("address", emp.Address, maxAddressLength)...)
	errs = append(errs, validateOptionalLength("city", emp.City, maxCityLength)...)
	errs = append(errs, validateOptionalLength("county", emp.County, maxCountyLength)...)
	errs = append(errs, validateOptionalLength("postal", emp.Postal, maxPostalLength)...)
	errs = append(errs, validateOptionalLength("phone", emp.Phone, maxPhoneLength)...)
	errs = append(errs, validateOptionalLength("web", emp.Web, maxWebLength)...)

	return errs
}

// ValidateEmployeePatch validates only the fields present in a partial
// (PATCH) payload; an omitted field is never touched.
func ValidateEmployeePatch(input *models.EmployeeUpdateInput) []apperrors.FieldError {
	var errs []apperrors.FieldError

	if input.FirstName != nil {
		if strings.TrimSpace(*input.FirstName) == "" {
			errs = append(errs, apperrors.FieldError{Field: "first_name", Message: "first_name cannot be empty"})
		} else if len(*input.FirstName) > maxNameLength {
			errs = append(errs, apperrors.FieldError{Field: "first_name", Message: "first_name must be at most 100 characters"})
		}
	}

	if input.LastName != nil {
		if strings.TrimSpace(*input.LastName) == "" {
			errs = append(errs, apperrors.FieldError{Field: "last_name", Message: "last_name cannot be empty"})
		} else if len(*input.LastName) > maxNameLength {
			errs = append(errs, apperrors.FieldError{Field: "last_name", Message: "last_name must be at most 100 characters"})
		}
	}

	if input.Email != nil && *input.Email != "" {
		errs = append(errs, validateEmailField(*input.Email)...)
	}

	if input.CompanyName != nil {
		errs = append(errs, validateOptionalLength("company_name", *input.CompanyName, maxCompanyLength)...)
	}
	if input.Address != nil {
		errs = append(errs, validateOptionalLength("address", *input.Address, maxAddressLength)...)
	}
	if input.City != nil {
		errs = append(errs, validateOptionalLength("city", *input.City, maxCityLength)...)
	}
	if input.County != nil {
		errs = append(errs, validateOptionalLength("county", *input.County, maxCountyLength)...)
	}
	if input.Postal != nil {
		errs = append(errs, validateOptionalLength("postal", *input.Postal, maxPostalLength)...)
	}
	if input.Phone != nil {
		errs = append(errs, validateOptionalLength("phone", *input.Phone, maxPhoneLength)...)
	}
	if input.Web != nil {
		errs = append(errs, validateOptionalLength("web", *input.Web, maxWebLength)...)
	}

	return errs
}

// validateEmailField is shared by ValidateEmployeeFull and ValidateEmployeePatch.
func validateEmailField(email string) []apperrors.FieldError {
	var errs []apperrors.FieldError

	if _, err := mail.ParseAddress(email); err != nil {
		errs = append(errs, apperrors.FieldError{Field: "email", Message: "email must be a valid email address"})
		return errs // don't bother with a length check on a format we already rejected
	}
	if len(email) > maxEmailLength {
		errs = append(errs, apperrors.FieldError{Field: "email", Message: "email must be at most 150 characters"})
	}
	return errs
}

// validateOptionalLength is the shared max-length check for optional
// fields that are still column-bounded in MySQL (varchar(N)).
func validateOptionalLength(field, value string, max int) []apperrors.FieldError {
	if len(value) > max {
		return []apperrors.FieldError{
			{Field: field, Message: field + " must be at most " + itoa(max) + " characters"},
		}
	}
	return nil
}

// itoa avoids pulling in strconv just for this one conversion.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
