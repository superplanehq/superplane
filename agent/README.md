# SuperPlane Agents

Built with [Pydantic AI](https://ai.pydantic.dev/).

Starting points:

- System prompt is in `agent/src/ai/system_prompt.txt`
- Evals are in `agent/evals/cases.py`
- Patterns are in `agent/patterns`
- Run evals with `make test.agent.evals`
- Run a subset by case `name` from `evals/cases.py` (comma-separated; one or many): `make test.agent.evals CASES=github_and_slack` or `CASES=github_and_slack,manual_run_then_two_noops`
- List eval case names: `make test.agent.evals AGENT_EVAL_RUNNER_ARGS=--list-cases`
- Run unit tests with `make test.agent.unit`
- Lint with `make -C agent lint` (auto-fix: `make -C agent lint.fix`)
- Format with `make -C agent format` (check only: `make -C agent format.check`)
- Type check with `make -C agent typecheck`

Make changes the TDD way: always start by writing a failing eval.
