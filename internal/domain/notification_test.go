package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

var testNow = time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

func TestNewNotification_CreatesWithPendingStatus(t *testing.T) {
	recipient := validEmailRecipient(t)
	fp := domain.ComputeFingerprint(domain.ChannelEmail, recipient, "Hello", "World", "ref-1")

	n, err := domain.NewNotification(
		uuid.New(),
		"key-1",
		fp,
		domain.ChannelEmail,
		recipient,
		"Hello",
		"World",
		"ref-1",
		testNow,
	)

	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, n.Status)
	require.Equal(t, 0, n.AttemptCount)
	require.Equal(t, testNow, n.CreatedAt)
	require.Equal(t, testNow, n.UpdatedAt)
}

func TestNewNotification_RejectsEmptyKey(t *testing.T) {
	recipient := validEmailRecipient(t)

	_, err := domain.NewNotification(
		uuid.New(), "", "fp", domain.ChannelEmail, recipient, "Hello", "World", "ref-1", testNow,
	)

	require.ErrorIs(t, err, domain.ErrEmptyNotificationKey)
}

func TestNewNotification_RejectsEmptySubject(t *testing.T) {
	recipient := validEmailRecipient(t)

	_, err := domain.NewNotification(
		uuid.New(), "key-1", "fp", domain.ChannelEmail, recipient, "", "World", "ref-1", testNow,
	)

	require.ErrorIs(t, err, domain.ErrEmptySubject)
}

func TestNewNotification_RejectsEmptyBody(t *testing.T) {
	recipient := validEmailRecipient(t)

	_, err := domain.NewNotification(
		uuid.New(), "key-1", "fp", domain.ChannelEmail, recipient, "Hello", "", "ref-1", testNow,
	)

	require.ErrorIs(t, err, domain.ErrEmptyBody)
}

func TestNewNotification_RejectsInvalidRecipient(t *testing.T) {
	recipient := mustRecipient(t, `{"email":"not-an-email"}`)

	_, err := domain.NewNotification(
		uuid.New(), "key-1", "fp", domain.ChannelEmail, recipient, "Hello", "World", "ref-1", testNow,
	)

	require.ErrorIs(t, err, domain.ErrInvalidRecipient)
}

func TestClaim_TransitionsFromPendingToSending(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()
	leaseUntil := testNow.Add(30 * time.Second)

	err := n.Claim(token, leaseUntil, testNow)

	require.NoError(t, err)
	require.Equal(t, domain.StatusSending, n.Status)
	require.Equal(t, token, n.LeaseToken)
	require.Equal(t, leaseUntil, n.LeaseUntil)
	require.Equal(t, 1, n.AttemptCount)
	require.Equal(t, testNow, n.UpdatedAt)
}

func TestClaim_ReclaimsIfLeaseExpired(t *testing.T) {
	n := validPendingNotification(t)
	firstToken := uuid.New()
	firstLease := testNow.Add(30 * time.Second)

	err := n.Claim(firstToken, firstLease, testNow)
	require.NoError(t, err)

	future := testNow.Add(60 * time.Second)
	secondToken := uuid.New()
	secondLease := future.Add(30 * time.Second)

	err = n.Claim(secondToken, secondLease, future)
	require.NoError(t, err)
	require.Equal(t, secondToken, n.LeaseToken)
	require.Equal(t, secondLease, n.LeaseUntil)
	require.Equal(t, 2, n.AttemptCount)
}

func TestClaim_RejectsIfLeaseStillActive(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()
	leaseUntil := testNow.Add(30 * time.Second)

	err := n.Claim(token, leaseUntil, testNow)
	require.NoError(t, err)

	secondToken := uuid.New()
	secondLease := testNow.Add(45 * time.Second)
	err = n.Claim(secondToken, secondLease, testNow.Add(5*time.Second))

	require.ErrorIs(t, err, domain.ErrLeaseActive)
	require.Equal(t, token, n.LeaseToken)
}

