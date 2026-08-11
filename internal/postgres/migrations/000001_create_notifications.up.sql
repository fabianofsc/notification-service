CREATE TABLE notifications (
    id                  uuid PRIMARY KEY,
    notification_key    varchar(255) NOT NULL,
    payload_fingerprint text NOT NULL,
    channel             varchar(16) NOT NULL,
    recipient           jsonb NOT NULL,
    recipient_search    varchar NOT NULL,
    subject             varchar(180) NOT NULL,
    body                varchar(2000) NOT NULL,
    reference_id        text NOT NULL DEFAULT '',
    status              varchar(16) NOT NULL DEFAULT 'PENDING',
    lease_token         uuid,
    lease_until         timestamptz,
    attempt_count       int NOT NULL DEFAULT 0,
    failure_reason      text NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL,
    sent_at             timestamptz,
    updated_at          timestamptz NOT NULL,

    CONSTRAINT notifications_status_check CHECK (status IN ('PENDING', 'SENDING', 'SENT', 'FAILED'))
);

CREATE UNIQUE INDEX idx_notifications_notification_key ON notifications(notification_key);
CREATE INDEX idx_notifications_recipient_search ON notifications(recipient_search);
CREATE INDEX idx_notifications_status ON notifications(status);