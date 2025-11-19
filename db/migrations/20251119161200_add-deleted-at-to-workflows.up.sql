begin;

ALTER TABLE workflows ADD COLUMN deleted_at TIMESTAMP NULL;
CREATE INDEX idx_workflows_deleted_at ON workflows(deleted_at);

commit;