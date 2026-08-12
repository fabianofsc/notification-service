package email

import (
	"context"
	"log/slog"
)

type FakeProvider struct {
	FailKey string
	Log     *slog.Logger
}

func (p *FakeProvider) Send(ctx context.Context, to string, subject string, body string, deliveryKey string) (bool, error) {
	if p.FailKey != "" && deliveryKey == p.FailKey {
		p.Log.Info("email delivery failed (deterministic)",
			"delivery_key", deliveryKey,
		)
		return false, nil
	}
	p.Log.Info("email sent (simulated)",
		"delivery_key", deliveryKey,
	)
	return true, nil
}
