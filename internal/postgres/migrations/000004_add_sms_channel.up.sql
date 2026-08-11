-- Migration 000001 no longer creates notifications_channel_check.
-- Channel validation is domain-only. This migration is a no-op;
-- it exists so migration numbering is contiguous.
SELECT 1;