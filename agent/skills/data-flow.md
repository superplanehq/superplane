---
description: Runs, run items, message chain, payloads, output channels—how events move on a canvas.
---

# Data flow

SuperPlane is an event-driven workflow engine. Every node on the canvas emits a payload, and other nodes subscribe to these events to create workflows.

## How it works

When an external event occurs (for example a GitHub push), it triggers a node on the canvas. That node processes the event, emits a payload, and downstream nodes that subscribe receive the data and continue the chain.

Each node:

1. **Receives** an event from its subscribed sources
2. **Processes** the event (executes an action, transforms data, etc.)
3. **Emits** a payload for downstream nodes

As the workflow executes, payloads from each node accumulate into a **message chain**. Any node can access data from any upstream node in this chain using expressions.

## Runs and run items

### Run item

A **run item** is a single execution within a single node:

- For **trigger** nodes: one received event (e.g. a GitHub push event)
- For **action** nodes: one execution (e.g. running a GitHub workflow)

Each run item produces a payload that downstream nodes can access.

### Run

A **run** is a collection of run items and the dependencies between them—a complete workflow execution from start to finish.

- Starts with a **root event** (usually from a trigger node)
- Grows as each node adds its run item to the chain
- Tracks full execution history and data flow

## The message chain

As a run executes, each node’s output is added to the **message chain**, accessible via **`$`**—like a message bus of outputs visible to the current node.

Example workflow: GitHub onPush → Filter → Deployment. When it runs, each node adds its output to `$`:

```json
{
  "GitHub onPush": { "ref": "refs/heads/main", "commit": "abc123" },
  "Filter": { "passed": true },
  "Deployment": { "status": "success", "url": "https://app.example.com" }
}
```

From the Deployment node you can read upstream data, e.g. `$['GitHub onPush'].ref`, `$['Filter'].passed`.

Use **`root()`** for the original event that started the run, and **`previous()`** for the immediate upstream node. Load the **expressions** skill for more detail.

## Exploring runs on the canvas

The canvas is a live view; multiple runs can run at once.

- **Node status** — quick view of the current or most recent run item for that node
- **Run history** — sidebar on a node: past executions/events through that node, with results
- **Run chain** — from a history item: all run items in that run across nodes
- **Inspecting run items** — select items in the chain to see payload details

## Payloads

Every node emits a **payload** (JSON) from its execution.

### Trigger components

Listen to external resources; payload is the event data (webhooks, integrations). Examples: GitHub onPush, onRelease, Slack onAppMention.

### Action components

Subscribe upstream, perform work, emit results as payload. Examples: GitHub runWorkflow, Slack sendMessage, HTTP request.

### Output channels

Nodes can emit on **one or more named channels** to route by outcome.

Examples:

| Component | Channels | Use |
|-----------|----------|-----|
| GitHub runWorkflow | passed, failed | Success vs failure |
| Approval | approved, rejected | Decision |
| Merge | success, stopped, timeout | Merge outcome |
| Dash0 listIssues | clear, degraded, critical | Severity |
| PagerDuty listIncidents | clear, low, high | Urgency |

Subscribe downstream to a specific channel to branch the workflow.