func TestClaim_RejectsTerminalSent(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()
	leaseUntil := testNow.Add(30 * time.Second)

	err := n.Claim(token, leaseUntil, testNow)
	require.NoError(t, err)

	err = n.CompleteSuccess(token, testNow)
	require.NoError(t, err)

	newToken := uuid.New()
	err = n.Claim(newToken, testNow.Add(30*time.Second), testNow)
	require.ErrorIs(t, err, domain.ErrAlreadyTerminal)
}

func TestClaim_RejectsTerminalFailed(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()
	leaseUntil := testNow.Add(30 * time.Second)

	err := n.Claim(token, leaseUntil, testNow)
	require.NoError(t, err)

	err = n.CompleteFailure(token, "send failed", testNow)
	require.NoError(t, err)

	newToken := uuid.New()
	err = n.Claim(newToken, testNow.Add(30*time.Second), testNow)
	require.ErrorIs(t, err, domain.ErrAlreadyTerminal)
}

func TestCompleteSuccess_SetsSentStatus(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()

	err := n.Claim(token, testNow.Add(30*time.Second), testNow)
	require.NoError(t, err)

	err = n.CompleteSuccess(token, testNow)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSent, n.Status)
	require.Equal(t, testNow, n.SentAt)
	require.Equal(t, uuid.Nil, n.LeaseToken)
}

func TestCompleteSuccess_RejectsWrongToken(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()

	err := n.Claim(token, testNow.Add(30*time.Second), testNow)
	require.NoError(t, err)

	wrongToken := uuid.New()
	err = n.CompleteSuccess(wrongToken, testNow)
	require.ErrorIs(t, err, domain.ErrLeaseTokenMismatch)
	require.Equal(t, domain.StatusSending, n.Status)
}

func TestCompleteSuccess_RejectsNotSending(t *testing.T) {
	n := validPendingNotification(t)

	err := n.CompleteSuccess(uuid.New(), testNow)
	require.ErrorIs(t, err, domain.ErrNotSending)
}

func TestCompleteFailure_SetsFailedStatus(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()

	err := n.Claim(token, testNow.Add(30*time.Second), testNow)
	require.NoError(t, err)

	err = n.CompleteFailure(token, "provider offline", testNow)
	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, n.Status)
	require.Equal(t, "provider offline", n.FailureReason)
	require.Equal(t, uuid.Nil, n.LeaseToken)
}

func TestCompleteFailure_RejectsWrongToken(t *testing.T) {
	n := validPendingNotification(t)
	token := uuid.New()

	err := n.Claim(token, testNow.Add(30*time.Second), testNow)
	require.NoError(t, err)

	wrongToken := uuid.New()
	err = n.CompleteFailure(wrongToken, "error", testNow)
	require.ErrorIs(t, err, domain.ErrLeaseTokenMismatch)
	require.Equal(t, domain.StatusSending, n.Status)
}

func TestStatus_IsTerminal(t *testing.T) {
	require.False(t, domain.StatusPending.IsTerminal())
	require.False(t, domain.StatusSending.IsTerminal())
	require.True(t, domain.StatusSent.IsTerminal())
	require.True(t, domain.StatusFailed.IsTerminal())
}

func validEmailRecipient(t *testing.T) domain.Recipient {
	t.Helper()
	return mustRecipient(t, `{"email":"cliente@example.com"}`)
}

func mustRecipient(t *testing.T, raw string) domain.Recipient {
	t.Helper()
	r, err := domain.NewRecipient([]byte(raw))
	require.NoError(t, err)
	return r
}

func validPendingNotification(t *testing.T) domain.Notification {
	t.Helper()
	recipient := validEmailRecipient(t)
	fp := domain.ComputeFingerprint(domain.ChannelEmail, recipient, "Hello", "World", "ref-1")
	n, err := domain.NewNotification(
		uuid.New(), "key-"+uuid.New().String(), fp,
		domain.ChannelEmail, recipient, "Hello", "World", "ref-1", testNow,
	)
	require.NoError(t, err)
	return n
}