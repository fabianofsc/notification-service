package domain

import (
	"github.com/google/uuid"
	"time"
)

type DeliveryStatus string

const (
	DeliveryStatusPending DeliveryStatus = "PENDING"
	DeliveryStatusSent    DeliveryStatus = "SENT"
	DeliveryStatusFailed  DeliveryStatus = "FAILED"
)

type Delivery struct {
	ID               uuid.UUID
	NotificationID   uuid.UUID
	DeliveryKey      string
	Status           DeliveryStatus
	AttemptNumber    int
	ProviderResponse string
	FailureReason    string
	CreatedAt        time.Time
	CompletedAt      time.Time
}

func NewDelivery(
	id uuid.UUID,
	notificationID uuid.UUID,
	deliveryKey string,
	attemptNumber int,
	now time.Time,
) Delivery {
	return Delivery{
		ID:             id,
		NotificationID: notificationID,
		DeliveryKey:    deliveryKey,
		Status:         DeliveryStatusPending,
		AttemptNumber:  attemptNumber,
		CreatedAt:      now,
	}
}

func (d *Delivery) CompleteSuccess(response string, now time.Time) {
	d.Status = DeliveryStatusSent
	d.ProviderResponse = response
	d.CompletedAt = now
}

func (d *Delivery) CompleteFailure(reason string, now time.Time) {
	d.Status = DeliveryStatusFailed
	d.FailureReason = reason
	d.CompletedAt = now
}
