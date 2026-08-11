package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() uuid.UUID
}

type NotificationRepository interface {
	Insert(ctx context.Context, n domain.Notification) (domain.Notification, error)
	ClaimBatch(ctx context.Context, batchSize int, leaseDuration time.Duration) ([]domain.Notification, error)
	Complete(ctx context.Context, id uuid.UUID, status domain.Status, leaseToken uuid.UUID, now time.Time, failureReason string) (bool, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.Notification, error)
	FindByNotificationKey(ctx context.Context, key string) (domain.Notification, error)
}

type DeliveryRepository interface {
	InsertDelivery(ctx context.Context, d domain.Delivery) error
	CompleteDeliverySuccess(ctx context.Context, id uuid.UUID, response string, now time.Time) error
	CompleteDeliveryFailure(ctx context.Context, id uuid.UUID, reason string, now time.Time) error
}

type EmailProvider interface {
	Send(ctx context.Context, to string, subject string, body string, deliveryKey string) (bool, error)
}