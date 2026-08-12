ALTER TABLE notifications
    ADD CONSTRAINT notifications_channel_check CHECK (channel IN ('EMAIL'));
