---
description: Canonical definitions for canvas, workflow, node, run, payload, subscription, message chain, and related terms.
---

# Glossary

Core terminology used across SuperPlane.

## Canvas

A **canvas** is the workspace where you design and run workflows. It is a graph of nodes connected by **subscriptions** that define how events flow. A canvas usually represents multiple possible workflows.

## Workflow

A **workflow** is the behavior expressed by a canvas: what should happen when an event occurs, which steps run, and how data moves between steps.

## Node

A **node** is a single step on a canvas. Each node receives an event, performs work, and emits an event to downstream nodes that subscribe to it.

## Component

A **component** is the type of a node (e.g. Webhook, Manual Run, Filter, or a provider-specific action). Components define required configuration and what they emit.

## Trigger

A **trigger** is a component that **starts** a workflow execution—typically external events (webhooks, schedules) or manual runs.

## Action

An **action** is a component that runs **in response to an upstream event**—calling external systems, transforming data, routing, or human-in-the-loop steps.

## Integration

An **integration** connects SuperPlane to an external system (GitHub, Slack, PagerDuty, etc.) and provides triggers and actions you place as nodes.

## Event

An **event** is the unit of work flowing between nodes. It carries the **payload** and is delivered to subscribed downstream nodes.

## Payload

A **payload** is the JSON data for an event or node execution—what you inspect in run history and reference in expressions.

## Output channel (channel)

A **channel** is a **named output** from a node (e.g. `passed` / `failed`, `approved` / `rejected`). Channels route events to different downstream paths by outcome.

## Subscription

A **subscription** connects one node’s output (optionally a specific channel) to another node’s input. The canvas is a graph of subscriptions.

## Run

A **run** is one **end-to-end** workflow execution—from the first trigger through all downstream work it causes.

## Run item

A **run item** is the execution record for **one node** within a run. A run contains many run items.

## Run history

**Run history** is the UI list of past executions for a node or canvas—payloads, timestamps, statuses, errors.

## Message chain

The **message chain** is the accumulated outputs from **upstream** nodes in a run. Downstream nodes use it (via `$` and expressions) to combine data from earlier steps.

## Expression

An **expression** is a small program to read and transform payload data—messages, conditions, routing. Load the **expressions** skill for syntax and examples.

## Service account

A **service account** is a non-human identity for API access from scripts and integrations, governed by RBAC.
