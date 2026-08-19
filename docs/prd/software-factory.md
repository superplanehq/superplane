# Software Factory

> Status: Draft
>
> This draft captures the agreed product foundation and domain model. Detailed
> lifecycle states, automation behavior, policies, and metrics will be
> specified as the remaining product questions are resolved.

## Overview

Software Factory is a new first-class resource in SuperPlane for managing
automated software work. It exists alongside SuperPlane Apps and has its own
identity, creation flow, data model, and dedicated product experience.

Existing SuperPlane Apps remain unchanged. Introducing Software Factory must
not alter their behavior, creation flow, pages, or underlying semantics.

A Factory owns multiple Canvases, which are presented as **Automations** in the
Factory context. Canvas ownership becomes polymorphic: a Canvas belongs either
to a SuperPlane App or to a Software Factory. A Factory also owns Work Orders.

A Work Order is the system of record for the implementation of one piece of
software work, from its initial description through the Factory's Automation
events to an explicitly declared success or failure. A successful Work Order
will typically have a pull request attached from GitHub or Bitbucket.

## Problem Statement

SuperPlane can automate software delivery through Apps, but an App is a
general-purpose automation construct. It does not provide a purpose-built
operational model for teams delegating software work and tracking it through
implementation.

Users need a dedicated place where they can:

- Create and identify a Software Factory as a distinct product resource.
- Understand how the Factory is performing at a glance.
- See which work requires human attention.
- Track each piece of work through a durable, chronological record.
- Reach the source-control pull request produced by that work.

## Product Decisions

The following decisions are confirmed for this draft:

1. Software Factory is a first-class citizen within SuperPlane.
2. Software Factory is separate from the existing SuperPlane App experience.
3. Existing Apps remain as they are and are not modified by this initiative.
4. A Canvas belongs either to a SuperPlane App or to a Software Factory.
5. A Factory owns multiple Canvases, called Automations in the Factory context.
6. A Factory has a name and may have a description.
7. A Factory can be created through an experience comparable to creating an
   App.
8. Organization members with a role above viewer can create Factories and
   manually create Work Orders.
9. A Factory has its own full, dedicated page.
10. The Factory overview presents high-level operational information, including
    throughput, success rate, and Work Orders that require attention.
11. Work Order is a new primary entity owned by a Factory.
12. A Work Order has a title and description.
13. A Work Order is the system of record for implementing one piece of work.
14. A Work Order can be created manually or by an Automation.
15. A Work Order begins as `draft` and becomes `ready` when approved.
16. An Automation can listen for newly ready Work Orders and begin processing
    them.
17. A Work Order can flow through multiple Automations.
18. A Factory and its Automations are not restricted to one repository.
19. Automations create Work Orders through a dedicated component.
20. The intended output of a Work Order is a pull request on GitHub or
    Bitbucket.
21. An Automation attaches a pull request to a Work Order through a component.
22. The initial product experience assumes one pull request per Work Order, but
    the domain model must allow multiple pull-request attachments.
23. An Automation explicitly marks a Work Order successful or unsuccessful
    through a component.
24. Work Order history is append-only. Approvals, retries, reopens, reruns, and
    outcomes add events rather than rewriting prior events.
25. Every Work Order has a dedicated page.
26. The Work Order page begins with the work description and then presents the
    sequence of Factory automation events in chronological order.
27. The chronology runs from the oldest event at the top to the newest event at
    the bottom.
28. Conversations, decisions, approvals, and steering instructions are durable
    Work Order Events.
29. The Factory page has a header with the Factory name and optional
    description.
30. The Factory page has four primary tabs: **Overview**, **Work Orders**,
    **Automations**, and **Velocity**.
31. Overview is the default tab and puts Work Orders requiring attention
    before summary metrics and other activity.
32. Work Orders are grouped into **Needs attention**, **Running**,
    **Recently done**, and **Unsuccessful** sections.
33. Automations are presented as a finite list within the Factory.
34. Opening an Automation opens the existing Canvas editing experience for
    that Factory-owned Canvas.
35. Velocity defaults to the last 14 days.
36. Velocity is filtered by repository, not by Automation.
37. Velocity presents the same delivery indicators for the Team total,
    human-authored work, and Factory-authored work without framing the
    comparison as a competition.
38. Human-authored velocity is aggregated from pull requests associated with
    organization members' Git accounts. The main Factory page does not show an
    unbounded per-person breakdown.
39. Tracked Factory cost includes model tokens and execution compute.
    Third-party service charges are excluded.

