package notification

import (
	"context"

	"github.com/google/uuid"
)

type EventKind string

const (
	EventInviteSent     EventKind = "invite.sent"
	EventInviteAccepted EventKind = "invite.accepted"
	EventInviteDeclined EventKind = "invite.declined"

	EventClinicUpdated      EventKind = "clinic.updated"
	EventFormUpdated        EventKind = "form.updated"
	EventTransactionCreated EventKind = "transaction.created"
	EventTransactionUpdated EventKind = "transaction.status_changed"
)

type Event struct {
	Kind      EventKind
	Title     string
	Body      string
	EntityID  uuid.UUID
	ExtraData map[string]any
}

type Notifier interface {
	SendToPractitioner(ctx context.Context, practitionerID uuid.UUID, actor *uuid.UUID, e Event) error
	SendToAccountant(ctx context.Context, accountantID uuid.UUID, actor *uuid.UUID, e Event) error
}
