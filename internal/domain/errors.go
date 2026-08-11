package domain

import "errors"

var (
	ErrEmptyNotificationKey = errors.New("notification_key must not be empty")
	ErrEmptySubject         = errors.New("subject must not be empty")
	ErrEmptyBody            = errors.New("body must not be empty")
	ErrInvalidChannel       = errors.New("invalid channel")
	ErrInvalidRecipient     = errors.New("invalid recipient")
	ErrEmptyRecipient       = errors.New("recipient must not be empty")
	ErrAlreadyTerminal      = errors.New("notification is already in a terminal state")
	ErrLeaseActive          = errors.New("active lease prevents claiming")
	ErrLeaseTokenMismatch   = errors.New("lease token does not match")
	ErrNotSending           = errors.New("notification is not in SENDING state")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrPayloadMismatch      = errors.New("payload does not match existing notification")
)