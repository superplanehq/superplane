# Building an Integration

High-level steps to add a new integration to SuperPlane.

[Watch a video that explains the process](http://www.youtube.com/watch?v=uWpsHBl8g0Q)

## 1. Pick an integration and claim a ticket

Choose the integration you want to build and **claim the existing issue by commenting on it** so we know you’re working on it.

See: [Integrations Board](https://github.com/orgs/superplanehq/projects/2/views/19).

## 2. Research the connection method

Research how SuperPlane should connect to the service:

- **Auth**: API key, OAuth, or other (where users get credentials, how they’re passed).
- **API**: REST/GraphQL endpoints, rate limits, webhooks vs polling for triggers.
- **Constraints**: Any limitations or quirks that affect design.

Document your findings in the ticket or in the PR description.

## 3. Build the integration

- **Backend**: Implement in [pkg/integrations](https://github.com/superplanehq/superplane/tree/main/pkg/integrations).
- **Frontend**: Add mappers in [web_src/src/pages/workflowv2/mappers](https://github.com/superplanehq/superplane/tree/main/web_src/src/pages/workflowv2/mappers).
- **Docs**: Write docs in the integration package. Generate with `make gen.components.docs`.
- **Tests**: Add unit tests in `pkg/integrations/<name>/`.

Keep the same structure and patterns as other integrations. avoid changing core engine or unrelated code.

More info:
- [Integration Development Guide](integrations.md)
- [Component implementations](component-implementations.md)

## 4. Open a PR and follow the PR guide

Open a pull request and follow **[Opening PRs for Integrations](integration-prs.md)** for:

- PR title and description (including issue link and video demo)
- Backend and frontend expectations
- CI, BugBot, and DCO (signed-off commits)
