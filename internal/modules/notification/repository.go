package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound           = errors.New("notification not found")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrMaxRetriesExceeded = errors.New("max retry count exceeded")
)

const maxRetries = 5

type Repository interface {
	CreateNotification(ctx context.Context, notification Notification) (uuid.UUID, error)
	CreateDeliveries(ctx context.Context, notificationID uuid.UUID, channels []Channel) error
	ListByRecipient(ctx context.Context, recipientID uuid.UUID, filter FilterNotification) ([]Notification, int, error)
	MarkRead(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error
	MarkDismissed(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error
	// Delivery worker methods
	MarkDeliveryDelivered(ctx context.Context, notificationID uuid.UUID, channel Channel) error
	MarkDeliveryFailed(ctx context.Context, notificationID uuid.UUID, channel Channel, errMsg string) error
	RetryDelivery(ctx context.Context, notificationID uuid.UUID, channel Channel) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateNotification(ctx context.Context, notification Notification) (uuid.UUID, error) {
	const q = `
		INSERT INTO tbl_notification (
			recipient_id, recipient_type, sender_id, sender_type,
			event_type, entity_type, entity_id, status, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'UNREAD', $8)
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, q,
		notification.RecipientID,
		notification.RecipientType,
		notification.SenderID,
		notification.SenderType,
		notification.EventType,
		notification.EntityType,
		notification.EntityID,
		notification.Payload,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert notification: %w", err)
	}
	return id, nil
}

func (r *repository) CreateDeliveries(ctx context.Context, notificationID uuid.UUID, channels []Channel) error {
	for _, ch := range channels {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO tbl_notification_delivery (notification_id, channel) VALUES ($1, $2)`,
			notificationID, ch,
		)
		if err != nil {
			return fmt.Errorf("insert delivery for channel %s: %w", ch, err)
		}
	}
	return nil
}

func (r *repository) ListByRecipient(ctx context.Context, recipientID uuid.UUID, filter FilterNotification) ([]Notification, int, error) {
	args := []any{recipientID}
	where := "WHERE recipient_id = $1 AND status != 'DISMISSED'"

	if filter.Status != nil {
		args = append(args, *filter.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tbl_notification "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	args = append(args, limit, offset)
	q := fmt.Sprintf(`
		SELECT id, recipient_id, sender_id, event_type, entity_type, entity_id,
		       status, payload, created_at, read_at AS readed_at
		FROM tbl_notification
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))

	var rows []Notification
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	return rows, total, nil
}

// MarkRead transitions UNREAD → READ.
func (r *repository) MarkRead(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tbl_notification
		 SET status = 'READ', read_at = NOW()
		 WHERE id = $1 AND recipient_id = $2 AND status = 'UNREAD'`,
		id, recipientID,
	)
	if err != nil {
		return err
	}
	return requireOneRow(res, ErrInvalidTransition)
}

// MarkDismissed transitions READ → DISMISSED.
func (r *repository) MarkDismissed(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tbl_notification
		 SET status = 'DISMISSED'
		 WHERE id = $1 AND recipient_id = $2 AND status = 'READ'`,
		id, recipientID,
	)
	if err != nil {
		return err
	}
	return requireOneRow(res, ErrInvalidTransition)
}

// ── Delivery worker methods ───────────────────────────────────────────────────

func (r *repository) MarkDeliveryDelivered(ctx context.Context, notificationID uuid.UUID, channel Channel) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tbl_notification_delivery
		 SET status = 'DELIVERED', delivered_at = NOW(), last_attempted_at = NOW()
		 WHERE notification_id = $1 AND channel = $2 AND status = 'PENDING'`,
		notificationID, channel,
	)
	if err != nil {
		return err
	}
	return requireOneRow(res, ErrInvalidTransition)
}

func (r *repository) MarkDeliveryFailed(ctx context.Context, notificationID uuid.UUID, channel Channel, errMsg string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tbl_notification_delivery
		 SET status = 'FAILED', retry_count = retry_count + 1,
		     last_attempted_at = NOW(), error_message = $3
		 WHERE notification_id = $1 AND channel = $2 AND status = 'PENDING'`,
		notificationID, channel, errMsg,
	)
	if err != nil {
		return err
	}
	return requireOneRow(res, ErrInvalidTransition)
}

func (r *repository) RetryDelivery(ctx context.Context, notificationID uuid.UUID, channel Channel) error {
	var retryCount int
	err := r.db.QueryRowContext(ctx,
		`SELECT retry_count FROM tbl_notification_delivery
		 WHERE notification_id = $1 AND channel = $2 AND status = 'FAILED'`,
		notificationID, channel,
	).Scan(&retryCount)
	if err != nil {
		return ErrNotFound
	}
	if retryCount >= maxRetries {
		return ErrMaxRetriesExceeded
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE tbl_notification_delivery SET status = 'PENDING'
		 WHERE notification_id = $1 AND channel = $2 AND status = 'FAILED'`,
		notificationID, channel,
	)
	if err != nil {
		return err
	}
	return requireOneRow(res, ErrInvalidTransition)
}

// requireOneRow returns errIfZero when no rows were affected (wrong state / not found).
func requireOneRow(res interface{ RowsAffected() (int64, error) }, errIfZero error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errIfZero
	}
	return nil
}
