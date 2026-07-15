package services

import "testing"

func TestValidateHeaders_Valid(t *testing.T) {
	header := []string{
		"first_name", "last_name", "company_name", "address",
		"city", "county", "postal", "phone", "email", "web",
	}
	if err := ValidateHeaders(header); err != nil {
		t.Fatalf("expected valid headers to pass, got error: %v", err)
	}
}

func TestValidateHeaders_CaseInsensitiveAndReordered(t *testing.T) {
	header := []string{
		"Email", "First_Name", "Last_Name", "Company_Name",
		"Address", "City", "County", "Postal", "Phone", "Web",
	}
	if err := ValidateHeaders(header); err != nil {
		t.Fatalf("expected reordered/case-insensitive headers to pass, got error: %v", err)
	}
}

func TestValidateHeaders_MissingColumn(t *testing.T) {
	header := []string{
		"first_name", "last_name", "company_name", "address",
		"city", "county", "postal", "phone", "email",
		// "web" intentionally omitted
	}
	if err := ValidateHeaders(header); err == nil {
		t.Fatal("expected missing 'web' column to fail validation")
	}
}

func TestValidateHeaders_TooFewColumns(t *testing.T) {
	header := []string{"first_name", "last_name"}
	if err := ValidateHeaders(header); err == nil {
		t.Fatal("expected too-few-columns header to fail validation")
	}
}
