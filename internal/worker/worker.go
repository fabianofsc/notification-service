package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

type WorkerDeps struct {
	Notifications app.NotificationRepository
	Deliveries    app.DeliveryRepository
	Email         app.EmailProvider
	Sms           app.SmsProvider
	Clock         app.Clock
	IDs           app.IDGenerator
	Log           *slog.Logger
}

type WorkerConfig struct {
	PollInterval   time.Duration
	BatchSize      int
	LeaseDuration  time.Duration
	MaxConcurrency int
}

type Worker struct {
	notifications  app.NotificationRepository
	deliveries     app.DeliveryRepository
	email          app.EmailProvider
	sms            app.SmsProvider
	clock          app.Clock
	ids            app.IDGenerator
	log            *slog.Logger
	pollInterval   time.Duration
	batchSize      int
	leaseDuration  time.Duration
	maxConcurrency int
}

func New(deps WorkerDeps, cfg WorkerConfig) *Worker {
	return &Worker{
		notifications:  deps.Notifications,
		deliveries:     deps.Deliveries,
		email:          deps.Email,
		sms:            deps.Sms,
		clock:          deps.Clock,
		ids:            deps.IDs,
		log:            deps.Log,
		pollInterval:   cfg.PollInterval,
		batchSize:      cfg.BatchSize,
		leaseDuration:  cfg.LeaseDuration,
		maxConcurrency: cfg.MaxConcurrency,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.log.Info("worker started",
		"poll_interval", w.pollInterval.String(),
		"batch_size", w.batchSize,
		"lease_duration", w.leaseDuration.String(),
	)
	defer w.log.Info("worker stopped")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.processBatch(context.Background())
			return nil
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	claimed, err := w.notifications.ClaimBatch(ctx, w.batchSize, w.leaseDuration)
	if err != nil {
		w.log.Error("claim batch failed", "error", err)
		return
	}

	if len(claimed) == 0 {
		return
	}

	sem := make(chan struct{}, w.maxConcurrency)
	var wg sync.WaitGroup

	for _, n := range claimed {
		wg.Add(1)
		go func(n domain.Notification) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			w.processOne(ctx, n)
		}(n)
	}
	wg.Wait()
}

func (w *Worker) processOne(ctx context.Context, n domain.Notification) {
	w.log.Info("notification claimed",
		"notification_id", n.ID.String(),
		"notification_key", n.NotificationKey,
		"channel", string(n.Channel),
	)

	to, err := w.resolveDestination(n)
	if err != nil {
		w.failNotification(ctx, n, fmt.Sprintf("invalid recipient: %s", err))
		return
	}

	deliveryID := w.ids.NewID()
	deliveryKey := deliveryID.String()
	now := w.clock.Now()

	delivery := domain.NewDelivery(deliveryID, n.ID, deliveryKey, n.AttemptCount, now)
	if err := w.deliveries.InsertDelivery(ctx, delivery); err != nil {
		w.log.Error("insert delivery failed",
			"error", err,
			"delivery_key", deliveryKey,
			"notification_id", n.ID.String(),
		)
		return
	}

	ok, sendErr := w.dispatch(ctx, n, to, deliveryKey)
	now = w.clock.Now()

	if sendErr == nil && ok {
		completed, completeErr := w.notifications.Complete(ctx, n.ID, domain.StatusSent, n.LeaseToken, now, "")
		if completeErr != nil {
			w.log.Error("complete notification failed",
				"error", completeErr,
				"notification_id", n.ID.String(),
			)
			return
		}
		if !completed {
			w.log.Warn("complete notification no rows affected",
				"notification_id", n.ID.String(),
			)
			return
		}
		if err := w.deliveries.CompleteDeliverySuccess(ctx, deliveryID, "", now); err != nil {
			w.log.Error("complete delivery success failed",
				"error", err,
				"delivery_key", deliveryKey,
			)
		}
		w.log.Info("delivery completed",
			"delivery_key", deliveryKey,
			"success", true,
		)
	} else {
		reason := "send failed"
		if sendErr != nil {
			reason = sendErr.Error()
		}
		completed, completeErr := w.notifications.Complete(ctx, n.ID, domain.StatusFailed, n.LeaseToken, now, reason)
		if completeErr != nil {
			w.log.Error("complete notification failed",
				"error", completeErr,
				"notification_id", n.ID.String(),
			)
			return
		}
		if !completed {
			w.log.Warn("complete notification no rows affected",
				"notification_id", n.ID.String(),
			)
			return
		}
		if err := w.deliveries.CompleteDeliveryFailure(ctx, deliveryID, reason, now); err != nil {
			w.log.Error("complete delivery failure failed",
				"error", err,
				"delivery_key", deliveryKey,
			)
		}
		w.log.Info("delivery completed",
			"delivery_key", deliveryKey,
			"success", false,
		)
	}
}

func (w *Worker) dispatch(ctx context.Context, n domain.Notification, to string, deliveryKey string) (bool, error) {
	switch n.Channel {
	case domain.ChannelEmail:
		return w.email.Send(ctx, to, n.Subject, n.Body, deliveryKey)
	case domain.ChannelSMS:
		return w.sms.Send(ctx, to, n.Subject, n.Body, deliveryKey)
	default:
		return false, fmt.Errorf("unsupported channel: %s", n.Channel)
	}
}

func (w *Worker) resolveDestination(n domain.Notification) (string, error) {
	switch n.Channel {
	case domain.ChannelEmail:
		return n.Recipient.Email()
	case domain.ChannelSMS:
		return n.Recipient.PhoneNumber()
	default:
		return "", fmt.Errorf("unsupported channel: %s", n.Channel)
	}
}

func (w *Worker) failNotification(ctx context.Context, n domain.Notification, reason string) {
	now := w.clock.Now()
	completed, err := w.notifications.Complete(ctx, n.ID, domain.StatusFailed, n.LeaseToken, now, reason)
	if err != nil {
		w.log.Error("complete notification failed",
			"error", err,
			"notification_id", n.ID.String(),
		)
		return
	}
	if !completed {
		w.log.Warn("complete notification no rows affected",
			"notification_id", n.ID.String(),
		)
	}
}