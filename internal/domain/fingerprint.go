package domain

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func ComputeFingerprint(channel Channel, recipient Recipient, subject string, body string, referenceID string, callbackID string, callbackName string) string {
	var normalizedTo string
	switch channel {
	case ChannelEmail:
		email, err := recipient.Email()
		if err != nil {
			return ""
		}
		normalizedTo = email
	case ChannelSMS:
		phone, err := recipient.PhoneNumber()
		if err != nil {
			return ""
		}
		normalizedTo = phone
	}

	fields := []string{
		string(channel),
		normalizedTo,
		strings.TrimSpace(subject),
		strings.TrimSpace(body),
		strings.TrimSpace(referenceID),
		strings.TrimSpace(callbackID),
		strings.TrimSpace(callbackName),
	}
	var canonical strings.Builder
	for _, field := range fields {
		fmt.Fprintf(&canonical, "%d:%s", len(field), field)
	}

	hash := sha256.Sum256([]byte(canonical.String()))
	return fmt.Sprintf("%x", hash)
}
