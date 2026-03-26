package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/shared/common"
	"github.com/iamarpitzala/acareca/internal/shared/util"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	CreateNotification(ctx context.Context, recipientEntityID uuid.UUID, senderEntityID *uuid.UUID, eventType EventType, entityType EntityType, entityID uuid.UUID, payload NotificationPayload) error

	ListByRecipient(ctx context.Context, recipientEntityID uuid.UUID, filter FilterNotification) (*util.RsList, error)

	MarkRead(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error
	MarkDismissed(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error

	// Recipient resolution helpers (entity ID → entity ID, no user lookup needed)
	GetPractitionerIDByInviteID(ctx context.Context, inviteID uuid.UUID) (*uuid.UUID, error)
	GetPractitionerIDByClinicID(ctx context.Context, clinicID uuid.UUID) (*uuid.UUID, error)
	GetPractitionerIDByFormID(ctx context.Context, formID uuid.UUID) (*uuid.UUID, error)
	GetPractitionerIDByEntryID(ctx context.Context, entryID uuid.UUID) (*uuid.UUID, error)
	GetAccountantEntityIDByEmail(ctx context.Context, email string) (*uuid.UUID, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateNotification(ctx context.Context, recipientEntityID uuid.UUID, senderEntityID *uuid.UUID, eventType EventType, entityType EntityType, entityID uuid.UUID, payload NotificationPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}

	const q = `
		INSERT INTO tbl_notification (
			recipient_id, sender_id, event_type, entity_type, entity_id, status, payload
		) VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
	`
	_, err = r.db.ExecContext(ctx, q, recipientEntityID, senderEntityID, eventType, entityType, entityID, payloadBytes)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (r *repository) ListByRecipient(ctx context.Context, recipientEntityID uuid.UUID, filter FilterNotification) (*util.RsList, error) {
	baseFilters := map[string]interface{}{
		"recipient_id": recipientEntityID,
	}
	if filter.Status != nil && *filter.Status != "" {
		baseFilters["status"] = *filter.Status
	}

	countFilter := common.NewFilter(nil, baseFilters, nil, nil, nil)

	countBase := `FROM tbl_notification`
	totalQuery, totalArgs := common.BuildQuery(countBase, countFilter, allowedColumns, nil, true)
	var total int
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(totalQuery), totalArgs...); err != nil {
		return nil, fmt.Errorf("count notifications: %w", err)
	}

	listBase := `SELECT id, recipient_id, sender_id, event_type, entity_type, entity_id, status, payload, retry_count, created_at, readed_at FROM tbl_notification`
	mergedFilter := common.NewFilter(filter.Filter.Search, baseFilters, nil, filter.Filter.Limit, filter.Filter.Offset)
	mergedFilter.SortBy = filter.Filter.SortBy
	mergedFilter.OrderBy = filter.Filter.OrderBy
	listQuery, listArgs := common.BuildQuery(listBase, mergedFilter, allowedColumns, nil, false)

	var items []Notification
	if err := r.db.SelectContext(ctx, &items, r.db.Rebind(listQuery), listArgs...); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	var rs util.RsList
	rs.MapToList(items, total, *mergedFilter.Offset, *mergedFilter.Limit)
	return &rs, nil
}

func (r *repository) MarkRead(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error {
	const q = `
		UPDATE tbl_notification
		SET status = 'READ', readed_at = NOW()
		WHERE id = $1 AND recipient_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, notificationID, recipientEntityID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("notification not found")
	}
	return nil
}

func (r *repository) MarkDismissed(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error {
	const q = `
		UPDATE tbl_notification
		SET status = 'DISMISSED'
		WHERE id = $1 AND recipient_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, notificationID, recipientEntityID)
	if err != nil {
		return fmt.Errorf("mark dismissed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("notification not found")
	}
	return nil
}

// GetAccountantEntityIDByEmail resolves an accountant's entity ID from their email.
// Used when the invitee (accountant) is the actor and we only have their email.
func (r *repository) GetAccountantEntityIDByEmail(ctx context.Context, email string) (*uuid.UUID, error) {
	var entityID uuid.UUID
	const q = `
		SELECT a.id
		FROM tbl_accountant a
		JOIN tbl_user u ON u.id = a.user_id
		WHERE u.email = $1
		  AND a.deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.QueryRowxContext(ctx, q, email).Scan(&entityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get accountant entity id by email: %w", err)
	}
	return &entityID, nil
}

// GetPractitionerIDByInviteID resolves the practitioner entity ID from an invite.
func (r *repository) GetPractitionerIDByInviteID(ctx context.Context, inviteID uuid.UUID) (*uuid.UUID, error) {
	var practitionerID uuid.UUID
	const q = `
		SELECT practitioner_id
		FROM tbl_invitation
		WHERE id = $1
		LIMIT 1
	`
	err := r.db.QueryRowxContext(ctx, q, inviteID).Scan(&practitionerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get practitioner id by invite id: %w", err)
	}
	return &practitionerID, nil
}

func (r *repository) GetPractitionerIDByClinicID(ctx context.Context, clinicID uuid.UUID) (*uuid.UUID, error) {
	var practitionerID uuid.UUID
	const q = `
		SELECT practitioner_id
		FROM tbl_clinic
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.QueryRowxContext(ctx, q, clinicID).Scan(&practitionerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get practitioner id by clinic id: %w", err)
	}
	return &practitionerID, nil
}

func (r *repository) GetPractitionerIDByFormID(ctx context.Context, formID uuid.UUID) (*uuid.UUID, error) {
	var practitionerID uuid.UUID
	const q = `
		SELECT c.practitioner_id
		FROM tbl_form f
		JOIN tbl_clinic c ON c.id = f.clinic_id
		WHERE f.id = $1
		  AND c.deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.QueryRowxContext(ctx, q, formID).Scan(&practitionerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get practitioner id by form id: %w", err)
	}
	return &practitionerID, nil
}

func (r *repository) GetPractitionerIDByEntryID(ctx context.Context, entryID uuid.UUID) (*uuid.UUID, error) {
	var practitionerID uuid.UUID
	const q = `
		SELECT c.practitioner_id
		FROM tbl_form_entry e
		JOIN tbl_clinic c ON c.id = e.clinic_id
		WHERE e.id = $1
		  AND e.deleted_at IS NULL
		  AND c.deleted_at IS NULL
		LIMIT 1
	`
	err := r.db.QueryRowxContext(ctx, q, entryID).Scan(&practitionerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get practitioner id by entry id: %w", err)
	}
	return &practitionerID, nil
}
