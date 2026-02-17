# LaunchDarkly Integration

Production-ready SuperPlane integration for LaunchDarkly feature flags.

## Included components

- **Trigger:** On Feature Flag Change
- **Action:** Get Feature Flag
- **Action:** Delete Feature Flag

## Authentication

Use a LaunchDarkly API access token in connection settings.

## Setup

1. Install dependencies.
2. Configure connection with API token.
3. Use actions/triggers in your workflow.

## Scripts

- `npm run lint` - lint source code
- `npm run typecheck` - TypeScript validation
- `npm run test` - run unit tests
- `npm run test:coverage` - run tests with coverage

## Webhook trigger setup

In LaunchDarkly, configure a webhook endpoint to point to your SuperPlane trigger URL.

## Notes

- `Delete Feature Flag` is destructive.
- `Get Feature Flag` handles `404` as non-fatal (`found: false`).
- Trigger supports optional event filtering via `eventTypes`.
