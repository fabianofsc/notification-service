package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

func TestComputeFingerprint_SamePayloadProducesSameHash(t *testing.T) {
	r1 := mustRecipient(t, `{"email":"cliente@example.com"}`)
	r2 := mustRecipient(t, `{"email":"cliente@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r1, "Hello", "World", "ref-1", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r2, "Hello", "World", "ref-1", "", "")

	require.Equal(t, fp1, fp2)
	require.Len(t, fp1, 64)
}

func TestComputeFingerprint_DifferentPayloadProducesDifferentHash(t *testing.T) {
	r1 := mustRecipient(t, `{"email":"a@example.com"}`)
	r2 := mustRecipient(t, `{"email":"b@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r1, "Hello", "World", "r", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r2, "Hello", "World", "r", "", "")

	require.NotEqual(t, fp1, fp2)
}

func TestComputeFingerprint_DifferentSubjectProducesDifferentHash(t *testing.T) {
	r := mustRecipient(t, `{"email":"a@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r, "Hello", "World", "r", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r, "Bye", "World", "r", "", "")

	require.NotEqual(t, fp1, fp2)
}

func TestComputeFingerprint_DifferentBodyProducesDifferentHash(t *testing.T) {
	r := mustRecipient(t, `{"email":"a@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r, "X", "World", "r", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r, "X", "Earth", "r", "", "")

	require.NotEqual(t, fp1, fp2)
}

func TestComputeFingerprint_DifferentCallbackProducesDifferentHash(t *testing.T) {
	r := mustRecipient(t, `{"email":"a@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r, "X", "Y", "r", "cb-1", "order_confirmed")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r, "X", "Y", "r", "cb-2", "order_confirmed")

	require.NotEqual(t, fp1, fp2)
}

func TestComputeFingerprint_NormalizesEmail(t *testing.T) {
	r1 := mustRecipient(t, `{"email":"Cliente@Example.COM"}`)
	r2 := mustRecipient(t, `{"email":"cliente@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r1, "H", "W", "r", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r2, "H", "W", "r", "", "")

	require.Equal(t, fp1, fp2)
}

func TestComputeFingerprint_NormalizesWhitespace(t *testing.T) {
	r := mustRecipient(t, `{"email":"a@example.com"}`)

	fp1 := domain.ComputeFingerprint(domain.ChannelEmail, r, "  Hello  ", "  World  ", "  ref  ", "", "")
	fp2 := domain.ComputeFingerprint(domain.ChannelEmail, r, "Hello", "World", "ref", "", "")

	require.Equal(t, fp1, fp2)
}