CREATE TABLE factory_human_review_decisions (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations(id),
    factory_id uuid NOT NULL REFERENCES factories(id),
    pull_request_id uuid NOT NULL REFERENCES factory_pull_requests(id) ON DELETE CASCADE,
    revision_id uuid NOT NULL REFERENCES factory_pull_request_revisions(id),
    source_run_id uuid REFERENCES workflow_runs(id),
    run_id uuid REFERENCES workflow_runs(id),
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
    publication_pending boolean NOT NULL DEFAULT true,
    publication_error text NOT NULL DEFAULT '',
    retry_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX factory_human_review_current ON factory_human_review_decisions (pull_request_id) WHERE state <> 'superseded';
CREATE UNIQUE INDEX factory_human_review_run ON factory_human_review_decisions (run_id) WHERE run_id IS NOT NULL;
