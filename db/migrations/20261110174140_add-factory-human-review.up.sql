CREATE TABLE factory_human_review_decisions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    factory_id uuid NOT NULL REFERENCES factories(id),
    pull_request_id uuid NOT NULL REFERENCES factory_pull_requests(id) ON DELETE CASCADE,
    revision_id uuid NOT NULL REFERENCES factory_pull_request_revisions(id),
    source_run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL,
    run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL,
    configuration_version text NOT NULL,
    state text NOT NULL,
    assessed_commit text NOT NULL,
    assessment jsonb NOT NULL DEFAULT '{}',
    result jsonb NOT NULL DEFAULT '{}',
    review_required boolean NOT NULL DEFAULT false,
    rationale text NOT NULL DEFAULT '',
    selected_reviewers jsonb NOT NULL DEFAULT '[]',
    approval_ids jsonb NOT NULL DEFAULT '[]',
    blocking_reason text NOT NULL DEFAULT '',
    check_run_id bigint NOT NULL DEFAULT 0,
    check_url text NOT NULL DEFAULT '',
    publication_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX factory_human_review_current ON factory_human_review_decisions (pull_request_id) WHERE state <> 'superseded';
CREATE UNIQUE INDEX factory_human_review_run ON factory_human_review_decisions (run_id) WHERE run_id IS NOT NULL;

ALTER TABLE factory_pull_requests ADD COLUMN human_review_request_id uuid;

CREATE INDEX idx_factory_pull_requests_human_review_pending
  ON factory_pull_requests (id)
  WHERE human_review_request_id IS NOT NULL;

CREATE INDEX factory_human_review_source_run
    ON factory_human_review_decisions (source_run_id)
    WHERE source_run_id IS NOT NULL;

CREATE INDEX factory_human_review_factory
    ON factory_human_review_decisions (factory_id);
