package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/domain"
)

func TestRecipient_ValidateForEmail_AcceptsValidEmail(t *testing.T) {
	r := mustRecipient(t, `{"email":"cliente@example.com"}`)
	err := r.ValidateFor(domain.ChannelEmail)
	require.NoError(t, err)
}

func TestRecipient_Email_NormalizesLowercase(t *testing.T) {
	r := mustRecipient(t, `{"email":"Cliente@Example.COM"}`)
	email, err := r.Email()
	require.NoError(t, err)
	require.Equal(t, "cliente@example.com", email)
}

func TestRecipient_Email_TrimsWhitespace(t *testing.T) {
	r := mustRecipient(t, `{"email":"  cliente@example.com  "}`)
	email, err := r.Email()
	require.NoError(t, err)
	require.Equal(t, "cliente@example.com", email)
}

func TestRecipient_ValidateForEmail_RejectsInvalid(t *testing.T) {
	invalid := []string{
		`{}`,
		`{"email":""}`,
		`{"email":"not-an-email"}`,
		`{"email":"@example.com"}`,
	}
	for _, raw := range invalid {
		t.Run(raw, func(t *testing.T) {
			r := mustRecipient(t, raw)
			err := r.ValidateFor(domain.ChannelEmail)
			require.ErrorIs(t, err, domain.ErrInvalidRecipient)
		})
	}
}

func TestRecipient_NewRecipient_RejectsEmpty(t *testing.T) {
	_, err := domain.NewRecipient(nil)
	require.ErrorIs(t, err, domain.ErrEmptyRecipient)

	_, err = domain.NewRecipient([]byte{})
	require.ErrorIs(t, err, domain.ErrEmptyRecipient)
}

func TestRecipient_NormalizedSearch_ReturnsEmail(t *testing.T) {
	r := mustRecipient(t, `{"email":"cliente@example.com"}`)
	s, err := r.NormalizedSearch(domain.ChannelEmail)
	require.NoError(t, err)
	require.Equal(t, "cliente@example.com", s)
}

func TestRecipient_Raw_PreservesInput(t *testing.T) {
	raw := `{"email":"cliente@example.com"}`
	r := mustRecipient(t, raw)
	require.JSONEq(t, raw, string(r.Raw()))
}