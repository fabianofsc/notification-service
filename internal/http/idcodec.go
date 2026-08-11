package http

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type idKind string

const (
	notificationKind idKind = "ntf"
	deliveryKind     idKind = "dly"
)

var errInvalidID = errors.New("invalid id format")

func encodeID(kind idKind, id uuid.UUID) string {
	return fmt.Sprintf("%s_%s", kind, id.String())
}

func decodeID(kind idKind, s string) (uuid.UUID, error) {
	prefix := string(kind) + "_"
	if !strings.HasPrefix(s, prefix) {
		return uuid.Nil, fmt.Errorf("%w: expected prefix %s_", errInvalidID, kind)
	}

	id, err := uuid.Parse(s[len(prefix):])
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", errInvalidID, err)
	}

	return id, nil
}