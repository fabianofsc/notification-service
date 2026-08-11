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

	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		string(channel),
		normalizedTo,
		strings.TrimSpace(subject),
		strings.TrimSpace(body),
		strings.TrimSpace(referenceID),
		strings.TrimSpace(callbackID),
		strings.TrimSpace(callbackName),
	)

	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash)
}