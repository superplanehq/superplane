# Linear PoC (Issue #2485)

This folder contains a minimal, reviewable PoC for the Linear integration bounty:

- Webhook parsing for `Issue` + `create` events
- Mutation variable builder for Linear `issueCreate`
- Unit tests for both paths

The PoC is intentionally scoped to de-risk the API wiring before adding:

- Full integration registration
- Trigger/action components
- Frontend mappers
- End-to-end setup flow and demo video
