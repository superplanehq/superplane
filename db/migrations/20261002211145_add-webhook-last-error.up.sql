begin;

ALTER TABLE webhooks
  ADD COLUMN last_error TEXT NOT NULL DEFAULT '';

commit;
