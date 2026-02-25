begin;

CREATE TABLE organization_agent_settings (
  id                          uuid NOT NULL DEFAULT uuid_generate_v4(),
  organization_id             uuid NOT NULL,
  agent_mode_enabled          boolean NOT NULL DEFAULT false,
  openai_api_key_ciphertext   BYTEA,
  openai_key_encryption_key_id CHARACTER VARYING(255),
  openai_key_last4            CHARACTER VARYING(8),
  openai_key_status           CHARACTER VARYING(32) NOT NULL DEFAULT 'not_configured',
  openai_key_validated_at     timestamp without time zone,
  openai_key_validation_error text,
  updated_by                  uuid,
  created_at                  timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at                  timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (id),
  CONSTRAINT organization_agent_settings_organization_id_key UNIQUE (organization_id),
  CONSTRAINT organization_agent_settings_openai_key_status_check
    CHECK (openai_key_status IN ('not_configured', 'valid', 'invalid', 'unchecked')),
  FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_organization_agent_settings_organization_id ON organization_agent_settings(organization_id);

commit;
