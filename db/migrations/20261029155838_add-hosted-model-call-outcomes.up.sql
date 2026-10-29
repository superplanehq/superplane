BEGIN;

CREATE TABLE hosted_model_call_outcomes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  outcome TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  runner_task_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  CONSTRAINT hosted_model_call_outcomes_provider_known
    CHECK (provider IN ('anthropic', 'openai', 'openrouter')),
  CONSTRAINT hosted_model_call_outcomes_outcome_known
    CHECK (outcome IN ('attempt_1', 'attempt_2', 'attempt_3', 'attempt_4', 'error')),
  CONSTRAINT hosted_model_call_outcomes_task_sequence_key
    UNIQUE (runner_task_id, sequence)
);

CREATE INDEX idx_hosted_model_call_outcomes_occurred_at
  ON hosted_model_call_outcomes (occurred_at);

COMMIT;
