package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

func TestGetNotification_Found(t *testing.T) {
	id := uuid.New()
	repo := &fakeNotificationsRepository{
		notifications: []domain.Notification{
			{
				ID:              id,
				NotificationKey: "key-1",
				Status:          domain.StatusSent,
			},
		},
	}

	deps := app.GetNotificationDeps{Notifications: repo}
	n, err := app.GetNotification(context.Background(), deps, app.GetNotificationInput{ID: id})
	require.NoError(t, err)
	require.Equal(t, id, n.ID)
	require.Equal(t, "key-1", n.NotificationKey)
}

func TestGetNotification_NotFound(t *testing.T) {
	repo := &fakeNotificationsRepository{}
	deps := app.GetNotificationDeps{Notifications: repo}

	_, err := app.GetNotification(context.Background(), deps, app.GetNotificationInput{ID: uuid.New()})
	require.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestGetNotification_RepoError(t *testing.T) {
	repo := &fakeNotificationsRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (domain.Notification, error) {
			return domain.Notification{}, fmt.Errorf("db down")
		},
	}
	deps := app.GetNotificationDeps{Notifications: repo}

	_, err := app.GetNotification(context.Background(), deps, app.GetNotificationInput{ID: uuid.New()})
	require.Error(t, err)
	require.Contains(t, err.Error(), "db down")
}