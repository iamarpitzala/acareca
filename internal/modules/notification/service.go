package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	auditctx "github.com/iamarpitzala/acareca/internal/shared/audit"
)

type Service interface {
	// Invitation notifications
	NotifyInviteSent(
		ctx context.Context,
		actorUserID *uuid.UUID,
		practitionerID uuid.UUID,
		inviteID uuid.UUID,
		recipientEmail string,
		senderName string,
	) error
	NotifyInviteProcessed(
		ctx context.Context,
		actorUserID *uuid.UUID,
		inviteID uuid.UUID,
		practitionerID uuid.UUID,
		action string, // "reject" | other (accept)
	) error

	// Content notifications (account -> practitioner)
	NotifyClinicUpdated(ctx context.Context, actorUserID *uuid.UUID, clinicID uuid.UUID) error
	NotifyFormUpdated(ctx context.Context, actorUserID *uuid.UUID, formID uuid.UUID) error
	NotifyTransactionCreated(ctx context.Context, actorUserID *uuid.UUID, entryID uuid.UUID) error
	NotifyTransactionStatusChanged(
		ctx context.Context,
		actorUserID *uuid.UUID,
		entryID uuid.UUID,
		fromStatus string,
		toStatus string,
	) error

	// User-facing notification list
	ListNotifications(
		ctx context.Context,
		recipientID uuid.UUID,
		status *Status,
		page int,
		limit int,
	) (*ListNotificationsResponse, error)
	MarkRead(ctx context.Context, recipientID, notificationID uuid.UUID) error
	MarkDismissed(ctx context.Context, recipientID, notificationID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) NotifyInviteSent(
	ctx context.Context,
	actorUserID *uuid.UUID,
	practitionerID uuid.UUID,
	inviteID uuid.UUID,
	recipientEmail string,
	senderName string,
) error {
	recipientUserID, err := s.repo.GetUserIDByEmail(ctx, recipientEmail)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		// Recipient hasn't registered yet; you can queue this via outbox later.
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Invitation received",
		Body:       fmt.Sprintf("%s invited you to collaborate.", senderName),
		SenderName: senderName,
		EntityName: "Invitation",
		ExtraData: map[string]any{
			"practitioner_id": practitionerID.String(),
		},
	}

	return s.repo.CreateNotification(
		ctx,
		*recipientUserID,
		actorUserID,
		EventInviteSent,
		EntityInvite,
		inviteID,
		payload,
	)
}

func (s *service) NotifyInviteProcessed(
	ctx context.Context,
	actorUserID *uuid.UUID,
	inviteID uuid.UUID,
	practitionerID uuid.UUID,
	action string,
) error {
	recipientUserID, err := s.repo.GetUserIDByPractitionerID(ctx, practitionerID)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
		// Usually the actor is the invitee, not the practitioner; but keep it safe.
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

	return s.repo.CreateNotification(
		ctx,
		*recipientUserID,
		actorUserID,
		eventType,
		EntityInvite,
		inviteID,
		payload,
	)
}

func (s *service) NotifyClinicUpdated(ctx context.Context, actorUserID *uuid.UUID, clinicID uuid.UUID) error {
	recipientUserID, err := s.repo.GetPractitionerUserIDByClinicID(ctx, clinicID)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Clinic updated",
		Body:       "A clinic was updated.",
		EntityName: "Clinic",
		ExtraData: map[string]any{
			"clinic_id": clinicID.String(),
		},
	}
	return s.repo.CreateNotification(ctx, *recipientUserID, actorUserID, EventClinicUpdated, EntityClinic, clinicID, payload)
}

func (s *service) NotifyFormUpdated(ctx context.Context, actorUserID *uuid.UUID, formID uuid.UUID) error {
	recipientUserID, err := s.repo.GetPractitionerUserIDByFormID(ctx, formID)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Form updated",
		Body:       "A form was updated.",
		EntityName: "Form",
		ExtraData: map[string]any{
			"form_id": formID.String(),
		},
	}
	return s.repo.CreateNotification(ctx, *recipientUserID, actorUserID, EventFormUpdated, EntityForm, formID, payload)
}

func (s *service) NotifyTransactionCreated(ctx context.Context, actorUserID *uuid.UUID, entryID uuid.UUID) error {
	recipientUserID, err := s.repo.GetPractitionerUserIDByEntryID(ctx, entryID)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
		return nil
	}

	payload := NotificationPayload{
		Title:      "Transaction created",
		Body:       "A transaction was created.",
		EntityName: "Transaction",
		ExtraData: map[string]any{
			"transaction_id": entryID.String(),
		},
	}
	return s.repo.CreateNotification(ctx, *recipientUserID, actorUserID, EventTransactionCreated, EntityTransaction, entryID, payload)
}

func (s *service) NotifyTransactionStatusChanged(
	ctx context.Context,
	actorUserID *uuid.UUID,
	entryID uuid.UUID,
	fromStatus string,
	toStatus string,
) error {
	recipientUserID, err := s.repo.GetPractitionerUserIDByEntryID(ctx, entryID)
	if err != nil {
		return err
	}
	if recipientUserID == nil {
		return nil
	}
	if actorUserID != nil && *actorUserID == *recipientUserID {
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
	return s.repo.CreateNotification(ctx, *recipientUserID, actorUserID, EventTransactionUpdated, EntityTransaction, entryID, payload)
}

func (s *service) ListNotifications(
	ctx context.Context,
	recipientID uuid.UUID,
	status *Status,
	page int,
	limit int,
) (*ListNotificationsResponse, error) {
	items, unread, total, err := s.repo.ListByRecipient(ctx, recipientID, status, page, limit)
	if err != nil {
		return nil, err
	}
	return &ListNotificationsResponse{
		Notifications: items,
		UnreadCount:   unread,
		Total:         total,
	}, nil
}

func (s *service) MarkRead(ctx context.Context, recipientID, notificationID uuid.UUID) error {
	return s.repo.MarkRead(ctx, recipientID, notificationID)
}

func (s *service) MarkDismissed(ctx context.Context, recipientID, notificationID uuid.UUID) error {
	return s.repo.MarkDismissed(ctx, recipientID, notificationID)
}

// ActorUserIDFromAudit attempts to parse audit user id into uuid.
func ActorUserIDFromAudit(ctx context.Context) *uuid.UUID {
	meta := auditctx.GetMetadata(ctx)
	if meta == nil || meta.UserID == nil || *meta.UserID == "" {
		return nil
	}
	id, err := uuid.Parse(*meta.UserID)
	if err != nil {
		return nil
	}
	return &id
}

// Timeout helper for async scenarios (future-proofing).
func nowUTC() time.Time { return time.Now().UTC() }
