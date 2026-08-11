package domain

import (
	"github.com/google/uuid"
	"time"
)

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSending Status = "SENDING"
	StatusSent    Status = "SENT"
	StatusFailed  Status = "FAILED"
)

func (s Status) IsTerminal() bool {
	return s == StatusSent || s == StatusFailed
}

type Channel string

const (
	ChannelEmail Channel = "EMAIL"
	ChannelSMS   Channel = "SMS"
)

type Notification struct {
	ID                 uuid.UUID
	NotificationKey    string
	PayloadFingerprint string
	Channel            Channel
	Recipient          Recipient
	Subject            string
	Body               string
	ReferenceID        string
	CallbackID         string
	CallbackName       string
	Status             Status
	LeaseToken         uuid.UUID
	LeaseUntil         time.Time
	AttemptCount       int
	FailureReason      string
	CreatedAt          time.Time
	SentAt             time.Time
	UpdatedAt          time.Time
}

func NewNotification(
	id uuid.UUID,
	notificationKey string,
	payloadFingerprint string,
	channel Channel,
	recipient Recipient,
	subject string,
	body string,
	referenceID string,
	callbackID string,
	callbackName string,
	now time.Time,
) (Notification, error) {
	if notificationKey == "" {
		return Notification{}, ErrEmptyNotificationKey
	}
	if subject == "" {
		return Notification{}, ErrEmptySubject
	}
	if body == "" {
		return Notification{}, ErrEmptyBody
	}
	if err := recipient.ValidateFor(channel); err != nil {
		return Notification{}, err
	}

	return Notification{
		ID:                 id,
		NotificationKey:    notificationKey,
		PayloadFingerprint: payloadFingerprint,
		Channel:            channel,
		Recipient:          recipient,
		Subject:            subject,
		Body:               body,
		ReferenceID:        referenceID,
		CallbackID:         callbackID,
		CallbackName:       callbackName,
		Status:             StatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func (n *Notification) Claim(token uuid.UUID, leaseUntil time.Time, now time.Time) error {
	if n.Status.IsTerminal() {
		return ErrAlreadyTerminal
	}
	if n.Status == StatusSending && n.LeaseUntil.After(now) {
		return ErrLeaseActive
	}

	n.Status = StatusSending
	n.LeaseToken = token
	n.LeaseUntil = leaseUntil
	n.AttemptCount++
	n.UpdatedAt = now
	return nil
}

func (n *Notification) CompleteSuccess(token uuid.UUID, now time.Time) error {
	if n.Status != StatusSending {
		return ErrNotSending
	}
	if n.LeaseToken != token {
		return ErrLeaseTokenMismatch
	}

	n.Status = StatusSent
	n.LeaseToken = uuid.Nil
	n.SentAt = now
	n.UpdatedAt = now
	return nil
}

func (n *Notification) CompleteFailure(token uuid.UUID, reason string, now time.Time) error {
	if n.Status != StatusSending {
		return ErrNotSending
	}
	if n.LeaseToken != token {
		return ErrLeaseTokenMismatch
	}

	n.Status = StatusFailed
	n.LeaseToken = uuid.Nil
	n.FailureReason = reason
	n.UpdatedAt = now
	return nil
}