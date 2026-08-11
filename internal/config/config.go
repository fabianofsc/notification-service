package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	BasicAuthUser    string
	BasicAuthPass    string
	LeaseDuration    time.Duration
	PollInterval     time.Duration
	BatchSize        int
	MaxConcurrency   int
}

type Lookup func(key string) (value string, ok bool)

const (
	defaultHTTPAddr       = ":8080"
	defaultBasicAuthUser  = "notification"
	defaultBasicAuthPass  = "notification"
	defaultLeaseDuration  = 30 * time.Second
	defaultPollInterval   = 2 * time.Second
	defaultBatchSize      = 10
	defaultMaxConcurrency = 5
)

func Load(lookup Lookup) (Config, error) {
	var cfg Config
	var err error

	if cfg.HTTPAddr, err = optionalListenAddr(lookup, "PORT", defaultHTTPAddr); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseURL, err = requiredPostgresDSN(lookup, "DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.BasicAuthUser, err = optionalNonEmpty(lookup, "BASIC_AUTH_USERNAME", defaultBasicAuthUser); err != nil {
		return Config{}, err
	}
	if cfg.BasicAuthPass, err = optionalNonEmpty(lookup, "BASIC_AUTH_PASSWORD", defaultBasicAuthPass); err != nil {
		return Config{}, err
	}
	if cfg.LeaseDuration, err = optionalDuration(lookup, "LEASE_DURATION", defaultLeaseDuration); err != nil {
		return Config{}, err
	}
	if cfg.PollInterval, err = optionalDuration(lookup, "POLL_INTERVAL", defaultPollInterval); err != nil {
		return Config{}, err
	}
	if cfg.BatchSize, err = optionalPositiveInt(lookup, "BATCH_SIZE", defaultBatchSize); err != nil {
		return Config{}, err
	}
	if cfg.MaxConcurrency, err = optionalPositiveInt(lookup, "MAX_CONCURRENCY", defaultMaxConcurrency); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func fail(variable, reason string) error {
	return fmt.Errorf("%s: %s", variable, reason)
}

func requiredPostgresDSN(lookup Lookup, variable string) (string, error) {
	v, ok := lookup(variable)
	if !ok || v == "" {
		return "", fail(variable, "required and must not be empty")
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", fail(variable, "must be a valid postgres:// DSN")
	}
	return v, nil
}

func optionalNonEmpty(lookup Lookup, variable, def string) (string, error) {
	v, ok := lookup(variable)
	if !ok || v == "" {
		return def, nil
	}
	return v, nil
}

func optionalListenAddr(lookup Lookup, variable, def string) (string, error) {
	v, ok := lookup(variable)
	if !ok || v == "" {
		v = def
	} else {
		if _, err := strconv.Atoi(v); err != nil {
			return "", fail(variable, "must be a valid port number")
		}
		v = ":" + v
	}
	if _, _, err := net.SplitHostPort(v); err != nil {
		return "", fail(variable, "must be a valid listen address")
	}
	return v, nil
}

func optionalDuration(lookup Lookup, variable string, def time.Duration) (time.Duration, error) {
	v, ok := lookup(variable)
	if !ok || v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fail(variable, "must be a valid Go duration")
	}
	if d <= 0 {
		return 0, fail(variable, "must be positive")
	}
	return d, nil
}

func optionalPositiveInt(lookup Lookup, variable string, def int) (int, error) {
	v, ok := lookup(variable)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fail(variable, "must be a valid integer")
	}
	if n <= 0 {
		return 0, fail(variable, "must be positive")
	}
	return n, nil
}