## Goals

1. Make Software Factory recognizable and manageable as its own SuperPlane
   resource.
2. Let users create a Factory without changing how they create or use Apps.
3. Let each Factory own multiple Canvas-backed Automations.
4. Let Automations create and operate on Work Orders across repositories.
5. Give each Factory a dedicated operational overview.
6. Establish Work Order as the durable record for a piece of delegated
   software work.
7. Make the history and current position of a Work Order understandable from
   one chronological page.
8. Preserve human collaboration and intervention as part of the Work Order's
   durable history.
9. Support retries and reopens without destroying or rewriting earlier
   attempts.
10. Connect completed Factory work to one or more GitHub or Bitbucket pull
    requests.

## Non-Goals

- Replacing, redesigning, or changing SuperPlane Apps.
- Treating a Factory as merely a renamed App in the user interface.
- Replacing GitHub or Bitbucket as the system of record for source code,
  review, or merge history.
- Restricting a Factory to one source-control repository.
- Defining the complete lifecycle state machine, approval policy, or steering
  behavior beyond the initial requirements in this draft.
- Finalizing formulas for throughput, success rate, or attention.
- Attributing salary, human labor cost, or third-party service charges in the
  Factory cost metric.

## Primary Users

- **Engineering leaders** who need to understand Factory output, reliability,
  and items requiring intervention.
- **Software engineers** who create or follow Work Orders and review the pull
  requests they produce.
- **Factory operators** who configure a Factory and keep its work moving.

## Core Entities

### Software Factory

A Software Factory is an organization-level product resource that owns Work
Orders and multiple Canvas-backed Automations.

Confirmed user-facing fields:

- `name`
- `description` (optional)

The Factory also needs a stable identifier and ownership metadata as a
persisted resource. A Factory is not bound to a single repository. Provider
connections, status, and other setup fields remain to be defined.

### Automation

An Automation is the Factory-facing name for a Canvas owned by a Software
Factory. It uses the Canvas execution model while appearing within the
Factory's dedicated product experience.

Confirmed relationships and behavior:

- A Factory owns multiple Automations.
- Each Automation is backed by a Canvas whose owner is the Factory.
- A Factory-owned Canvas is presented as an Automation, not as an App.
- Different Automations in one Factory can interact with different
  repositories.
- An Automation can create a Work Order through a dedicated component.
- Automations can operate on an existing Work Order and append to its durable
  chronology.
- A source-control component can create a branch and pull request, then attach
  the pull request to the active Work Order.
- A Work Order component can mark the active Work Order successful or
  unsuccessful.

The mechanism for carrying Work Order context between Automations remains to
be defined.

### Work Order

A Work Order represents one requested software change and the complete record
of the Factory's attempt to implement it.

Confirmed user-facing fields:

- `title`
- `description`

Confirmed relationships:

- A Work Order belongs to a Software Factory.
- A Work Order can be created manually by an authorized organization member.
- A Work Order can be created by an Automation, including in response to an
  external event such as a GitHub issue assignment.
- A Work Order records events produced as it moves through Factory
  Automations.
- An Automation component can attach one or more GitHub or Bitbucket pull
  requests to a Work Order.
- An Automation component explicitly marks the Work Order successful or
  unsuccessful.

The initial product experience assumes one primary pull request per Work Order.
The domain relationship must nevertheless support multiple pull-request
attachments so one Work Order can span separate repositories when necessary.

Success and failure are explicit Work Order operations; they are not inferred
solely from a pull request's state.

### Work Order Lifecycle

The first confirmed lifecycle states are:

- `draft`: The Work Order exists but has not been approved for Factory
  execution.
- `ready`: The Work Order has been approved and can be picked up by an
  Automation.
- `successful`: An Automation explicitly recorded a successful outcome.
- `unsuccessful`: An Automation explicitly recorded an unsuccessful outcome.

Approval appends a Work Order Event and moves the current state from `draft` to
`ready`. A Factory Automation can listen for newly ready Work Orders and use
that event as its trigger.

A Work Order may flow through multiple Automations. Outputs from one Automation
must be able to hand the Work Order to another Automation while preserving the
same Work Order identity and chronology. The exact orchestration model is not
yet decided. It may use a coordinating Automation, direct handoffs between
Automations, Work Order events, or another piping mechanism.

Successful and unsuccessful Work Orders can be retried or reopened. A retry,
reopen, or rerun does not delete or replace the earlier attempt. It appends a
new event and starts additional Automation activity against the same Work
Order.

