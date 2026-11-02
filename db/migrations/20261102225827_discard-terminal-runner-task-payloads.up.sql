ALTER TABLE runner_tasks
  ALTER COLUMN payload_ciphertext DROP NOT NULL,
  ADD CONSTRAINT runner_tasks_nonterminal_payload
    CHECK (state IN ('succeeded', 'failed', 'canceled', 'lost') OR payload_ciphertext IS NOT NULL);
