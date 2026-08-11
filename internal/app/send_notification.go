package app

import (
	"context"
	"fmt"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

type SendNotificationDeps struct {
	Notifications NotificationRepository
	Clock         Clock
	IDs           IDGenerator
}

type SendNotificationInput struct {
	NotificationKey string
	Channel         domain.Channel
	Recipient       domain.Recipient
	Subject         string
	Body            string
	ReferenceID     string
}

func SendNotification(ctx context.Context, deps SendNotificationDeps, input SendNotificationInput) (domain.Notification, error) {
	now := deps.Clock.Now()
	id := deps.IDs.NewID()

	fingerprint := domain.ComputeFingerprint(input.Channel, input.Recipient, input.Subject, input.Body, input.ReferenceID)

	n, err := domain.NewNotification(
		id,
		input.NotificationKey,
		fingerprint,
		input.Channel,
		input.Recipient,
		input.Subject,
		input.Body,
		input.ReferenceID,
		now,
	)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("send notification: %w", err)
	}

	inserted, err := deps.Notifications.Insert(ctx, n)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("send notification: %w", err)
	}

	if inserted.ID != id {
		if inserted.PayloadFingerprint != fingerprint {
			return domain.Notification{}, domain.ErrPayloadMismatch
		}
		return inserted, nil
	}

	return inserted, nil
}