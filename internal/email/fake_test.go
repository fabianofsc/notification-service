package email_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/email"
)

type bufHandler struct {
	buf *bytes.Buffer
	h   slog.Handler
}

func (b *bufHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return b.h.Enabled(ctx, l)
}

func (b *bufHandler) Handle(ctx context.Context, r slog.Record) error {
	b.buf.WriteString(r.Message)
	b.buf.WriteString(" ")
	r.Attrs(func(a slog.Attr) bool {
		b.buf.WriteString(a.String())
		b.buf.WriteString(" ")
		return true
	})
	return b.h.Handle(ctx, r)
}

func (b *bufHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufHandler{buf: b.buf, h: b.h.WithAttrs(attrs)}
}

func (b *bufHandler) WithGroup(name string) slog.Handler {
	return &bufHandler{buf: b.buf, h: b.h.WithGroup(name)}
}

func newBufLogger() (*slog.Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	h := &bufHandler{buf: buf, h: slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})}
	return slog.New(h), buf
}

func TestFakeProvider_Send_ReturnsTrueByDefault(t *testing.T) {
	logger, _ := newBufLogger()
	p := &email.FakeProvider{Log: logger}

	success, err := p.Send(context.Background(), "user@example.com", "Hello", "Body text", "dly_key_01")

	require.NoError(t, err)
	require.True(t, success)
}

func TestFakeProvider_Send_ReturnsFalseWhenFailKeyMatches(t *testing.T) {
	logger, _ := newBufLogger()
	p := &email.FakeProvider{FailKey: "dly_fail_01", Log: logger}

	success, err := p.Send(context.Background(), "user@example.com", "Hello", "Body text", "dly_fail_01")

	require.NoError(t, err)
	require.False(t, success)
}

func TestFakeProvider_Send_DifferentFailKeyStillSucceeds(t *testing.T) {
	logger, _ := newBufLogger()
	p := &email.FakeProvider{FailKey: "dly_fail_01", Log: logger}

	success, err := p.Send(context.Background(), "user@example.com", "Hello", "Body text", "dly_other")

	require.NoError(t, err)
	require.True(t, success)
}

func TestFakeProvider_Send_LogsNeverContainSubjectBodyOrEmail(t *testing.T) {
	logger, buf := newBufLogger()
	p := &email.FakeProvider{Log: logger}

	_, err := p.Send(context.Background(), "user@example.com", "Sensitive subject", "Confidential body", "dly_key_01")
	require.NoError(t, err)

	logOutput := buf.String()
	require.NotContains(t, logOutput, "Sensitive subject")
	require.NotContains(t, logOutput, "Confidential body")
	require.NotContains(t, logOutput, "user@example.com")
}

func TestFakeProvider_Send_FailureLogsDeliverKey(t *testing.T) {
	logger, buf := newBufLogger()
	p := &email.FakeProvider{FailKey: "dly_fail_01", Log: logger}

	_, err := p.Send(context.Background(), "user@example.com", "Subject", "Body", "dly_fail_01")
	require.NoError(t, err)

	logOutput := buf.String()
	require.Contains(t, logOutput, "dly_fail_01")
}

func TestFakeProvider_Send_ZeroFailKeyAlwaysSucceeds(t *testing.T) {
	logger, _ := newBufLogger()
	p := &email.FakeProvider{Log: logger}

	tests := []struct {
		name        string
		deliveryKey string
	}{
		{"empty delivery key", ""},
		{"normal delivery key", "dly_abc_123"},
		{"key that looks like empty string match", "dly_empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, err := p.Send(context.Background(), "user@example.com", "Hello", "Body", tt.deliveryKey)
			require.NoError(t, err)
			require.True(t, success)
		})
	}
}