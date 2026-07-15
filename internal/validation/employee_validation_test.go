package validation

import (
	"strings"
	"testing"

	"excel-crud-app/internal/models"
)

func TestValidateEmployeeFull_ValidRecord(t *testing.T) {
	emp := &models.Employee{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@example.com",
	}
	if errs := ValidateEmployeeFull(emp); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got: %+v", errs)
	}
}

func TestValidateEmployeeFull_MissingRequiredFields(t *testing.T) {
	emp := &models.Employee{}
	errs := ValidateEmployeeFull(emp)

	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing first_name/last_name")
	}

	var gotFirstName, gotLastName bool
	for _, e := range errs {
		switch e.Field {
		case "first_name":
			gotFirstName = true
			if !strings.Contains(e.Message, "required") {
				t.Errorf("expected 'required' message for first_name, got: %q", e.Message)
			}
		case "last_name":
			gotLastName = true
		}
	}
	if !gotFirstName || !gotLastName {
		t.Fatalf("expected errors for both first_name and last_name, got: %+v", errs)
	}
}

func TestValidateEmployeeFull_InvalidEmailFormat(t *testing.T) {
	emp := &models.Employee{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "not-an-email",
	}
	errs := ValidateEmployeeFull(emp)

	found := false
	for _, e := range errs {
		if e.Field == "email" {
			found = true
			if !strings.Contains(e.Message, "valid email") {
				t.Errorf("expected email format message, got: %q", e.Message)
			}
		}
	}
	if !found {
		t.Fatal("expected an email validation error for an invalid address")
	}
}

func TestValidateEmployeeFull_EmailOptional(t *testing.T) {
	emp := &models.Employee{FirstName: "Ada", LastName: "Lovelace"}
	if errs := ValidateEmployeeFull(emp); len(errs) != 0 {
		t.Fatalf("expected no errors when email is omitted, got: %+v", errs)
	}
}

func TestValidateEmployeePatch_OnlyValidatesSuppliedFields(t *testing.T) {
	city := "London"
	input := &models.EmployeeUpdateInput{City: &city}

	if errs := ValidateEmployeePatch(input); len(errs) != 0 {
		t.Fatalf("expected no errors when only an optional field is supplied, got: %+v", errs)
	}
}

func TestValidateEmployeePatch_RejectsBlankingRequiredName(t *testing.T) {
	empty := ""
	input := &models.EmployeeUpdateInput{FirstName: &empty}

	errs := ValidateEmployeePatch(input)
	if len(errs) == 0 {
		t.Fatal("expected an error when first_name is explicitly set to empty")
	}
	if errs[0].Field != "first_name" || !strings.Contains(errs[0].Message, "cannot be empty") {
		t.Errorf("expected a 'cannot be empty' message, got: %+v", errs[0])
	}
}

func TestValidateEmployeePatch_InvalidEmailFormat(t *testing.T) {
	badEmail := "nope"
	input := &models.EmployeeUpdateInput{Email: &badEmail}

	errs := ValidateEmployeePatch(input)
	if len(errs) == 0 {
		t.Fatal("expected an error for an invalid email in a patch payload")
	}
}
