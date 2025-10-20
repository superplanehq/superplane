BEGIN;

ALTER TABLE role_metadata ADD COLUMN org_id VARCHAR(255);

ALTER TABLE role_metadata DROP CONSTRAINT uq_role_metadata_key;
ALTER TABLE role_metadata ADD CONSTRAINT uq_role_metadata_key UNIQUE (role_name, domain_type, domain_id, org_id);


CREATE INDEX idx_role_metadata_org_id ON role_metadata (org_id);

COMMIT;