The current state may be stored or projected for efficient access, but its
history must always be explainable from the append-only event record. Whether
title and description can be edited while a Work Order is still a draft
remains open.

### Work Order Event

A Work Order Event represents something that happened while the Factory was
processing a Work Order. Events form the chronology displayed on the Work
Order page.

At minimum, an event will need:

- A stable identity.
- A timestamp.
- A human-readable description of what happened.
- A relationship to the Work Order.

The durable chronology includes:

- Automation progress and outcomes.
- Human and agent conversations.
- Decisions.
- Approval requests and responses.
- Steering instructions.
- Pull-request attachment.
- Explicit success or failure.
- Retry, reopen, and rerun requests and outcomes.

Actor, event type, Automation/run references, artifacts, lower-level logs, and
visibility rules remain to be defined.

## Information Architecture

Software Factory must have dedicated routes and pages rather than reusing the
existing App detail experience.

Conceptually:

```text
Software Factory
|-- Overview
|   |-- Needs attention
|   |-- Factory summary metrics
|   `-- Current activity
|-- Work Orders
|   |-- Needs attention
|   |-- Running
|   |-- Recently done
|   `-- Unsuccessful
|-- Automations
|   `-- Canvas-backed Automation
|-- Velocity
|   |-- Repository filter
|   |-- Team, human-authored, and Factory-authored indicators
|   `-- Tracked token and compute cost
`-- Work Order
|   |-- Description
|   `-- Chronological automation events
```

The exact placement of Factories in organization navigation and resource
lists remains open.

## Factory Creation

Users must be able to create a Factory through a first-class flow comparable
to App creation.

The initial creation contract includes:

- A required Factory name.
- An optional Factory description.
- An organization member with a role above viewer.

The following creation details are not yet specified:

- Whether GitHub or Bitbucket must be connected during creation.
- Which optional AI-assisted setup, wizard, or Automation templates are
  available.

A repository is not an ownership constraint of the Factory. Its Automations
may work with different repositories.

The current product direction is to create a blank Factory with zero
Automations. After creation, users can add Automations with AI assistance, a
guided wizard, templates, or manual Canvas authoring. The exact onboarding
experience remains to be designed.

## Factory Overview

Every Factory has a dedicated page with the Factory name, optional
description, operational status, primary actions, and four tabs:

- **Overview**
- **Work Orders**
- **Automations**
- **Velocity**

Overview is the default tab. Its purpose is operational orientation, not
detailed analysis. Work Orders that require a human decision, approval,
clarification, or intervention appear first so the user can immediately
understand what is blocked.

The overview must communicate at least:

- Factory throughput.
- Factory success rate.
- Work Orders that require attention.
- Active Work Orders.
- Tracked Factory execution cost.

The Overview may summarize running, recently completed, and unsuccessful work,
but the complete state-grouped lists belong on the Work Orders tab. Success
rate can be grounded in the explicit successful and unsuccessful outcomes
recorded by Work Order components. The exact denominator, targets, and
drill-down behavior remain to be specified.

### Factory Page Layout

The recommended layout is responsive and uses the available width with
consistent page gutters and a restrained maximum content width of roughly
1,500-1,600 pixels. Operational tables and comparative charts benefit from
horizontal space, while the cap preserves scanning comfort on very wide
displays.

This width recommendation applies to the Factory page and its tabular views.
The Work Order page may use a narrower reading column for chronology, and the
Canvas-backed Automation editor remains full-bleed.

## Work Orders Tab

The Work Orders tab is the complete operational queue for Factory work. It
groups Work Orders into:

1. **Needs attention:** Work paused for a decision, approval, clarification,
   or intervention.
2. **Running:** Work currently moving through Factory Automations.
3. **Recently done:** Recently successful Work Orders and their delivered pull
   requests.
4. **Unsuccessful:** Work Orders explicitly marked unsuccessful and available
   for review, retry, or reopen.

The list is ordered for actionability within each group and lets the user open
the dedicated Work Order page.

## Automations Tab

The Automations tab presents the Factory's Canvas-backed Automations as a
finite operational list. Each row should communicate enough context to select
the correct Automation, including its name, description, trigger, status,
current activity, recent success, and last run.

Opening an Automation uses the existing Canvas view and editing behavior.
Factory context changes the ownership and product terminology, not the core
Canvas authoring experience.

## Velocity Tab

The Velocity tab explains repository delivery performance without treating
human and Factory output as opposing teams.

