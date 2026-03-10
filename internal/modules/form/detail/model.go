package detail

import (
	"github.com/google/uuid"
)

type Detail struct {
	ClinicID    uuid.UUID `db:"clinic_id" json:"clinic_id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Status      string    `db:"status" json:"status"`
	Method      string    `db:"method" json:"method"`
	OwnerShare  int       `db:"owner_share" json:"owner_share"`
	ClinicShare int       `db:"clinic_share" json:"clinic_share"`
}
