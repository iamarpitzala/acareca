package accountant

import (
	"time"

	"github.com/google/uuid"
)

type RsAccountantUser struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	Email        string    `json:"email"         db:"email"`
	FirstName    string    `json:"first_name"    db:"first_name"`
	LastName     string    `json:"last_name"     db:"last_name"`
	Phone        string    `json:"phone"         db:"phone"`
	IsSuperadmin bool      `json:"is_superadmin" db:"is_superadmin"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}