The default period is the last 14 days. When a Factory operates across
multiple repositories, the primary filter is repository. Automations are not
a velocity filter because several Automations may contribute to one Work Order
and repository outcome.

The same delivery indicators are shown for three cohorts:

- **Team total:** All qualifying repository work in the selected period.
- **Human-authored:** Qualifying pull requests attributed to organization
  members through their Git accounts.
- **Factory-authored:** Qualifying pull requests attached to Factory Work
  Orders.

The main view remains aggregated and does not list every human contributor.
Individual drill-downs may be designed later.

The initial velocity matrix includes:

- Merged pull-request throughput.
- Pull-request cycle time.
- Success rate.
- Tracked execution cost.

The page should also show throughput over time and a clear cost breakdown.
Tracked cost includes:

- Model tokens.
- Execution compute.

Tracked cost excludes third-party service charges. Human labor and salary are
also outside this cost definition. Until a separate attribution model is
defined, human-authored work may show tracked execution cost as unavailable
rather than implying that human work has no cost.

## Work Order Page

Every Work Order has its own dedicated page.

The page has two primary parts:

1. **Work description:** The title and description establish the requested
   outcome and remain visible as the context for the implementation.
2. **Chronology:** An ordered chain of events shows how the Work Order moves
   through Factory automations.

The chronology is the main structural model of the page. It must make it
possible to understand:

- What happened.
- In what order it happened.
- Where the Work Order currently is.
- Whether it needs human attention.
- What pull request was produced, when one exists.

The page uses a vertical timeline. The oldest event appears at the top and new
events are appended at the bottom. Conversations, decisions, approvals, and
steering instructions appear in the same durable chronology as Automation
events.

## Work Order Creation and Automation Flow

A Work Order can enter the Factory manually or through an Automation. It is
created as a draft, then approved to become ready for Automation processing.

An issue-driven flow can work as follows:

1. A source-control trigger receives a new or newly assigned GitHub or
   Bitbucket issue through a webhook.
2. The Automation determines that the issue should be handled by the Factory.
3. The Automation calls a **Create Work Order** component and maps the issue
   payload into the Work Order.
4. The Work Order is approved, appending an approval event and moving it from
   `draft` to `ready`.
5. An Automation listening for newly ready Work Orders starts processing it.
6. The Work Order becomes the durable context as work is piped through one or
   more Automations.
7. Automations append progress, collaboration, decisions, handoffs, and retries
   to the Work Order.
8. A source-control component creates the implementation branch and pull
   request, then attaches the pull request to the Work Order.
9. A Work Order component marks the Work Order successful or unsuccessful.

This example does not constrain all Work Orders to issue-based intake. Other
Automation triggers may create Work Orders through the same component.

## Append-Only History and Retries

The Work Order is an append-only operational record. Existing events are not
edited or removed when plans change, an Automation is retried, or a completed
Work Order is reopened.

Each action appends a new event, including:

- Approval and transition to ready.
- Automation pickup and handoff.
- Retry of an Automation or failed step.
- Reopen of a successful or unsuccessful Work Order.
- New conversations, decisions, approvals, or steering.
- Pull-request attachment.
- A new success or failure outcome.

The page therefore shows every attempt and explains how the current state was
reached. A rerun is additional activity within the same Work Order, not a
replacement for its previous history.

## Source-Control Outcome

The intended output of a successful Work Order is one or more pull requests on
these providers:

- GitHub
- Bitbucket

The Work Order remains the system of record for the Factory's implementation
process. The pull request remains the source-control record for code changes,
review, checks, and merge history.

The initial interface is optimized for one primary pull request. The data model
must allow multiple attachments so a Work Order can produce changes in
separate repositories, such as frontend and backend pull requests.

Pull-request creation and Work Order completion are separate explicit actions:

1. A source-control component creates the branch and pull request.
2. The component attaches the pull request to the active Work Order without
   replacing any existing pull-request attachment.
3. A Work Order component marks the Work Order successful or unsuccessful when
   the Automation determines the outcome.

The exact policy an Automation uses before marking success is configurable
workflow behavior rather than a status inferred globally from pull-request
creation, review, checks, or merge.

## Functional Requirements

### Factory Resource

1. Users can create a Software Factory with a name and optional description.
2. Organization members with a role above viewer can create a Factory.
3. Users can distinguish Factories from Apps throughout the interface.
4. Users can open a Factory's dedicated page.
5. A Factory can own multiple Canvases.
6. A Factory-owned Canvas is presented as an Automation in the Factory
   context.
