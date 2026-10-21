# Built-in automation settings

When adding or changing a supported user setting, also maintain its behavior
during `ResetOrganizationBuiltInAutomations`. Database persistence alone does
not guarantee that the setting survives: reset rebuilds the automation from
the current template.

- Extend the relevant settings reader and builder input. Shared agent options
  belong in `factoryAgentSettingFields` in `factory_agent_settings.go`. Intake,
  PR-feedback, scheduled-intake, and merge-check settings have their own readers.
- Preserve explicit false, empty, and zero values where supported. Apply
  template defaults only when a setting is absent.
- Keep managed prompts and workflow structure from the current template.
  Preserve supported user instructions separately from managed instructions.
- Extend `reset_built_in_automation_settings_test.go` to save a non-default
  value, reset the organization, and verify the published result. Cover the
  affected automation types and providers. Check execution behavior when a
  setting controls generated filters or gates.
- Keep admin reset distinct from the user-facing restore-defaults action.
  Keep organization reset atomic when saved settings cannot be applied.
- An installation reset of one automation runs that same organization reset
  for the selected automation only. Continue with the other organizations when
  one organization cannot apply its saved settings.

Run `make test PKG_TEST_PACKAGES=./pkg/grpc/actions/factories` from the repository
root, plus the Go checks required by the root `AGENTS.md`.
