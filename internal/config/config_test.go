package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/config"
)

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":        "postgres://notification:notification@localhost:5432/notification_service?sslmode=disable",
		"BASIC_AUTH_USERNAME": "svc",
		"BASIC_AUTH_PASSWORD": "s3cret",
		"PORT":                "9090",
		"LEASE_DURATION":      "10s",
		"POLL_INTERVAL":       "1s",
		"BATCH_SIZE":          "20",
		"MAX_CONCURRENCY":     "8",
	}
}

func lookupFrom(env map[string]string) config.Lookup {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func TestLoad_AllValid_ProducesExpectedConfig(t *testing.T) {
	cfg, err := config.Load(lookupFrom(validEnv()))
	require.NoError(t, err)

	require.Equal(t, ":9090", cfg.HTTPAddr)
	require.Equal(t, "postgres://notification:notification@localhost:5432/notification_service?sslmode=disable", cfg.DatabaseURL)
	require.Equal(t, "svc", cfg.BasicAuthUser)
	require.Equal(t, "s3cret", cfg.BasicAuthPass)
	require.Equal(t, 10*time.Second, cfg.LeaseDuration)
	require.Equal(t, 1*time.Second, cfg.PollInterval)
	require.Equal(t, 20, cfg.BatchSize)
	require.Equal(t, 8, cfg.MaxConcurrency)
}

func TestLoad_Defaults_ApplyWhenOptionalsAbsent(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL": "postgres://n:n@localhost:5432/db?sslmode=disable",
	}

	cfg, err := config.Load(lookupFrom(env))
	require.NoError(t, err)

	require.Equal(t, ":8080", cfg.HTTPAddr)
	require.Equal(t, "notification", cfg.BasicAuthUser)
	require.Equal(t, "notification", cfg.BasicAuthPass)
	require.Equal(t, 30*time.Second, cfg.LeaseDuration)
	require.Equal(t, 2*time.Second, cfg.PollInterval)
	require.Equal(t, 10, cfg.BatchSize)
	require.Equal(t, 5, cfg.MaxConcurrency)
}

func TestLoad_DatabaseURL_Required(t *testing.T) {
	_, err := config.Load(lookupFrom(map[string]string{}))
	require.Error(t, err)
	require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_DatabaseURL_Missing(t *testing.T) {
	_, err := config.Load(lookupFrom(map[string]string{"DATABASE_URL": ""}))
	require.Error(t, err)
	require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_DatabaseURL_InvalidScheme(t *testing.T) {
	env := validEnv()
	env["DATABASE_URL"] = "mysql://localhost/db"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_DatabaseURL_NotAURL(t *testing.T) {
	env := validEnv()
	env["DATABASE_URL"] = "not-a-url"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_Port_Invalid(t *testing.T) {
	env := validEnv()
	env["PORT"] = "not-a-port"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "PORT")
}

func TestLoad_LeaseDuration_Invalid(t *testing.T) {
	env := validEnv()
	env["LEASE_DURATION"] = "not-a-duration"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "LEASE_DURATION")
}

func TestLoad_LeaseDuration_Zero(t *testing.T) {
	env := validEnv()
	env["LEASE_DURATION"] = "0s"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "LEASE_DURATION")
}

func TestLoad_LeaseDuration_Negative(t *testing.T) {
	env := validEnv()
	env["LEASE_DURATION"] = "-5s"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "LEASE_DURATION")
}

func TestLoad_PollInterval_Invalid(t *testing.T) {
	env := validEnv()
	env["POLL_INTERVAL"] = "soon"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "POLL_INTERVAL")
}

func TestLoad_BatchSize_Invalid(t *testing.T) {
	env := validEnv()
	env["BATCH_SIZE"] = "many"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "BATCH_SIZE")
}

func TestLoad_BatchSize_Zero(t *testing.T) {
	env := validEnv()
	env["BATCH_SIZE"] = "0"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "BATCH_SIZE")
}

func TestLoad_MaxConcurrency_Invalid(t *testing.T) {
	env := validEnv()
	env["MAX_CONCURRENCY"] = "lots"
	_, err := config.Load(lookupFrom(env))
	require.Error(t, err)
	require.ErrorContains(t, err, "MAX_CONCURRENCY")
}

func TestLoad_BasicAuth_DefaultsWhenEmpty(t *testing.T) {
	env := validEnv()
	env["BASIC_AUTH_USERNAME"] = ""
	env["BASIC_AUTH_PASSWORD"] = ""
	cfg, err := config.Load(lookupFrom(env))
	require.NoError(t, err)
	require.Equal(t, "notification", cfg.BasicAuthUser)
	require.Equal(t, "notification", cfg.BasicAuthPass)
}