7. A Canvas belongs either to an App or to a Factory.
8. A newly created Factory can exist with zero Automations.
9. Existing App behavior and pages continue to work without change.

### Automations

1. Users can create multiple Automations within a Factory.
2. Automations within one Factory can interact with different repositories.
3. An Automation can call a component to create a Work Order.
4. An Automation can listen for Work Orders that transition to `ready`.
5. The same Work Order can flow through multiple Automations.
6. Automations operating on a Work Order can append durable events to it.
7. A source-control component can attach a GitHub or Bitbucket pull request to
   the active Work Order.
8. A Work Order component can explicitly mark the active Work Order successful
   or unsuccessful.

### Factory Overview

1. Users can see high-level Factory throughput.
2. Users can see high-level Factory success rate.
3. Users can see Work Orders that require their attention.
4. Users can open a Work Order from the Factory overview.
5. Users see the Factory name and optional description above the tab
   navigation.
6. Overview is the default of four tabs: Overview, Work Orders, Automations,
   and Velocity.
7. Work Orders requiring attention appear before summary metrics and other
   activity.

### Work Orders Page

1. Users can see Work Orders grouped into Needs attention, Running, Recently
   done, and Unsuccessful.
2. Users can open any listed Work Order.
3. Users can distinguish work waiting on them from work the Factory is
   actively processing.

### Automations Page

1. Users can see the set of Automations owned by the Factory.
2. Each Automation shows identifying and operational context.
3. Opening an Automation opens its existing Canvas view.

### Velocity

1. Velocity defaults to a 14-day period.
2. Users can filter velocity by repository when the Factory works across
   multiple repositories.
3. Velocity is not filtered by Automation.
4. Users can compare the same delivery indicators for Team total,
   human-authored, and Factory-authored work.
5. Human-authored data is aggregated rather than rendered as an unbounded
   contributor list.
6. Velocity communicates the cohorts neutrally rather than presenting humans
   and the Factory as competitors.
7. Tracked cost includes model tokens and execution compute.
8. Tracked cost excludes third-party service charges and human labor.

### Work Orders

1. The system persists each Work Order as a first-class entity.
2. Each Work Order has a title and description.
3. Each Work Order belongs to a Factory.
4. An organization member with a role above viewer can create a Work Order
   manually.
5. An Automation can create a Work Order through a component.
6. A new Work Order starts in `draft`.
7. Approving a Work Order appends an event and moves it to `ready`.
8. Each Work Order has a stable, dedicated page.
9. The system persists the events associated with a Work Order.
10. The Work Order page displays the oldest event at the top and appends the
    newest event at the bottom.
11. Conversations, decisions, approvals, and steering instructions are
    persisted as Work Order Events.
12. A Work Order supports multiple GitHub or Bitbucket pull-request
    attachments, while the initial interface emphasizes one primary pull
    request.
13. Work Order success or failure is explicitly recorded by an Automation
    component.
14. Retrying, reopening, or rerunning work appends events without changing or
    deleting prior events.

## Preliminary Acceptance Criteria

1. An organization member with a role above viewer can create a Factory
   without creating or modifying an App.
2. A created Factory is persisted and can be reopened through its dedicated
   page.
3. The Factory page shows its name, description, and a high-level overview.
4. The overview has defined places for throughput, success rate, and Work
   Orders requiring attention.
5. A new Factory can be persisted with no Automations.
6. A Factory can later own multiple Canvas-backed Automations.
7. Existing App-owned Canvases continue to behave as they do today.
8. Automations in the same Factory can work with different repositories.
9. An authorized organization member can manually create a Work Order.
10. An Automation can create a Work Order through a dedicated component.
11. A Work Order can be persisted with a title and description under a Factory.
12. A Work Order is initially a draft.
13. Approval appends an event, moves the Work Order to ready, and makes it
    available to a listening Automation.
14. The same Work Order can be processed by multiple Automations without
    changing its identity.
15. Opening a Work Order shows its description before its event history.
16. Work Order Events are displayed oldest-first from top to bottom.
17. Conversations, decisions, approvals, and steering instructions persist in
    the event chronology.
18. A source-control component can attach a GitHub or Bitbucket pull request to
    a Work Order.
19. The Work Order can retain more than one pull-request attachment.
20. A Work Order component can mark the Work Order successful or unsuccessful.
21. A retry, rerun, or reopen appends new events and preserves all earlier
    attempts.
