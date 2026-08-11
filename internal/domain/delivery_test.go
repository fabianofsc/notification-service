package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

func TestNewDelivery_CreatesWithPendingStatus(t *testing.T) {
	notifID := uuid.New()
	deliveryID := uuid.New()

	d := domain.NewDelivery(deliveryID, notifID, "dkey-1", 1, testNow)

	require.Equal(t, deliveryID, d.ID)
	require.Equal(t, notifID, d.NotificationID)
	require.Equal(t, "dkey-1", d.DeliveryKey)
	require.Equal(t, 1, d.AttemptNumber)
	require.Equal(t, domain.DeliveryStatusPending, d.Status)
	require.Equal(t, testNow, d.CreatedAt)
}

func TestDelivery_CompleteSuccess_SetsSent(t *testing.T) {
	d := domain.NewDelivery(uuid.New(), uuid.New(), "dk", 1, testNow)

	d.CompleteSuccess("OK", testNow)

	require.Equal(t, domain.DeliveryStatusSent, d.Status)
	require.Equal(t, "OK", d.ProviderResponse)
	require.Equal(t, testNow, d.CompletedAt)
}

func TestDelivery_CompleteFailure_SetsFailed(t *testing.T) {
	d := domain.NewDelivery(uuid.New(), uuid.New(), "dk", 1, testNow)

	d.CompleteFailure("network error", testNow)

	require.Equal(t, domain.DeliveryStatusFailed, d.Status)
	require.Equal(t, "network error", d.FailureReason)
	require.Equal(t, testNow, d.CompletedAt)
}