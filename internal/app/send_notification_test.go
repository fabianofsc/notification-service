package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

var testTime = time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

func newDeps() (app.SendNotificationDeps, *fakeNotificationsRepository) {
	fixedID := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000001")
	repo := &fakeNotificationsRepository{}
	return app.SendNotificationDeps{
		Notifications: repo,
		Clock:         &fakeClock{now: testTime},
		IDs:           &fakeIDGenerator{next: fixedID},
	}, repo
}

func validInput() app.SendNotificationInput {
	r, _ := domain.NewRecipient([]byte(`{"email":"user@example.com"}`))
	return app.SendNotificationInput{
		NotificationKey: "key-1",
		Channel:         domain.ChannelEmail,
		Recipient:       r,
		Subject:         "Hello",
		Body:            "World",
		ReferenceID:     "ref-1",
	}
}

func TestSendNotification_Success(t *testing.T) {
	deps, _ := newDeps()
	input := validInput()

	n, err := app.SendNotification(context.Background(), deps, input)
	require.NoError(t, err)
	require.Equal(t, "key-1", n.NotificationKey)
	require.Equal(t, domain.StatusPending, n.Status)
	require.False(t, n.PayloadFingerprint == "")
}

func TestSendNotification_EmptyKey(t *testing.T) {
	deps, _ := newDeps()
	input := validInput()
	input.NotificationKey = ""

	_, err := app.SendNotification(context.Background(), deps, input)
	require.ErrorIs(t, err, domain.ErrEmptyNotificationKey)
}

func TestSendNotification_InvalidRecipient(t *testing.T) {
	deps, _ := newDeps()
	input := validInput()
	r, _ := domain.NewRecipient([]byte(`{"email":"not-an-email"}`))
	input.Recipient = r

	_, err := app.SendNotification(context.Background(), deps, input)
	require.ErrorIs(t, err, domain.ErrInvalidRecipient)
}

func TestSendNotification_EmptySubject(t *testing.T) {
	deps, _ := newDeps()
	input := validInput()
	input.Subject = ""

	_, err := app.SendNotification(context.Background(), deps, input)
	require.ErrorIs(t, err, domain.ErrEmptySubject)
}

func TestSendNotification_EmptyBody(t *testing.T) {
	deps, _ := newDeps()
	input := validInput()
	input.Body = ""

	_, err := app.SendNotification(context.Background(), deps, input)
	require.ErrorIs(t, err, domain.ErrEmptyBody)
}

func TestSendNotification_IdempotentSameFingerprint(t *testing.T) {
	deps, repo := newDeps()
	input := validInput()

	existing := domain.Notification{
		ID:                 uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000999"),
		NotificationKey:    "key-1",
		PayloadFingerprint: domain.ComputeFingerprint(input.Channel, input.Recipient, input.Subject, input.Body, input.ReferenceID),
		Channel:            domain.ChannelEmail,
		Recipient:          input.Recipient,
		Subject:            "Hello",
		Body:               "World",
		ReferenceID:        "ref-1",
		Status:             domain.StatusPending,
	}

	repo.insertFn = func(ctx context.Context, n domain.Notification) (domain.Notification, error) {
		return existing, nil
	}

	n, err := app.SendNotification(context.Background(), deps, input)
	require.NoError(t, err)
	require.Equal(t, existing.ID, n.ID)
}

func TestSendNotification_PayloadMismatch(t *testing.T) {
	deps, repo := newDeps()
	input := validInput()

	existing := domain.Notification{
		ID:                 uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000999"),
		NotificationKey:    "key-1",
		PayloadFingerprint: "completely-different-fingerprint",
		Channel:            domain.ChannelEmail,
		Recipient:          input.Recipient,
		Subject:            "Hello",
		Body:               "World",
		ReferenceID:        "ref-1",
		Status:             domain.StatusPending,
	}

	repo.insertFn = func(ctx context.Context, n domain.Notification) (domain.Notification, error) {
		return existing, nil
	}

	_, err := app.SendNotification(context.Background(), deps, input)
	require.ErrorIs(t, err, domain.ErrPayloadMismatch)
}

func TestSendNotification_RepoError(t *testing.T) {
	deps, repo := newDeps()
	input := validInput()

	repo.insertFn = func(ctx context.Context, n domain.Notification) (domain.Notification, error) {
		return domain.Notification{}, fmt.Errorf("db error")
	}

	_, err := app.SendNotification(context.Background(), deps, input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "db error")
}