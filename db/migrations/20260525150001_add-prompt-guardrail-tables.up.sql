begin;

-- Prompt guardrail policies: per-org (or per-workflow/node) enforcement configuration
CREATE TABLE IF NOT EXISTS prompt_guardrail_policies (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                          UUID NOT NULL,
    workflow_id                     UUID,
    node_id                         TEXT,
    component_type                  TEXT,

    enforcement_mode                TEXT NOT NULL DEFAULT 'audit_only',

    rule_overrides                  JSONB NOT NULL DEFAULT '{}',

    soft_block_score_threshold      INT NOT NULL DEFAULT 70,
    hard_block_score_threshold      INT NOT NULL DEFAULT 90,

    classifier_enabled              BOOLEAN NOT NULL DEFAULT FALSE,
    classifier_required_for_release BOOLEAN NOT NULL DEFAULT FALSE,
    classifier_sampling_rate        FLOAT NOT NULL DEFAULT 1.0,
    classifier_sensitivity          TEXT NOT NULL DEFAULT 'balanced',

    soft_block_timeout_seconds      INT NOT NULL DEFAULT 86400,

    provider_policies               JSONB NOT NULL DEFAULT '{}',

    created_by                      UUID NOT NULL,
    updated_by                      UUID,
    created_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_enforcement_mode CHECK (
        enforcement_mode IN ('audit_only', 'warn_only', 'soft_block', 'hard_block')
    ),
    CONSTRAINT valid_classifier_sensitivity CHECK (
        classifier_sensitivity IN ('strict', 'balanced', 'lenient')
    )
);

CREATE INDEX IF NOT EXISTS idx_prompt_guardrail_policies_org_id
    ON prompt_guardrail_policies(org_id);

CREATE INDEX IF NOT EXISTS idx_prompt_guardrail_policies_workflow_id
    ON prompt_guardrail_policies(workflow_id)
    WHERE workflow_id IS NOT NULL;

-- Prompt scan results: one record per guardrail checkpoint per execution
CREATE TABLE IF NOT EXISTS prompt_scan_results (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                UUID NOT NULL,
    workflow_id           UUID NOT NULL,
    execution_id          UUID NOT NULL,
    node_id               TEXT NOT NULL,

    scan_phase            TEXT NOT NULL,
    scan_direction        TEXT NOT NULL DEFAULT 'input',

    provider              TEXT,
    component_type        TEXT,

    risk_score            INT NOT NULL DEFAULT 0,
    enforcement_action    TEXT NOT NULL,
    execution_state       TEXT NOT NULL,

    findings              JSONB NOT NULL DEFAULT '[]',

    content_hash          TEXT,

    override_approved     BOOLEAN,
    override_approved_by  UUID,
    override_approved_at  TIMESTAMP WITH TIME ZONE,
    override_justification TEXT,
    override_expires_at   TIMESTAMP WITH TIME ZONE,

    classifier_result_id  UUID,

    created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_scan_results_execution_id
    ON prompt_scan_results(execution_id);

CREATE INDEX IF NOT EXISTS idx_prompt_scan_results_org_created
    ON prompt_scan_results(org_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prompt_scan_results_org_action_created
    ON prompt_scan_results(org_id, enforcement_action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_prompt_scan_results_pending_override
    ON prompt_scan_results(override_approved)
    WHERE override_approved IS NULL;

-- Prompt classifier results: async LLM classification job tracking
CREATE TABLE IF NOT EXISTS prompt_classifier_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id   UUID NOT NULL REFERENCES prompt_scan_results(id),

    status           TEXT NOT NULL DEFAULT 'pending',

    submitted_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at       TIMESTAMP WITH TIME ZONE,
    completed_at     TIMESTAMP WITH TIME ZONE,

    classifier_model   TEXT,
    classifier_version TEXT,

    risk_score       INT,
    findings         JSONB,
    raw_response     TEXT,
    token_count      INT,
    latency_ms       INT,

    error_code       TEXT,
    error_message    TEXT,
    retry_count      INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_prompt_classifier_results_scan_result_id
    ON prompt_classifier_results(scan_result_id);

CREATE INDEX IF NOT EXISTS idx_prompt_classifier_results_pending
    ON prompt_classifier_results(status, submitted_at)
    WHERE status = 'pending';

-- Prompt override approvals: audit trail for soft-block manual approvals
CREATE TABLE IF NOT EXISTS prompt_override_approvals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_result_id UUID NOT NULL REFERENCES prompt_scan_results(id),
    org_id         UUID NOT NULL,

    approved_by    UUID NOT NULL,
    approved_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    scope          TEXT NOT NULL DEFAULT 'execution',

    justification  TEXT NOT NULL,
    expires_at     TIMESTAMP WITH TIME ZONE,

    reviewer_ip    TEXT,
    mfa_verified   BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_prompt_override_approvals_scan_result_id
    ON prompt_override_approvals(scan_result_id);

CREATE INDEX IF NOT EXISTS idx_prompt_override_approvals_org_id
    ON prompt_override_approvals(org_id);

-- Prompt guardrail bypass tokens: allow specific executions to skip scanning
CREATE TABLE IF NOT EXISTS prompt_guardrail_bypass_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token        TEXT NOT NULL UNIQUE,
    org_id       UUID NOT NULL,
    workflow_id  UUID,
    node_id      TEXT,
    rules        JSONB NOT NULL DEFAULT '[]',
    issued_by    UUID NOT NULL,
    expires_at   TIMESTAMP WITH TIME ZONE NOT NULL,
    usage_limit  INT NOT NULL DEFAULT 1,
    usage_count  INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_guardrail_bypass_tokens_org_id
    ON prompt_guardrail_bypass_tokens(org_id);

CREATE INDEX IF NOT EXISTS idx_prompt_guardrail_bypass_tokens_token
    ON prompt_guardrail_bypass_tokens(token);

commit;
