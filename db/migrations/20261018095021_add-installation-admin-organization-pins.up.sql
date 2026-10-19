BEGIN;

CREATE TABLE installation_admin_organization_pins (
  account_id      uuid NOT NULL,
  organization_id uuid NOT NULL,
  pinned_at       timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at      timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT installation_admin_organization_pins_pkey PRIMARY KEY (account_id, organization_id),
  CONSTRAINT installation_admin_organization_pins_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
  CONSTRAINT installation_admin_organization_pins_organization_id_fkey
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX installation_admin_organization_pins_account_pinned_at_idx
  ON installation_admin_organization_pins (account_id, pinned_at DESC);

COMMIT;
