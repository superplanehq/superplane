CREATE TABLE organization_enterprise_features (
    organization_id uuid        NOT NULL
        REFERENCES organizations(id) ON DELETE CASCADE,
    feature_key     text        NOT NULL,
    granted_by      uuid        NULL,
    source          text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, feature_key),
    CONSTRAINT organization_enterprise_features_key_check
        CHECK (feature_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT organization_enterprise_features_source_check
        CHECK (source IN ('admin', 'auto_single_org'))
);

CREATE INDEX organization_enterprise_features_feature_key_idx
    ON organization_enterprise_features (feature_key);
