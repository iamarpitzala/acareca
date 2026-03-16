package accountant

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetAllUsers(ctx context.Context) ([]RsAccountantUser, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetAllUsers(ctx context.Context) ([]RsAccountantUser, error) {
	var users []RsAccountantUser

	query := `
		SELECT id, email, first_name, last_name, phone, is_superadmin, created_at, updated_at
		FROM tbl_user 
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}
	return users, nil
}
