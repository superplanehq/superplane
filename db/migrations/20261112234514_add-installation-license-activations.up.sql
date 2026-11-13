CREATE TABLE installation_license_activations (
    id                  bigserial   PRIMARY KEY,
    license_id          uuid        NOT NULL,
    license_features    text[]      NOT NULL,
    activated_at        timestamptz NOT NULL DEFAULT now(),
    organization_count  integer     NOT NULL,
    auto_granted        boolean     NOT NULL DEFAULT false,
    deactivated_at      timestamptz NULL,
    deactivation_reason text        NULL,
    removed_grants      jsonb       NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT installation_license_activations_reason_check
        CHECK (deactivation_reason IS NULL OR deactivation_reason IN
               ('expired', 'revoked', 'removed', 'replaced')),
    CONSTRAINT installation_license_activations_closed_check
        CHECK ((deactivated_at IS NULL) = (deactivation_reason IS NULL)),
    CONSTRAINT installation_license_activations_org_count_check
        CHECK (organization_count >= 0)
);

CREATE UNIQUE INDEX installation_license_activations_one_open
    ON installation_license_activations ((true))
    WHERE deactivated_at IS NULL;

CREATE INDEX installation_license_activations_license_id_idx
    ON installation_license_activations (license_id);

CREATE INDEX installation_license_activations_activated_at_idx
    ON installation_license_activations (activated_at DESC);
