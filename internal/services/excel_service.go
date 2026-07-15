package services

import (
	"fmt"
	"strings"

	"excel-crud-app/internal/models"

	"github.com/xuri/excelize/v2"
)

// ExpectedHeaders are the column headers we expect to find in the first row
// of the Excel file.
var ExpectedHeaders = []string{
	"first_name",
	"last_name",
	"company_name",
	"address",
	"city",
	"county",
	"postal",
	"phone",
	"email",
	"web",
}

// RowError captures a row-level parsing/validation problem so the caller
// can report partial success instead of failing the whole import.
type RowError struct {
	Row     int
	Message string
}

func (e RowError) String() string {
	return fmt.Sprintf("row %d: %s", e.Row, e.Message)
}

// ValidateHeaders checks that the first row of the sheet contains exactly
// the expected columns (order-independent), returning a descriptive error
// if the file doesn't match the expected schema.
func ValidateHeaders(header []string) error {
	if len(header) < len(ExpectedHeaders) {
		return fmt.Errorf(
			"expected %d columns (%s) but got %d",
			len(ExpectedHeaders), strings.Join(ExpectedHeaders, ", "), len(header),
		)
	}

	normalized := make(map[string]int, len(header))
	for i, h := range header {
		normalized[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var missing []string
	for _, want := range ExpectedHeaders {
		if _, ok := normalized[want]; !ok {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required column(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

// ParseExcel opens the workbook at path, validates its header row against
// ExpectedHeaders, and converts each subsequent row into an Employee.
// Rows that fail basic validation (missing first/last name) are skipped
// and reported in rowErrors rather than aborting the whole import.
func ParseExcel(path string) (employees []models.Employee, rowErrors []RowError, err error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("could not open excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, nil, fmt.Errorf("workbook does not contain any sheets")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read sheet %q: %w", sheetName, err)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("sheet %q is empty", sheetName)
	}

	header := rows[0]
	if err := ValidateHeaders(header); err != nil {
		return nil, nil, err
	}

	// Map expected column name -> its actual index in this file (order
	// independent, in case columns were reordered but headers still valid).
	colIndex := make(map[string]int, len(header))
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	get := func(row []string, key string) string {
		idx, ok := colIndex[key]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	for i, row := range rows[1:] {
		rowNum := i + 2 // +1 for header, +1 for 1-indexing
		if len(row) == 0 {
			continue // skip fully blank rows
		}

		emp := models.Employee{
			FirstName:   get(row, "first_name"),
			LastName:    get(row, "last_name"),
			CompanyName: get(row, "company_name"),
			Address:     get(row, "address"),
			City:        get(row, "city"),
			County:      get(row, "county"),
			Postal:      get(row, "postal"),
			Phone:       get(row, "phone"),
			Email:       get(row, "email"),
			Web:         get(row, "web"),
		}

		if emp.FirstName == "" || emp.LastName == "" {
			rowErrors = append(rowErrors, RowError{
				Row:     rowNum,
				Message: "missing required field(s): first_name and/or last_name",
			})
			continue
		}

		employees = append(employees, emp)
	}

	return employees, rowErrors, nil
}
