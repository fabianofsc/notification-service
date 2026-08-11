package domain

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func ComputeFingerprint(channel Channel, recipient Recipient, subject string, body string, referenceID string, callbackID string, callbackName string) string {
	var normalizedEmail string
	if channel == ChannelEmail {
		email, err := recipient.Email()
		if err != nil {
			return ""
		}
		normalizedEmail = email
	}

	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		string(channel),
		normalizedEmail,
		strings.TrimSpace(subject),
		strings.TrimSpace(body),
		strings.TrimSpace(referenceID),
		strings.TrimSpace(callbackID),
		strings.TrimSpace(callbackName),
	)

	hash := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", hash)
}