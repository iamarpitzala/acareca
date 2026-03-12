package form

import (
	"github.com/jmoiron/sqlx"
)

type IRepository interface {
	// Sync(ctx context.Context, req *RqBulkSyncFields) (*RsBulkSyncFields, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) IRepository {
	return &repository{db: db}
}
