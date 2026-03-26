package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	auditctx "github.com/iamarpitzala/acareca/internal/shared/audit"
)

type Service interface {
	// Invitation notifications
	NotifyInviteSent(
		ctx context.Context,
		actorEntityID *uuid.UUID, // practitioner entity ID
		inviteID uuid.UUID,
		recipientEmail string,
		senderName string,
	) error
	NotifyInviteProcessed(
		ctx context.Context,
		actorEntityID *uuid.UUID, // accountant entity ID (the invitee acting)
		inviteID uuid.UUID,
		practitionerID uuid.UUID, // recipient
		action string, // "reject" | other (accept)
	) error

	// Content notifications (account -> practitioner)
	NotifyClinicUpdated(ctx context.Context, actorEntityID *uuid.UUID, clinicID uuid.UUID) error
	NotifyFormUpdated(ctx context.Context, actorEntityID *uuid.UUID, formID uuid.UUID) error
	NotifyTransactionCreated(ctx context.Context, actorEntityID *uuid.UUID, entryID uuid.UUID) error
	NotifyTransactionStatusChanged(ctx context.Context, actorEntityID *uuid.UUID, entryID uuid.UUID, fromStatus string, toStatus string) error

	// User-facing notification list
	ListNotifications(ctx context.Context, recipientEntityID uuid.UUID, filter FilterNotification) (*ListNotificationsResponse, error)
	MarkRead(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error
	MarkDismissed(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) NotifyInviteSent(
	ctx context.Context,
	actorEntityID *uuid.UUID,
	inviteID uuid.UUID,
	recipientEmail string,
	senderName string,
) error {
	// Resolve the accountant entity ID from email (recipient may not exist yet)
	recipientEntityID, err := s.repo.GetAccountantEntityIDByEmail(ctx, recipientEmail)
	if err != nil {
		return err
	}
	if recipientEntityID == nil {
		// Recipient hasn't registered yet; skip for now (outbox pattern can handle later)
		return nil
	}
	// Don't notify yourself
	if actorEntityID != nil && *actorEntityID == *recipientEntityID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Invitation received",
		Body:       fmt.Sprintf("%s invited you to collaborate.", senderName),
		SenderName: senderName,
		EntityName: "Invitation",
		ExtraData: map[string]any{
			"invite_id": inviteID.String(),
		},
	}

	return s.repo.CreateNotification(ctx, *recipientEntityID, actorEntityID, EventInviteSent, EntityInvite, inviteID, payload)
}

func (s *service) NotifyInviteProcessed(
	ctx context.Context,
	actorEntityID *uuid.UUID,
	inviteID uuid.UUID,
	practitionerID uuid.UUID,
	action string,
) error {
	// Recipient is always the practitioner
	if actorEntityID != nil && *actorEntityID == practitionerID {
		return nil
	}

	var eventType EventType
	var title, body string
	switch action {
	case "reject":
		eventType = EventInviteDeclined
		title = "Invitation declined"
		body = "The invitee declined the invitation."
	default:
		eventType = EventInviteAccepted
		title = "Invitation accepted"
		body = "The invitee accepted the invitation."
	}

	payload := NotificationPayload{
		Title:      title,
		Body:       body,
		EntityName: "Invitation",
		ExtraData: map[string]any{
			"invite_id": inviteID.String(),
		},
	}

	return s.repo.CreateNotification(ctx, practitionerID, actorEntityID, eventType, EntityInvite, inviteID, payload)
}

func (s *service) NotifyClinicUpdated(ctx context.Context, actorEntityID *uuid.UUID, clinicID uuid.UUID) error {
	recipientID, err := s.repo.GetPractitionerIDByClinicID(ctx, clinicID)
	if err != nil {
		return err
	}
	if recipientID == nil {
		return nil
	}
	if actorEntityID != nil && *actorEntityID == *recipientID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Clinic updated",
		Body:       "A clinic was updated.",
		EntityName: "Clinic",
		ExtraData:  map[string]any{"clinic_id": clinicID.String()},
	}
	return s.repo.CreateNotification(ctx, *recipientID, actorEntityID, EventClinicUpdated, EntityClinic, clinicID, payload)
}

func (s *service) NotifyFormUpdated(ctx context.Context, actorEntityID *uuid.UUID, formID uuid.UUID) error {
	recipientID, err := s.repo.GetPractitionerIDByFormID(ctx, formID)
	if err != nil {
		return err
	}
	if recipientID == nil {
		return nil
	}
	if actorEntityID != nil && *actorEntityID == *recipientID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Form updated",
		Body:       "A form was updated.",
		EntityName: "Form",
		ExtraData:  map[string]any{"form_id": formID.String()},
	}
	return s.repo.CreateNotification(ctx, *recipientID, actorEntityID, EventFormUpdated, EntityForm, formID, payload)
}

func (s *service) NotifyTransactionCreated(ctx context.Context, actorEntityID *uuid.UUID, entryID uuid.UUID) error {
	recipientID, err := s.repo.GetPractitionerIDByEntryID(ctx, entryID)
	if err != nil {
		return err
	}
	if recipientID == nil {
		return nil
	}
	if actorEntityID != nil && *actorEntityID == *recipientID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Transaction created",
		Body:       "A transaction was created.",
		EntityName: "Transaction",
		ExtraData:  map[string]any{"transaction_id": entryID.String()},
	}
	return s.repo.CreateNotification(ctx, *recipientID, actorEntityID, EventTransactionCreated, EntityTransaction, entryID, payload)
}

func (s *service) NotifyTransactionStatusChanged(
	ctx context.Context,
	actorEntityID *uuid.UUID,
	entryID uuid.UUID,
	fromStatus string,
	toStatus string,
) error {
	recipientID, err := s.repo.GetPractitionerIDByEntryID(ctx, entryID)
	if err != nil {
		return err
	}
	if recipientID == nil {
		return nil
	}
	if actorEntityID != nil && *actorEntityID == *recipientID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Transaction status changed",
		Body:       fmt.Sprintf("Status changed from %s to %s.", fromStatus, toStatus),
		EntityName: "Transaction",
		ExtraData: map[string]any{
			"transaction_id": entryID.String(),
			"from_status":    fromStatus,
			"to_status":      toStatus,
		},
	}
	return s.repo.CreateNotification(ctx, *recipientID, actorEntityID, EventTransactionUpdated, EntityTransaction, entryID, payload)
}

func (s *service) ListNotifications(ctx context.Context, recipientEntityID uuid.UUID, filter FilterNotification) (*ListNotificationsResponse, error) {
	rs, err := s.repo.ListByRecipient(ctx, recipientEntityID, filter)
	if err != nil {
		return nil, err
	}

	items, _ := rs.Items.([]Notification)
	return &ListNotificationsResponse{
		Notifications: items,
		Total:         rs.Total,
	}, nil
}

func (s *service) MarkRead(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error {
	return s.repo.MarkRead(ctx, recipientEntityID, notificationID)
}

func (s *service) MarkDismissed(ctx context.Context, recipientEntityID, notificationID uuid.UUID) error {
	return s.repo.MarkDismissed(ctx, recipientEntityID, notificationID)
}

// ActorEntityIDFromAudit extracts the entity ID (practitioner/accountant) from audit context.
func ActorEntityIDFromAudit(ctx context.Context) *uuid.UUID {
	meta := auditctx.GetMetadata(ctx)
	if meta == nil || meta.PracticeID == nil || *meta.PracticeID == "" {
		return nil
	}
	id, err := uuid.Parse(*meta.PracticeID)
	if err != nil {
		return nil
	}
	return &id
}
