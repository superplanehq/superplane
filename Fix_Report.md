# Fix Report - Issue #2818

## Summary
Completed the implementation of the `List Releases` component for the GitHub integration. This includes the component logic, test coverage, and documentation. Additionally, addressed interface compliance issues and ensured alignment with recent architectural changes (#2838).

## Changes

### 1. GitHub: List Releases Component
- Implemented `ListReleases` struct in `pkg/integrations/github/list_releases.go`.
- Added support for pagination (`PerPage`, `Page`).
- Integrated with `go-github` to fetch releases from the specified repository.
- Registered the component in `pkg/integrations/github/github.go`.
- Added example output in `pkg/integrations/github/example_output_list_releases.json`.
- **Frontend**: Implemented UI mapper in `web_src/src/pages/workflowv2/mappers/github/list_releases.ts` and registered it in `index.ts` to provide rich execution details in the workflow view.

### 2. Testing & Refactoring
- **Testable Client**: Refactored `NewClient` in `pkg/integrations/github/client.go` to support custom `http.RoundTripper`. Added `NewClientWithTransport` to allow mocking in unit tests.
- **Unit Tests**: Implemented comprehensive tests in `pkg/integrations/github/list_releases_test.go`, covering setup validation, successful execution, and pagination.
- **Mocking**: Updated tests to use a valid dummy PEM key and mock the GitHub token exchange process required by `ghinstallation`.

### 3. Interface Compliance
- Ensured `ListReleases` implements the latest `core.Component` interface, including the `ExampleOutput()` method (via `pkg/integrations/github/example.go`).
- Verified all other GitHub components and triggers comply with the updated interfaces (`ExampleOutput` for components, `ExampleData` for triggers).

### 4. Architectural Alignment (#2838)
- Verified that the component uses `ctx.Integration.ID()` where appropriate (internal IDs) while maintaining the use of GitHub's numeric `InstallationID` for API authentication.
- Confirmed that recent changes to `SyncContext` and `IntegrationCleanupContext` (removal of `InstallationID` field) are respected.

## Test Results
All GitHub integration tests passed successfully:
```
ok      github.com/superplanehq/superplane/pkg/integrations/github      0.379s
```

## Files Modified/Created
- `pkg/integrations/github/list_releases.go` (Created)
- `pkg/integrations/github/list_releases_test.go` (Created)
- `pkg/integrations/github/example_output_list_releases.json` (Created)
- `web_src/src/pages/workflowv2/mappers/github/list_releases.ts` (Created)
- `pkg/integrations/github/client.go` (Modified)
- `pkg/integrations/github/github.go` (Modified)
- `pkg/integrations/github/example.go` (Modified)
- `web_src/src/pages/workflowv2/mappers/github/index.ts` (Modified)
