package http

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

type CreateNotificationRequest struct {
	NotificationKey string          `json:"notification_key"`
	Channel         string          `json:"channel"`
	Recipient       json.RawMessage `json:"recipient"`
	Subject         string          `json:"subject"`
	Body            string          `json:"body"`
	ReferenceID     string          `json:"reference_id"`
}

func (r CreateNotificationRequest) Validate() error {
	if r.NotificationKey == "" {
		return fmt.Errorf("notification_key is required")
	}
	if r.Channel != string(domain.ChannelEmail) {
		return fmt.Errorf("channel must be EMAIL")
	}
	if len(r.Recipient) == 0 {
		return fmt.Errorf("recipient is required")
	}
	if r.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if r.Body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

type NotificationResponse struct {
	NotificationID  string     `json:"notification_id"`
	NotificationKey string     `json:"notification_key"`
	Channel         string     `json:"channel"`
	Status          string     `json:"status"`
	ReferenceID     string     `json:"reference_id"`
	CreatedAt       time.Time  `json:"created_at"`
	AttemptCount    int        `json:"attempt_count"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
	FailureReason   string     `json:"failure_reason,omitempty"`
}

func NewNotificationResponse(n domain.Notification) NotificationResponse {
	resp := NotificationResponse{
		NotificationID:  encodeID(notificationKind, n.ID),
		NotificationKey: n.NotificationKey,
		Channel:         string(n.Channel),
		Status:          string(n.Status),
		ReferenceID:     n.ReferenceID,
		CreatedAt:       n.CreatedAt.UTC(),
		AttemptCount:    n.AttemptCount,
		FailureReason:   n.FailureReason,
	}
	if !n.SentAt.IsZero() {
		sentAt := n.SentAt.UTC()
		resp.SentAt = &sentAt
	}
	return resp
}