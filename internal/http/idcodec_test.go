package http

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEncodeID_Roundtrips(t *testing.T) {
	id := uuid.New()

	encoded := encodeID(notificationKind, id)
	require.Equal(t, "ntf_"+id.String(), encoded)

	decoded, err := decodeID(notificationKind, encoded)
	require.NoError(t, err)
	require.Equal(t, id, decoded)
}

func TestEncodeID_DeliveryRoundtrips(t *testing.T) {
	id := uuid.New()

	encoded := encodeID(deliveryKind, id)
	require.Equal(t, "dly_"+id.String(), encoded)

	decoded, err := decodeID(deliveryKind, encoded)
	require.NoError(t, err)
	require.Equal(t, id, decoded)
}

func TestDecodeID_WrongPrefix(t *testing.T) {
	id := uuid.New()
	encoded := encodeID(deliveryKind, id)

	_, err := decodeID(notificationKind, encoded)
	require.ErrorIs(t, err, errInvalidID)
}

func TestDecodeID_InvalidUUID(t *testing.T) {
	_, err := decodeID(notificationKind, "ntf_not-a-uuid")
	require.ErrorIs(t, err, errInvalidID)
}

func TestDecodeID_Empty(t *testing.T) {
	_, err := decodeID(notificationKind, "")
	require.ErrorIs(t, err, errInvalidID)
}

func TestDecodeID_NoPrefix(t *testing.T) {
	_, err := decodeID(notificationKind, uuid.New().String())
	require.ErrorIs(t, err, errInvalidID)
}