# AI flow evals

These tests validate planner behavior by calling OpenAI directly through the same canvas planning code path used by `send_ai_message`.

## Run

```bash
OPENAI_EVALS=1 OPENAI_API_KEY=... make test PKG_TEST_PACKAGES=./test/evals
```

Without `OPENAI_EVALS=1`, tests are skipped.

Optional flakiness control:

```bash
OPENAI_EVAL_ATTEMPTS=3
```

Each eval retries until an invariant passes, up to the configured attempts.

## Naming

Each file is named after the behavior it validates, for example:

- `repo_reuse_eval_test.go`
- `pr_action_selection_eval_test.go`
- `setdata_configuration_eval_test.go`
- `getdata_lookup_eval_test.go`
- `getdata_emit_each_item_eval_test.go`
- `graph_rewiring_eval_test.go`
- `schedule_utc_default_eval_test.go`

