# SuperPlane Agents

Built with [Pydantic AI](https://ai.pydantic.dev/).

Starting points:

- System prompt is in `agent/src/ai/system_prompt.txt`
- Evals are in `agent/evals/cases.py`
- Patterns are in `agent/patterns`
- Run evals with `make test.agent.evals`
- Run a subset by name: `EVAL_CASES=github_and_slack,manual_run_then_two_noops make test.agent.evals`, or pass CLI flags: `make test.agent.evals AGENT_EVAL_RUNNER_ARGS='--cases github_and_slack'`
- List eval case names: `make test.agent.evals AGENT_EVAL_RUNNER_ARGS=--list-cases`
- Run unit tests with `make test.agent.unit`
- Lint with `make -C agent lint` (auto-fix: `make -C agent lint.fix`)
- Format with `make -C agent format` (check only: `make -C agent format.check`)
- Type check with `make -C agent typecheck`

Make changes the TDD way: always start by writing a failing eval.
