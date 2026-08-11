package sms_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/sms"
)

var _ app.SmsProvider = &sms.FakeProvider{}

func TestFakeSms_Send_ReturnsSuccess(t *testing.T) {
	p := &sms.FakeProvider{Log: slog.Default()}
	ok, err := p.Send(context.Background(), "+5511999999999", "", "Hello SMS", "dkey-1")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestFakeSms_Send_FailKeyMatch_Fails(t *testing.T) {
	p := &sms.FakeProvider{FailKey: "fail-me", Log: slog.Default()}
	ok, err := p.Send(context.Background(), "+5511999999999", "", "Hello", "fail-me")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFakeSms_Send_LogsWithoutSensitiveData(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	p := &sms.FakeProvider{Log: logger}

	p.Send(context.Background(), "+5511999999999", "", "secret body", "dkey-1")

	output := buf.String()
	require.Contains(t, output, "dkey-1")
	require.NotContains(t, output, "secret body")
	require.NotContains(t, output, "5511999999999")
}