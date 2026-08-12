package app_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

type fakeNotificationsRepository struct {
	notifications []domain.Notification
	insertFn      func(ctx context.Context, n domain.Notification) (domain.Notification, error)
	findByIDFn    func(ctx context.Context, id uuid.UUID) (domain.Notification, error)
	findByKeyFn   func(ctx context.Context, key string) (domain.Notification, error)
}

func (f *fakeNotificationsRepository) Insert(ctx context.Context, n domain.Notification) (app.InsertNotificationResult, error) {
	if f.insertFn != nil {
		inserted, err := f.insertFn(ctx, n)
		return app.InsertNotificationResult{Notification: inserted, Replayed: inserted.ID != n.ID}, err
	}
	f.notifications = append(f.notifications, n)
	return app.InsertNotificationResult{Notification: n}, nil
}

func (f *fakeNotificationsRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Notification, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	for _, n := range f.notifications {
		if n.ID == id {
			return n, nil
		}
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (f *fakeNotificationsRepository) FindByNotificationKey(ctx context.Context, key string) (domain.Notification, error) {
	if f.findByKeyFn != nil {
		return f.findByKeyFn(ctx, key)
	}
	for _, n := range f.notifications {
		if n.NotificationKey == key {
			return n, nil
		}
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (f *fakeNotificationsRepository) ClaimBatch(ctx context.Context, batchSize int, leaseDuration time.Duration, now time.Time) ([]domain.Notification, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeNotificationsRepository) Complete(ctx context.Context, id uuid.UUID, status domain.Status, leaseToken uuid.UUID, now time.Time, failureReason string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

type fakeIDGenerator struct {
	next uuid.UUID
}

func (g *fakeIDGenerator) NewID() uuid.UUID { return g.next }
