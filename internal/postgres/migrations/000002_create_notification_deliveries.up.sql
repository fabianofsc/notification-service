CREATE TABLE notification_deliveries (
    id                 uuid PRIMARY KEY,
    notification_id    uuid NOT NULL REFERENCES notifications(id),
    delivery_key       varchar(255) NOT NULL,
    status             varchar(16) NOT NULL DEFAULT 'PENDING',
    attempt_number     int NOT NULL,
    provider_response  text NOT NULL DEFAULT '',
    failure_reason     text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL,
    completed_at       timestamptz,

    CONSTRAINT delivery_status_check CHECK (status IN ('PENDING', 'SENT', 'FAILED'))
);

CREATE UNIQUE INDEX idx_deliveries_delivery_key ON notification_deliveries(delivery_key);
CREATE INDEX idx_deliveries_notification_id ON notification_deliveries(notification_id);