22. Existing App creation and App pages exhibit no behavioral regression.
23. The Factory page has Overview, Work Orders, Automations, and Velocity tabs.
24. The Overview puts Work Orders requiring attention first.
25. The Work Orders tab groups work into Needs attention, Running, Recently
    done, and Unsuccessful.
26. Opening an Automation from the Automations tab opens its Canvas view.
27. Velocity defaults to 14 days and can be filtered by repository.
28. Team total, human-authored, and Factory-authored velocity use the same
    indicators.
29. Factory cost includes token and execution compute cost but excludes
    third-party service charges.

## Assumptions Requiring Confirmation

These are working assumptions, not settled requirements:

1. **Canvas association:** The polymorphic relationship is represented as a
   single Canvas owner association whose owner is either an App or a Factory.
   The exact storage implementation is not prescribed here.
2. **Blank by default:** Factory creation initially produces a Factory with zero
   Automations. AI assistance, a wizard, or templates are optional next steps
   rather than mandatory creation behavior.
3. **Active Work Order context:** Components running on behalf of a Work Order
   receive a stable Work Order identifier so they can append events, attach a
   pull request, and record the outcome.
4. **Durable product events:** The Work Order chronology is persisted
   separately from lower-level Canvas execution logs.
5. **Current-state projection:** The current Work Order state can be stored as
   a projection for efficient queries, while its transitions remain represented
   by append-only events.
6. **Team total:** The Team total cohort combines qualifying human-authored and
   Factory-authored repository work.
7. **Human cost:** Human-authored work shows tracked execution cost as
   unavailable until a meaningful and explicitly scoped attribution model is
   defined.
8. **Factory page width:** The Factory page uses responsive available width
   with a maximum content width near 1,600 pixels. Work Order chronology may
   be narrower and Canvas remains full-bleed.

## Open Questions

### Resource Model

1. Should Factories appear beside Apps in the current resource list, or have a
   separate top-level navigation area?
2. Should blank Factory creation be the default, with AI, wizard, and template
   setup offered afterward, or should users choose an onboarding path during
   creation?
3. Can a Canvas ever move between App and Factory ownership, or is its owner
   immutable?

### Creation and Access

4. Which fields and connections are required during Factory creation beyond
   name and description?
5. Must provider and agent integrations be connected at Factory creation, or
   can they be added when an Automation needs them?
6. Do Factories, Automations, and Work Orders need separate authorization
   permissions?

### Automations and Work Orders

7. What data must the **Create Work Order** component accept beyond title and
   description, such as source issue, repository, labels, or requester?
8. Who or what can approve a draft Work Order: a human member, an Automation,
   or a Factory policy? Every approval still appends an explicit event.
9. What mechanism pipes a Work Order from one Automation into the next: a
   coordinating Automation, direct handoff, emitted Work Order events, or
   another model?
10. Which data and artifacts are passed during an Automation handoff?
11. Which non-terminal Work Order states are required beyond `draft` and
    `ready`?
12. When a completed Work Order is reopened, does it return to `ready` or enter
    a separate state?
13. Is title and description editing allowed while a Work Order is in `draft`,
    and are they immutable after approval?
14. How should cancelled or rejected work be represented when no pull request
    is produced?

### Pull Requests

15. When a Work Order has multiple pull requests, is one designated primary or
    are all attachments peers?
16. Does marking a multi-PR Work Order successful require every attached pull
    request to reach an Automation-defined condition?

### Chronology and Attention

17. Which Automation events belong in the durable chronology versus lower-level
    Canvas execution logs?
18. Are conversations a flat sequence of events, or can they form reply
    threads around a decision or checkpoint?
19. What actions make a Work Order require attention: a question, approval,
    failed Automation, missing permission, merge conflict, or other states?

### Overview and Velocity Metrics

20. Which throughput unit is primary on Overview: completed Work Orders,
    opened pull requests, merged pull requests, or a small set of these?
21. What is the precise success definition for each cohort? Factory work has
    explicit successful and unsuccessful Work Order outcomes, while
    human-authored work needs an equivalent repository-derived rule.
22. Should cancelled or reverted work affect the success-rate denominator, and
    in what time period should the outcome be attributed?
23. Is **Team total** definitively the combined human-authored and
    Factory-authored cohort, including Factory pull requests authored by a
    service Git account?
24. Should human-authored tracked execution cost remain unavailable, or should
    token and compute used by human-triggered automations be attributed to that
    cohort?
25. When a Factory has several repositories, should Velocity initially select
    the most recently active repository, a configured default, or an
    all-repositories aggregate?
