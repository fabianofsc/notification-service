package http

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

func TestCreateNotificationRequest_Valid(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "EMAIL",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "Hello",
		Body:            "World",
		ReferenceID:     "ref-1",
	}
	err := r.Validate()
	require.NoError(t, err)
}

func TestCreateNotificationRequest_NoReferenceID(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "EMAIL",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "Hello",
		Body:            "World",
	}
	err := r.Validate()
	require.NoError(t, err)
}

func TestCreateNotificationRequest_EmptyKey(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "",
		Channel:         "EMAIL",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "Hello",
		Body:            "World",
	}
	err := r.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "notification_key")
}

func TestCreateNotificationRequest_InvalidChannel(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "SMS",
		Recipient:       json.RawMessage(`{"phone_number":"+5511999999999"}`),
		Subject:         "Hello",
		Body:            "World",
	}
	err := r.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel")
}

func TestCreateNotificationRequest_EmptyChannel(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "Hello",
		Body:            "World",
	}
	err := r.Validate()
	require.Error(t, err)
}

func TestCreateNotificationRequest_EmptyRecipient(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "EMAIL",
		Recipient:       nil,
		Subject:         "Hello",
		Body:            "World",
	}
	err := r.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "recipient")
}

func TestCreateNotificationRequest_EmptySubject(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "EMAIL",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "",
		Body:            "World",
	}
	err := r.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "subject")
}

func TestCreateNotificationRequest_EmptyBody(t *testing.T) {
	r := CreateNotificationRequest{
		NotificationKey: "key-1",
		Channel:         "EMAIL",
		Recipient:       json.RawMessage(`{"email":"user@example.com"}`),
		Subject:         "Hello",
		Body:            "",
	}
	err := r.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "body")
}

func TestNotificationResponse_Pending(t *testing.T) {
	id := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000001")
	n := domain.Notification{
		ID:              id,
		NotificationKey: "key-1",
		Channel:         domain.ChannelEmail,
		Status:          domain.StatusPending,
		ReferenceID:     "ref-1",
		CreatedAt:       time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
	}

	resp := NewNotificationResponse(n)
	require.Equal(t, "ntf_018c3f4c-a1b2-7000-8000-000000000001", resp.NotificationID)
	require.Equal(t, "key-1", resp.NotificationKey)
	require.Equal(t, "EMAIL", resp.Channel)
	require.Equal(t, "PENDING", resp.Status)
	require.Equal(t, 0, resp.AttemptCount)
	require.Nil(t, resp.SentAt)
}

func TestNotificationResponse_Sent(t *testing.T) {
	id := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000002")
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	n := domain.Notification{
		ID:         id,
		Channel:    domain.ChannelEmail,
		Status:     domain.StatusSent,
		SentAt:     now,
		CreatedAt:  now,
	}

	resp := NewNotificationResponse(n)
	require.NotNil(t, resp.SentAt)
	require.Equal(t, now.UTC(), *resp.SentAt)
}

func TestNotificationResponse_Failed(t *testing.T) {
	id := uuid.MustParse("018c3f4c-a1b2-7000-8000-000000000003")
	n := domain.Notification{
		ID:             id,
		Channel:        domain.ChannelEmail,
		Status:         domain.StatusFailed,
		FailureReason:  "temporary failure",
		AttemptCount:   3,
		CreatedAt:      time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
	}

	resp := NewNotificationResponse(n)
	require.Equal(t, "FAILED", resp.Status)
	require.Equal(t, "temporary failure", resp.FailureReason)
	require.Equal(t, 3, resp.AttemptCount)
}

func TestCreateNotificationRequest_JSONMarshal(t *testing.T) {
	body := `{"notification_key":"key-1","channel":"EMAIL","recipient":{"email":"user@example.com"},"subject":"Hello","body":"World","reference_id":"ref-1"}`
	var req CreateNotificationRequest
	err := json.Unmarshal([]byte(body), &req)
	require.NoError(t, err)

	encoded, err := json.Marshal(req)
	require.NoError(t, err)

	var req2 CreateNotificationRequest
	err = json.Unmarshal(encoded, &req2)
	require.NoError(t, err)
	require.Equal(t, "key-1", req2.NotificationKey)
}