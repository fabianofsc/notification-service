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
	CallbackID      string
	CallbackName    string
}

type SendNotificationResult struct {
	Notification domain.Notification
	Replayed     bool
}

func SendNotification(ctx context.Context, deps SendNotificationDeps, input SendNotificationInput) (SendNotificationResult, error) {
	now := deps.Clock.Now()
	id := deps.IDs.NewID()

	fingerprint := domain.ComputeFingerprint(input.Channel, input.Recipient, input.Subject, input.Body, input.ReferenceID, input.CallbackID, input.CallbackName)

	n, err := domain.NewNotification(
		id,
		input.NotificationKey,
		fingerprint,
		input.Channel,
		input.Recipient,
		input.Subject,
		input.Body,
		input.ReferenceID,
		input.CallbackID,
		input.CallbackName,
		now,
	)
	if err != nil {
		return SendNotificationResult{}, fmt.Errorf("send notification: %w", err)
	}

	inserted, err := deps.Notifications.Insert(ctx, n)
	if err != nil {
		return SendNotificationResult{}, fmt.Errorf("send notification: %w", err)
	}

	if inserted.Replayed && inserted.Notification.PayloadFingerprint != fingerprint {
		return SendNotificationResult{}, domain.ErrPayloadMismatch
	}

	return SendNotificationResult{Notification: inserted.Notification, Replayed: inserted.Replayed}, nil
}
