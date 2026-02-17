# Integration Guide

## Components

### Trigger: On Feature Flag Change
Listens for LaunchDarkly webhook events:
- `flag.created`
- `flag.updated`
- `flag.deleted`
- `flag.archived`
- `flag.restored`

### Action: Get Feature Flag
Inputs:
- `projectKey`
- `flagKey`

Output:
- `found`
- `flag` (when found)
- `message`

### Action: Delete Feature Flag
Inputs:
- `projectKey`
- `flagKey`

Output:
- `deleted`
- `message`
- `statusCode`

## Quality gates implemented

- Input validation for all actions
- Explicit error handling for API and network failures
- Unit tests for auth, actions, and trigger
- Lint and TypeScript checks configured
- Coverage threshold configured in Jest
