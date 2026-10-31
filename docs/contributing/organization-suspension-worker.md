# Organization suspension cancellation

The regular worker polls pending suspension events every five seconds.
It uses the deployed organization suspension model. No batch coordination
flag is required. Without pending events, it does not cancel work.

Cancellation requests remain durable until workflow finalization, runner
completion, and confirmed agent stopping finish. Reactivation remains blocked
while cancellation is incomplete. Schedules retain their future configuration.

## Process configuration

- Regular workers require the existing database configuration and `RABBITMQ_URL`.
  Keep `START_RUN_FINALIZER=yes` and `START_EXECUTION_TERMINATOR=yes`.
- Runner gateways require their existing database and `RABBITMQ_URL` configuration.
  Control notifications wake gateways. Gateways read durable task cancellation
  requests. Runner completion, log reporting, and cleanup use their existing paths.
- Managed-agent cancellation requires `ANTHROPIC_API_KEY`, `ANTHROPIC_AGENT_ID`,
  and `ANTHROPIC_ENVIRONMENT_ID` on regular workers. The provider uses
  `https://api.anthropic.com/v1`. Missing configuration leaves agent cancellation
  pending. Organizations without active managed agents can still complete.

## Deployment source inspection

Local `docker-compose.dev.yml` passes all three managed-agent keys to the app
process. It also configures the cancellation workers and runner cleanup.

The checked-in worker chart previously lacked a managed-agent Secret reference.
The chart now accepts the existing `anthropic.secretName` configuration.
For managed-agent installations, set this value to the Secret with all three
keys. This prepares chart source. It does not publish a chart or verify a deployment.

The `superplanehq/launchpad` staging and production values reference
`superplane-anthropic` through `anthropic.secretName`. The separate hosted
`superplanehq/helm-chart` repository returned HTTP 404 during inspection.
Its worker Secret wiring remains unverified. Live Secret contents also remain
unverified. Confirm the three keys reach regular workers in managed-agent
installations before suspension events can target those sessions.

Deploy this worker after the model foundation. Deploy the subsequent suspension
API, UI, and admission enforcement after this worker. No companion source
change is prepared in the unavailable hosted chart repository.
