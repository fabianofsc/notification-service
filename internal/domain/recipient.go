package domain

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
)

type Recipient struct {
	raw json.RawMessage
}

func NewRecipient(raw json.RawMessage) (Recipient, error) {
	if len(raw) == 0 {
		return Recipient{}, ErrEmptyRecipient
	}
	return Recipient{raw: raw}, nil
}

func (r Recipient) Email() (string, error) {
	var v struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return "", fmt.Errorf("%w: invalid JSON", ErrInvalidRecipient)
	}
	if v.Email == "" {
		return "", fmt.Errorf("%w: email is required", ErrInvalidRecipient)
	}
	addr, err := mail.ParseAddress(v.Email)
	if err != nil {
		return "", fmt.Errorf("%w: invalid email format", ErrInvalidRecipient)
	}
	return strings.ToLower(strings.TrimSpace(addr.Address)), nil
}

func (r Recipient) PhoneNumber() (string, error) {
	var v struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.Unmarshal(r.raw, &v); err != nil {
		return "", fmt.Errorf("%w: invalid JSON", ErrInvalidRecipient)
	}
	if v.PhoneNumber == "" {
		return "", fmt.Errorf("%w: phone_number is required", ErrInvalidRecipient)
	}
	normalized := strings.TrimSpace(v.PhoneNumber)
	if !strings.HasPrefix(normalized, "+") {
		return "", fmt.Errorf("%w: phone_number must be in E.164 format", ErrInvalidRecipient)
	}
	return normalized, nil
}

func (r Recipient) ValidateFor(ch Channel) error {
	switch ch {
	case ChannelEmail:
		_, err := r.Email()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidRecipient, err)
		}
		return nil
	case ChannelSMS:
		_, err := r.PhoneNumber()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidRecipient, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported channel %q", ErrInvalidChannel, ch)
	}
}

func (r Recipient) NormalizedSearch(ch Channel) (string, error) {
	switch ch {
	case ChannelEmail:
		email, err := r.Email()
		if err != nil {
			return "", err
		}
		return email, nil
	case ChannelSMS:
		phone, err := r.PhoneNumber()
		if err != nil {
			return "", err
		}
		return phone, nil
	default:
		return "", fmt.Errorf("%w: unsupported channel %q", ErrInvalidChannel, ch)
	}
}

func (r Recipient) Raw() json.RawMessage {
	return r.raw
}