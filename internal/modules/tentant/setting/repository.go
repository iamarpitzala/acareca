package setting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("tentant not found")

type Repository interface {
	Create(ctx context.Context, t *Tentant) (*Tentant, error)
	GetByID(ctx context.Context, id int) (*Tentant, error)
	GetByUserID(ctx context.Context, userID string) (*Tentant, error)
	List(ctx context.Context) ([]*Tentant, error)
	Update(ctx context.Context, t *Tentant) (*Tentant, error)
	Delete(ctx context.Context, id int) error

	GetSettingByTentantID(ctx context.Context, tentantID int) (*TentantSetting, error)
	UpsertSetting(ctx context.Context, s *TentantSetting) (*TentantSetting, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, t *Tentant) (*Tentant, error) {
	query := `
		INSERT INTO tbl_tentant (user_id, abn, verifed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, abn, verifed, created_at, updated_at, deleted_at
	`
	now := time.Now()
	var out Tentant
	if err := r.db.QueryRowxContext(ctx, query,
		t.UserID, t.ABN, t.Verifed, now, now,
	).StructScan(&out); err != nil {
		return nil, fmt.Errorf("create tentant: %w", err)
	}
	return &out, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*Tentant, error) {
	query := `
		SELECT id, user_id, abn, verifed, created_at, updated_at, deleted_at
		FROM tbl_tentant
		WHERE id = $1 AND deleted_at IS NULL
	`
	var t Tentant
	if err := r.db.QueryRowxContext(ctx, query, id).StructScan(&t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get tentant: %w", err)
	}
	return &t, nil
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*Tentant, error) {
	query := `
		SELECT id, user_id, abn, verifed, created_at, updated_at, deleted_at
		FROM tbl_tentant
		WHERE user_id = $1 AND deleted_at IS NULL
	`
	var t Tentant
	if err := r.db.QueryRowxContext(ctx, query, userID).StructScan(&t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get tentant by user_id: %w", err)
	}
	return &t, nil
}

func (r *repository) List(ctx context.Context) ([]*Tentant, error) {
	query := `
		SELECT id, user_id, abn, verifed, created_at, updated_at, deleted_at
		FROM tbl_tentant
		WHERE deleted_at IS NULL
		ORDER BY id
	`
	var list []*Tentant
	if err := r.db.SelectContext(ctx, &list, query); err != nil {
		return nil, fmt.Errorf("list tentants: %w", err)
	}
	return list, nil
}

func (r *repository) Update(ctx context.Context, t *Tentant) (*Tentant, error) {
	query := `
		UPDATE tbl_tentant
		SET abn = $2, verifed = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, abn, verifed, created_at, updated_at, deleted_at
	`
	var out Tentant
	if err := r.db.QueryRowxContext(ctx, query, t.ID, t.ABN, t.Verifed, t.UpdatedAt).StructScan(&out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update tentant: %w", err)
	}
	return &out, nil
}

func (r *repository) Delete(ctx context.Context, id int) error {
	query := `UPDATE tbl_tentant SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tentant: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) GetSettingByTentantID(ctx context.Context, tentantID int) (*TentantSetting, error) {
	query := `
		SELECT id, tentant_id, timezone, logo, color, created_at, updated_at, deleted_at
		FROM tbl_tentant_setting
		WHERE tentant_id = $1 AND deleted_at IS NULL
	`
	var s TentantSetting
	if err := r.db.QueryRowxContext(ctx, query, tentantID).StructScan(&s); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get tentant setting: %w", err)
	}
	return &s, nil
}

func (r *repository) UpsertSetting(ctx context.Context, s *TentantSetting) (*TentantSetting, error) {
	// Try update first if record exists
	_, err := r.GetSettingByTentantID(ctx, s.TentantID)
	if err == nil {
		query := `
			UPDATE tbl_tentant_setting
			SET timezone = $2, logo = $3, color = $4, updated_at = $5
			WHERE tentant_id = $1 AND deleted_at IS NULL
			RETURNING id, tentant_id, timezone, logo, color, created_at, updated_at, deleted_at
		`
		var out TentantSetting
		if err := r.db.QueryRowxContext(ctx, query, s.TentantID, s.Timezone, s.Logo, s.Color, s.UpdatedAt).StructScan(&out); err != nil {
			return nil, fmt.Errorf("update tentant setting: %w", err)
		}
		return &out, nil
	}
	// Insert new
	query := `
		INSERT INTO tbl_tentant_setting (tentant_id, timezone, logo, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tentant_id, timezone, logo, color, created_at, updated_at, deleted_at
	`
	now := time.Now()
	var out TentantSetting
	if err := r.db.QueryRowxContext(ctx, query, s.TentantID, s.Timezone, s.Logo, s.Color, now, now).StructScan(&out); err != nil {
		return nil, fmt.Errorf("create tentant setting: %w", err)
	}
	return &out, nil
}
