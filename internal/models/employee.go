package models

import "time"

// Employee mirrors the columns in the source Excel file (first_name
// through web).
type Employee struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FirstName   string    `json:"first_name" gorm:"type:varchar(100);not null" binding:"required"`
	LastName    string    `json:"last_name" gorm:"type:varchar(100);not null" binding:"required"`
	CompanyName string    `json:"company_name" gorm:"type:varchar(150)"`
	Address     string    `json:"address" gorm:"type:varchar(255)"`
	City        string    `json:"city" gorm:"type:varchar(100)"`
	County      string    `json:"county" gorm:"type:varchar(100)"`
	Postal      string    `json:"postal" gorm:"type:varchar(20)"`
	Phone       string    `json:"phone" gorm:"type:varchar(30)"`
	Email       string    `json:"email" gorm:"type:varchar(150);index" binding:"omitempty,email"`
	Web         string    `json:"web" gorm:"type:varchar(255)"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName pins the table name explicitly rather than relying on GORM's
// pluralization guesses.
func (Employee) TableName() string {
	return "employees"
}

// EmployeeUpdateInput is the PATCH/PUT payload; only non-nil fields are applied.
type EmployeeUpdateInput struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	CompanyName *string `json:"company_name"`
	Address     *string `json:"address"`
	City        *string `json:"city"`
	County      *string `json:"county"`
	Postal      *string `json:"postal"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email" binding:"omitempty,email"`
	Web         *string `json:"web"`
}
