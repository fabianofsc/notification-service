package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

type GetNotificationDeps struct {
	Notifications NotificationRepository
}

type GetNotificationInput struct {
	ID uuid.UUID
}

func GetNotification(ctx context.Context, deps GetNotificationDeps, input GetNotificationInput) (domain.Notification, error) {
	n, err := deps.Notifications.FindByID(ctx, input.ID)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	if n.ID == uuid.Nil {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	return n, nil
}