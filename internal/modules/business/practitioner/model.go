package practitioner

import "github.com/google/uuid"

type Practitioner struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	Address   string    `db:"address"`
	City      string    `db:"city"`
	State     string    `db:"state"`
	Zip       string    `db:"zip"`
}

type RqCreatePractitioner struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone" validate:"required,e164"`
}

type RsPractitioner struct {
	ID     uuid.UUID `json:"id"`
	UserID string    `json:"user_id"`
	Email  string    `json:"email"`
}

func (p *Practitioner) ToRs() *RsPractitioner {
	return &RsPractitioner{
		ID:     p.ID,
		UserID: p.UserID.String(),
		Email:  p.Email,
	}
}
