ALTER TABLE notifications ADD COLUMN callback_id text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN callback_name text NOT NULL DEFAULT '';