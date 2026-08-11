package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

type handlerFakeNotificationRepo struct {
	insertFn  func(ctx context.Context, n domain.Notification) (domain.Notification, error)
	findByIDFn func(ctx context.Context, id uuid.UUID) (domain.Notification, error)
}

func (f *handlerFakeNotificationRepo) Insert(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	if f.insertFn != nil {
		return f.insertFn(ctx, n)
	}
	return n, nil
}

func (f *handlerFakeNotificationRepo) FindByID(ctx context.Context, id uuid.UUID) (domain.Notification, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (f *handlerFakeNotificationRepo) FindByNotificationKey(ctx context.Context, key string) (domain.Notification, error) {
	return domain.Notification{}, fmt.Errorf("not implemented")
}

func (f *handlerFakeNotificationRepo) ClaimBatch(ctx context.Context, batchSize int, leaseDuration time.Duration) ([]domain.Notification, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *handlerFakeNotificationRepo) Complete(ctx context.Context, id uuid.UUID, status domain.Status, leaseToken uuid.UUID, now time.Time, failureReason string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

type handlerFakeClock struct {
	now time.Time
}

func (c *handlerFakeClock) Now() time.Time { return c.now }

type handlerFakeIDGenerator struct {
	next uuid.UUID
}

func (g *handlerFakeIDGenerator) NewID() uuid.UUID { return g.next }

func TestSendNotificationHandler_Success(t *testing.T) {
	fixedID := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000001")
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	deps := app.SendNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"key-1","channel":"EMAIL","recipient":{"email":"user@example.com"},"subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var resp NotificationResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "ntf_018c3f4c-a1b2-7000-8000-000000000001", resp.NotificationID)
	require.Equal(t, "PENDING", resp.Status)
	require.Equal(t, 0, resp.AttemptCount)
}

func TestSendNotificationHandler_ValidationError(t *testing.T) {
	fixedID := uuid.New()
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	deps := app.SendNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"","channel":"EMAIL","recipient":{"email":"user@example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	require.Contains(t, resp["error"], "notification_key")
}

func TestSendNotificationHandler_InvalidJSON(t *testing.T) {
	fixedID := uuid.New()
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	deps := app.SendNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `not json`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSendNotificationHandler_PayloadMismatch(t *testing.T) {
	fixedID := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000001")
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	repo := &handlerFakeNotificationRepo{
		insertFn: func(ctx context.Context, n domain.Notification) (domain.Notification, error) {
			return domain.Notification{
				ID:                 uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000999"),
				NotificationKey:    "key-1",
				PayloadFingerprint: "different-fp",
				Channel:            domain.ChannelEmail,
				Status:             domain.StatusPending,
				CreatedAt:          fixedTime,
				UpdatedAt:          fixedTime,
			}, nil
		},
	}

	deps := app.SendNotificationDeps{
		Notifications: repo,
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"key-1","channel":"EMAIL","recipient":{"email":"user@example.com"},"subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestSendNotificationHandler_IdempotentReplay(t *testing.T) {
	fixedID := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000001")
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	existingID := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000999")

	recipient, _ := domain.NewRecipient([]byte(`{"email":"user@example.com"}`))
	fp := domain.ComputeFingerprint(domain.ChannelEmail, recipient, "Hello", "World", "", "", "")

	repo := &handlerFakeNotificationRepo{
		insertFn: func(ctx context.Context, n domain.Notification) (domain.Notification, error) {
			return domain.Notification{
				ID:                 existingID,
				NotificationKey:    "key-1",
				PayloadFingerprint: fp,
				Channel:            domain.ChannelEmail,
				Recipient:          recipient,
				Status:             domain.StatusPending,
				CreatedAt:          fixedTime,
				UpdatedAt:          fixedTime,
			}, nil
		},
	}

	deps := app.SendNotificationDeps{
		Notifications: repo,
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"key-1","channel":"EMAIL","recipient":{"email":"user@example.com"},"subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
}

func TestSendNotificationHandler_InvalidChannel(t *testing.T) {
	fixedID := uuid.New()
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	deps := app.SendNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"key-1","channel":"SMS","recipient":{"email":"user@example.com"},"subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSendNotificationHandler_InvalidRecipient(t *testing.T) {
	fixedID := uuid.New()
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	deps := app.SendNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
		Clock:         &handlerFakeClock{now: fixedTime},
		IDs:           &handlerFakeIDGenerator{next: fixedID},
	}

	body := `{"notification_key":"key-1","channel":"EMAIL","recipient":{"email":"invalid"},"subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := SendNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetNotificationHandler_Found(t *testing.T) {
	fixedTime := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	repo := &handlerFakeNotificationRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (domain.Notification, error) {
			return domain.Notification{
				ID:              id,
				NotificationKey: "key-1",
				Channel:         domain.ChannelEmail,
				Status:          domain.StatusSent,
				SentAt:          fixedTime,
				CreatedAt:       fixedTime,
			}, nil
		},
	}

	deps := app.GetNotificationDeps{Notifications: repo}

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/ntf_018c3f4c-a1b2-7000-8000-000000000001", nil)
	req.SetPathValue("id", "ntf_018c3f4c-a1b2-7000-8000-000000000001")
	rec := httptest.NewRecorder()

	handler := GetNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp NotificationResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "ntf_018c3f4c-a1b2-7000-8000-000000000001", resp.NotificationID)
	require.Equal(t, "SENT", resp.Status)
}

func TestGetNotificationHandler_NotFound(t *testing.T) {
	deps := app.GetNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/ntf_018c3f4c-a1b2-7000-8000-000000000001", nil)
	req.SetPathValue("id", "ntf_018c3f4c-a1b2-7000-8000-000000000001")
	rec := httptest.NewRecorder()

	handler := GetNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetNotificationHandler_InvalidPrefix(t *testing.T) {
	deps := app.GetNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/dly_018c3f4c-a1b2-7000-8000-000000000001", nil)
	req.SetPathValue("id", "dly_018c3f4c-a1b2-7000-8000-000000000001")
	rec := httptest.NewRecorder()

	handler := GetNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetNotificationHandler_InvalidUUID(t *testing.T) {
	deps := app.GetNotificationDeps{
		Notifications: &handlerFakeNotificationRepo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/ntf_not-a-uuid", nil)
	req.SetPathValue("id", "ntf_not-a-uuid")
	rec := httptest.NewRecorder()

	handler := GetNotificationHandler(deps)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}