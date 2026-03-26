package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	sharednotification "github.com/iamarpitzala/acareca/internal/shared/notification"
)

type Service interface {
	Publish(ctx context.Context, rq RqNotification) error
	List(ctx context.Context, recipientID uuid.UUID, filter FilterNotification) (RsListNotification, error)
	MarkRead(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error
	MarkDismissed(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error
}

type service struct {
	repo    Repository
	notifier *sharednotification.Hub
}

func NewService(repo Repository, notifier *sharednotification.Hub) Service {
	return &service{repo: repo, notifier: notifier}
}

func (s *service) Publish(ctx context.Context, rq RqNotification) error {
	if err := s.repo.CreateNotification(ctx, rq.MapToDB()); err != nil {
		return err
	}
	if s.notifier != nil {
		// Build a push-friendly struct so payload renders as JSON object, not base64
		push := map[string]any{
			"id":           rq.ID,
			"recipient_id": rq.RecipientID,
			"sender_id":    rq.SenderID,
			"event_type":   rq.EventType,
			"entity_type":  rq.EntityType,
			"entity_id":    rq.EntityID,
			"status":       rq.Status,
			"payload":      json.RawMessage(rq.Payload),
			"created_at":   rq.CreatedAt,
		}
		s.notifier.Push(rq.RecipientID, push)
	}
	return nil
}

func (s *service) List(ctx context.Context, recipientID uuid.UUID, filter FilterNotification) (RsListNotification, error) {
	notifications, total, err := s.repo.ListByRecipient(ctx, recipientID, filter)
	if err != nil {
		return RsListNotification{}, err
	}

	unread := 0
	for _, n := range notifications {
		if n.Status == StatusUnread {
			unread++
		}
	}

	return RsListNotification{
		Notifications: notifications,
		UnreadCount:   unread,
		Total:         total,
	}, nil
}

func (s *service) MarkRead(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error {
	return s.repo.MarkRead(ctx, id, recipientID)
}

func (s *service) MarkDismissed(ctx context.Context, id uuid.UUID, recipientID uuid.UUID) error {
	return s.repo.MarkDismissed(ctx, id, recipientID)
}
