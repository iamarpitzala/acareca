package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type notifierAdapter struct {
	db *sqlx.DB
}

func NewNotifier(database *sqlx.DB) Notifier {
	return &notifierAdapter{db: database}
}

func (n *notifierAdapter) SendToPractitioner(ctx context.Context, practitionerID uuid.UUID, actor *uuid.UUID, e Event) error {
	return n.insert(ctx, practitionerID, actor, e)
}

func (n *notifierAdapter) SendToAccountant(ctx context.Context, accountantID uuid.UUID, actor *uuid.UUID, e Event) error {
	return n.insert(ctx, accountantID, actor, e)
}

func (n *notifierAdapter) insert(ctx context.Context, recipientID uuid.UUID, senderID *uuid.UUID, e Event) error {
	if senderID != nil && *senderID == recipientID {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"title":      e.Title,
		"body":       e.Body,
		"extra_data": e.ExtraData,
	})
	if err != nil {
		return fmt.Errorf("notifier: marshal payload: %w", err)
	}

	const q = `
		INSERT INTO tbl_notification (
			recipient_id, sender_id, event_type, entity_type, entity_id, status, payload
		) VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
	`
	_, err = n.db.ExecContext(ctx, q,
		recipientID,
		senderID,
		string(e.Kind),
		entityTypeFromKind(e.Kind),
		e.EntityID,
		payload,
	)
	if err != nil {
		return fmt.Errorf("notifier: insert notification: %w", err)
	}
	return nil
}

func entityTypeFromKind(k EventKind) string {
	switch k {
	case EventInviteSent, EventInviteAccepted, EventInviteDeclined:
		return "invite"
	case EventClinicUpdated:
		return "clinic"
	case EventFormUpdated:
		return "form"
	case EventTransactionCreated, EventTransactionUpdated:
		return "transaction"
	default:
		return "unknown"
	}
}
