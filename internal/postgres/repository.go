package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, n domain.Notification) (app.InsertNotificationResult, error) {
	search, err := n.Recipient.NormalizedSearch(n.Channel)
	if err != nil {
		return app.InsertNotificationResult{}, err
	}

	row := r.pool.QueryRow(ctx,
		`INSERT INTO notifications (
			id, notification_key, payload_fingerprint, channel, recipient, recipient_search,
			subject, body, reference_id, callback_id, callback_name, status, attempt_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (notification_key) DO NOTHING
		RETURNING id, notification_key, payload_fingerprint, channel, recipient, recipient_search,
			subject, body, reference_id, callback_id, callback_name,
			status, lease_token, lease_until,
			attempt_count, failure_reason, created_at, sent_at, updated_at`,
		n.ID, n.NotificationKey, n.PayloadFingerprint, string(n.Channel), n.Recipient.Raw(), search,
		n.Subject, n.Body, n.ReferenceID, n.CallbackID, n.CallbackName, string(n.Status), n.AttemptCount, n.CreatedAt, n.UpdatedAt,
	)

	inserted, scanErr := scanNotification(row)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		existing, findErr := r.FindByNotificationKey(ctx, n.NotificationKey)
		if findErr != nil {
			return app.InsertNotificationResult{}, findErr
		}
		return app.InsertNotificationResult{Notification: existing, Replayed: true}, nil
	}
	if scanErr != nil {
		return app.InsertNotificationResult{}, scanErr
	}
	return app.InsertNotificationResult{Notification: inserted}, nil
}

func (r *Repository) ClaimBatch(ctx context.Context, batchSize int, leaseDuration time.Duration, now time.Time) ([]domain.Notification, error) {
	leaseUntil := now.Add(leaseDuration)

	rows, err := r.pool.Query(ctx,
		`UPDATE notifications
		SET status = 'SENDING',
		    lease_token = gen_random_uuid(),
		    lease_until = $1,
		    attempt_count = attempt_count + 1,
		    updated_at = $2
		WHERE id IN (
			SELECT id FROM notifications
			WHERE status = 'PENDING'
			   OR (status = 'SENDING' AND lease_until < $2)
			ORDER BY created_at
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, notification_key, payload_fingerprint, channel, recipient, recipient_search,
			subject, body, reference_id, callback_id, callback_name,
			status, lease_token, lease_until,
			attempt_count, failure_reason, created_at, sent_at, updated_at`,
		leaseUntil, now, batchSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}

	return result, rows.Err()
}

func (r *Repository) Complete(ctx context.Context, id uuid.UUID, status domain.Status, leaseToken uuid.UUID, now time.Time, failureReason string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications
		SET status = $1::text,
		    lease_token = NULL,
		    sent_at = CASE WHEN $1::text = 'SENT' THEN $2 ELSE sent_at END,
		    failure_reason = CASE WHEN $1::text = 'FAILED' THEN $3 ELSE failure_reason END,
		    updated_at = $2
		WHERE id = $4
		  AND lease_token = $5
		  AND status = 'SENDING'`,
		string(status), now, failureReason, id, leaseToken,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (domain.Notification, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, notification_key, payload_fingerprint, channel, recipient, recipient_search,
			subject, body, reference_id, callback_id, callback_name,
			status, lease_token, lease_until,
			attempt_count, failure_reason, created_at, sent_at, updated_at
		FROM notifications WHERE id = $1`, id,
	)
	n, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	if err != nil {
		return domain.Notification{}, err
	}
	return n, nil
}

func (r *Repository) FindByNotificationKey(ctx context.Context, key string) (domain.Notification, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, notification_key, payload_fingerprint, channel, recipient, recipient_search,
			subject, body, reference_id, callback_id, callback_name,
			status, lease_token, lease_until,
			attempt_count, failure_reason, created_at, sent_at, updated_at
		FROM notifications WHERE notification_key = $1`, key,
	)
	n, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	if err != nil {
		return domain.Notification{}, err
	}
	return n, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (domain.Notification, error) {
	var n domain.Notification
	var channelStr, statusStr string
	var recipientRaw []byte
	var searchStr string
	var sentAt, leaseUntil *time.Time

	err := row.Scan(
		&n.ID, &n.NotificationKey, &n.PayloadFingerprint,
		&channelStr, &recipientRaw, &searchStr,
		&n.Subject, &n.Body, &n.ReferenceID,
		&n.CallbackID, &n.CallbackName,
		&statusStr, &n.LeaseToken, &leaseUntil,
		&n.AttemptCount, &n.FailureReason,
		&n.CreatedAt, &sentAt, &n.UpdatedAt,
	)
	if err != nil {
		return domain.Notification{}, err
	}

	n.Channel = domain.Channel(channelStr)
	n.Status = domain.Status(statusStr)
	n.Recipient, _ = domain.NewRecipient(recipientRaw)
	if sentAt != nil {
		n.SentAt = *sentAt
	}
	if leaseUntil != nil {
		n.LeaseUntil = *leaseUntil
	}

	return n, nil
}

func (r *Repository) InsertDelivery(ctx context.Context, d domain.Delivery) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_deliveries (
			id, notification_id, delivery_key, status, attempt_number, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		d.ID, d.NotificationID, d.DeliveryKey, string(d.Status), d.AttemptNumber, d.CreatedAt,
	)
	return err
}

func (r *Repository) CompleteDeliverySuccess(ctx context.Context, id uuid.UUID, response string, now time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notification_deliveries
		SET status = 'SENT', provider_response = $1, completed_at = $2
		WHERE id = $3`,
		response, now, id,
	)
	return err
}

func (r *Repository) CompleteDeliveryFailure(ctx context.Context, id uuid.UUID, reason string, now time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notification_deliveries
		SET status = 'FAILED', failure_reason = $1, completed_at = $2
		WHERE id = $3`,
		reason, now, id,
	)
	return err
}
