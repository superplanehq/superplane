# The bottleneck

SuperPlane has the hard part already: safe, durable, auditable execution for
agent-driven work, with the same RBAC on every path whether a human or an agent
is acting. That answers whether an agent can touch production safely.

The next constraint is different. Once a team runs many agents opening many
changes a day, the question stops being "can the agent do it" and becomes "can a
human approve it fast enough to matter." The approval gate, the thing that makes
SuperPlane safe today, becomes the thing everything waits behind tomorrow. The
reviewer turns into the throughput limit, and the agents are not what is waiting.

This is visible in the tracker already:

- [#6405](https://github.com/superplanehq/superplane/issues/6405): parked approval
  runs pile up.
- [#1614](https://github.com/superplanehq/superplane/issues/1614): a request for
  batch queue management and reprioritization of queued items.

Both treat the symptom by making the queue easier to manage. The cause is that the
safe majority is in the human queue at all. You do not reprioritize a queue that
should not exist. You stop filling it with changes no human needed to see.

Approval should be exception-based. A person should see the changes that carry
risk, and only those.
