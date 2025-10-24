## Basic workflow - no blueprints

```
trigger -> http1 -> http2 -> http3
```

### 1 - New data is emitted by the trigger

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA STATE
ev1 wf1   trigger1 default {}   pending
```

### 2 - PendingWorkflowEventsWorker

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA STATE
ev1 wf1   trigger1 default {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT_SOURCE STATE
ex1 wf1   http1   ev1        ev1          pending
```

### 2 - http1 finishes

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA   STATE
ev1 wf1   trigger1 default {}     routed
ev2 wf1   http1    default {a: 1} pending

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT_SOURCE STATE
ex1 wf1   http1   ev1        ev1          finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA   STATE
ev1 wf1   trigger1 default {}     routed
ev2 wf1   http1    default {a: 1} routed

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT_SOURCE STATE
ex1 wf1   http1   ev1        ev1          finished
ex1 wf1   http2   ev1        ev2          pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA   STATE
ev1 wf1   trigger1 default {}     routed
ev2 wf1   http1    default {a: 1} routed
ev2 wf1   http2    default {b: 2} pending

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT_SOURCE STATE
ex1 wf1   http1   ev1        ev1          finished
ex1 wf1   http2   ev1        ev2          finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  CHANNEL DATA   STATE
ev1 wf1   trigger1 default {}     routed
ev2 wf1   http1    default {a: 1} routed
ev2 wf1   http2    default {b: 2} routed

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT_SOURCE STATE
ex1 wf1   http1   ev1        ev1          finished
ex1 wf1   http2   ev1        ev2          finished
ex1 wf1   http3   ev1        ev3          pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA   STATE
ev1 wf1   trigger1 -            default {}     routed
ev2 wf1   http1    ex1          default {a: 1} routed
ev3 wf1   http2    ex2          default {b: 2} routed
ev4 wf1   http3    ex3          default {b: 2} routed

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT PREVIOUS_EXECUTION STATE
ex1 wf1   http1   ev1        ev1   -                  finished
ex2 wf1   http2   ev1        ev2   ex1                finished
ex3 wf1   http3   ev1        ev3   ex2                finished
```

---

## Workflow with blueprint

The workflow is:

```
http1 -> blueprint1 -> http2
```

The blueprint is:

```
if1 -- true -> http1 --- ok --->
    - false -> http2 - not_ok ->
```

### Running through scenario

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1 -            default {}   pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1 -            default {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT PREVIOUS STATE
ex1 wf1   http1   ev1        ev1   -        pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1 -            default {}   routed
ev2 wf1   http1    ex1          default {}   pending

--- workflow_node_executions
ID  WF_ID NODE_ID ROOT_EVENT INPUT PREVIOUS STATE
ex1 wf1   http1   ev1        ev1   -        finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1 -            default {}   routed
ev2 wf1   http1    ex1          default {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID    ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1      ev1        ev1   -        -      finished
ex2 wf1   blueprint1 ev1        ev1   ex1      -      pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID  EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1 -            default {}   routed
ev2 wf1   http1    ex1          default {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID        ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1          ev1        ev1   -        -      finished
ex2 wf1   blueprint1     ev1        ev1   ex1      -      started
ex3 wf1   blueprint1:if1 ev1        ev1   ex2      ex2    pending
```

Then:


```sql
--- workflow_events
ID  WF_ID NODE_ID        EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1       -            default {}   routed
ev2 wf1   http1          ex1          default {}   routed
ev3 wf1   blueprint1:if1 ex3          true    {}   pending

--- workflow_node_executions
ID  WF_ID NODE_ID        ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1          ev1        ev1   -        -      finished
ex2 wf1   blueprint1     ev1        ev1   ex1      -      started
ex3 wf1   blueprint1:if1 ev1        ev1   ex2      ex2    finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID        EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1       -            default {}   routed
ev2 wf1   http1          ex1          default {}   routed
ev3 wf1   blueprint1:if1 ex3          true    {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID          ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1            ev1        ev1   -        -      finished
ex2 wf1   blueprint1       ev1        ev2   ex1      -      started
ex3 wf1   blueprint1:if1   ev1        ev1   ex2      ex2    finished
ex3 wf1   blueprint1:http1 ev1        ev3   ex3      ex2    pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID          EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1         -            default {}   routed
ev2 wf1   http1            ex1          default {}   routed
ev3 wf1   blueprint1:if1   ex3          true    {}   routed
ev4 wf1   blueprint1:http1 ex4          default {}   pending

--- workflow_node_executions
ID  WF_ID NODE_ID          ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1            ev1        ev1   -        -      finished
ex2 wf1   blueprint1       ev1        ev2   ex1      -      started
ex3 wf1   blueprint1:if1   ev1        ev1   ex2      ex2    finished
ex4 wf1   blueprint1:http1 ev1        ev3   ex3      ex2    finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID          EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1         -            default {}   routed
ev2 wf1   http1            ex1          default {}   routed
ev3 wf1   blueprint1:if1   ex3          true    {}   routed
ev4 wf1   blueprint1:http1 ex4          default {}   routed
ev5 wf1   blueprint1       ex2          ok      {}   pending

--- workflow_node_executions
ID  WF_ID NODE_ID          ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1            ev1        ev1   -        -      finished
ex2 wf1   blueprint1       ev1        ev2   ex1      -      finished
ex3 wf1   blueprint1:if1   ev1        ev1   ex2      ex2    finished
ex4 wf1   blueprint1:http1 ev1        ev3   ex3      ex2    finished
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID          EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1         -            default {}   routed
ev2 wf1   http1            ex1          default {}   routed
ev3 wf1   blueprint1:if1   ex3          true    {}   routed
ev4 wf1   blueprint1:http1 ex4          default {}   routed
ev5 wf1   blueprint1       ex2          ok      {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID          ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1            ev1        ev1   -        -      finished
ex2 wf1   blueprint1       ev1        ev2   ex1      -      finished
ex3 wf1   blueprint1:if1   ev1        ev1   ex2      ex2    finished
ex4 wf1   blueprint1:http1 ev1        ev3   ex3      ex2    finished
ex5 wf1   http2            ev1        ev5   ex2      -      pending
```

Then:

```sql
--- workflow_events
ID  WF_ID NODE_ID          EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1         -            default {}   routed
ev2 wf1   http1            ex1          default {}   routed
ev3 wf1   blueprint1:if1   ex3          true    {}   routed
ev4 wf1   blueprint1:http1 ex4          default {}   routed
ev5 wf1   blueprint1       ex2          ok      {}   routed
ev6 wf1   http2            ex5          default {}   pending

--- workflow_node_executions
ID  WF_ID NODE_ID          ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1            ev1        ev1   -        -      finished
ex2 wf1   blueprint1       ev1        ev2   ex1      -      finished
ex3 wf1   blueprint1:if1   ev1        ev1   ex2      ex2    finished
ex4 wf1   blueprint1:http1 ev1        ev3   ex3      ex2    finished
ex5 wf1   http2            ev1        ev5   ex2      -      finished
```

Lastly:

```sql
--- workflow_events
ID  WF_ID NODE_ID          EXECUTION_ID CHANNEL DATA STATE
ev1 wf1   trigger1         -            default {}   routed
ev2 wf1   http1            ex1          default {}   routed
ev3 wf1   blueprint1:if1   ex3          true    {}   routed
ev4 wf1   blueprint1:http1 ex4          default {}   routed
ev5 wf1   blueprint1       ex2          ok      {}   routed
ev6 wf1   http2            ex5          default {}   routed

--- workflow_node_executions
ID  WF_ID NODE_ID    ROOT_EVENT INPUT PREVIOUS PARENT STATE
ex1 wf1   http1      ev1        ev1   -        -      finished
ex2 wf1   blueprint1 ev1        ev2   ex1      -      finished
ex3 wf1   if1        ev1        ev2   ex2      ex2    finished
ex4 wf1   http1      ev1        ev3   ex3      ex2    finished
ex5 wf1   http2      ev1        ev5   ex4      -      finished
```

## How to solve the N+1 problem when fetching executions with input and output

--- 1. Fetch execution records
select * from workflow_node_executions where workflow_id = wf1 and node_id = http1;

--- 2. Fetch input records
select * from workflow_events where id IN (ev1, ev2, ev3);

--- 3. Fetch output records
select * from workflow_events where execution_id IN (ex1, ex2, ex3)

--- Build the API response using the data above. Not great, but not a N+1 problem anymore

## APIs

GET /components
GET /components/{name}
GET /components/{name}/actions

GET    /blueprints
POST   /blueprints
GET    /blueprints/{id}
PUT    /blueprints/{id}
DELETE /blueprints/{id}

GET    /workflows
POST   /workflows
GET    /workflows/{id}
PUT    /workflows/{id}
DELETE /workflows/{id}


GET   /workflows/{workflow_id}/nodes/{node_id}/executions                      - List all executions for a node
POST  /workflows/{workflow_id}/nodes/{node_id}/events                          - Manually generate an output event for a node
GET   /workflows/{workflow_id}/executions/{execution_id}/children              - List all child executions for a parent execution
POST  /workflows/{workflow_id}/executions/{execution_id}/actions/{action_name} - Invoke action on an execution
GET   /workflows/{workflow_id}/events                                          - List all root events for a workflow
GET   /workflows/{workflow_id}/events/{event_id}/executions                    - List all executions for a root event

## Output channels

```yaml
#
# Single output channel
#
outputChannels:
  - name: default
    nodeId: noop-noop-123
    channel: default

edges:
  - sourceId: filter-filter-q8jpss
    targetId: http-http-rt6ym1
    channel: default
  - sourceId: filter-filter-q8jpss
    targetId: http-http-t8irns
    channel: default
  - sourceId: http-http-t8irns
    targetId: noop-noop-123
    channel: default
  - sourceId: http-http-rt6ym1
    targetId: noop-noop-123
    channel: default

nodes:
  - id: filter-filter-q8jpss
    name: filter
    type: TYPE_COMPONENT
    configuration:
      expression: true
    component:
      name: filter
  - id: http-http-t8irns
    name: http
    type: TYPE_COMPONENT
    configuration:
      method: POST
      url: https://rbaskets.in/dhqfuva
    component:
      name: http
  - id: http-http-rt6ym1
    name: http
    type: TYPE_COMPONENT
    configuration:
      method: POST
      url: https://rbaskets.in/qe69jd4
    component:
      name: http
  - id: noop-noop-123
    name: noop
    type: TYPE_COMPONENT
    configuration: {}
    component:
      name: noop

#
# Multiple output channels
#
outputChannels:
  - name: up
    nodeId: http-http-rt6ym1
    nodeOutputChannel: default
  - name: down
    nodeId: http-http-t8irns
    nodeOutputChannel: default

edges:
  - sourceId: filter-filter-q8jpss
    targetId: http-http-rt6ym1
    channel: default
  - sourceId: filter-filter-q8jpss
    targetId: http-http-t8irns
    channel: default

nodes:
  - id: filter-filter-q8jpss
    name: filter
    type: TYPE_COMPONENT
    configuration:
      expression: true
    component:
      name: filter
  - id: http-http-t8irns
    name: http
    type: TYPE_COMPONENT
    configuration:
      method: POST
      url: https://rbaskets.in/dhqfuva
    component:
      name: http
  - id: http-http-rt6ym1
    name: http
    type: TYPE_COMPONENT
    configuration:
      method: POST
      url: https://rbaskets.in/qe69jd4
    component:
      name: http
```

## Triggers

The difference between triggers and components is that triggers cannot be connected to.

You can emit an event for any node in the workflow with:
  /api/v1/workflows/{workflow_id}/nodes/{node_id}/events

Trigger nodes may also have another endpoint, which uses different authentication than the one used by the API:
  /api/v1/webhooks/{webhook_id}

The handler for that endpoint will find the webhook and its handlers, and invoke the actions specified on them.



How do I give the plain key back to the user?
When I update the schedule configuration, the next_trigger should be updated too.

I'm thinking of going back to the Setup() method instead of the Start() one.

We also need a way to reset the secret for webhook triggers

scheduled trigger

1. PUT /workflows/{id} with trigger node with configuration
2. WorkflowNode record in pending state is created
3. Trigger.Init() is called, populating the node metadata with next_trigger
4. WorkflowNodeProvisioner picks pending workflow_node record
5. WorkflowNodeProvisioner calls Trigger.Setup() - nothing to be done
6. workflow_node record is updated to ready state

webhook trigger

1. PUT /workflows/{id} with trigger node with configuration
2. WorkflowNode record in pending state is created
3. Trigger.Init() is called, populating the node metadata with webhook ID, and returning the key
4. WorkflowNodeProvisioner picks pending workflow_node record
5. WorkflowNodeProvisioner calls Trigger.Setup() - nothing to be done
6. workflow_node record is updated to ready state

github trigger

1. PUT /workflows/{id} with trigger node with configuration
2. WorkflowNode record in pending state is created
3. Trigger.Init() is called, populating the node metadata with webhook ID
4. WorkflowNodeProvisioner picks pending workflow_node record
5. WorkflowNodeProvisioner calls Trigger.Setup() - webhook is created in github using values in webhook
6. workflow_node record is updated to ready state

a workflow node has none or one webhook
a webhook can be shared between multiple workflow nodes
each webhook has specific configuration


workflow_nodes:
```
WF_ID NODE_ID  WEBHOOK METADATA              CONFIGURATION
wf1   node1    hook1   {"repository": {...}} {"integration": "integration1", "repository": "repository1", "events": ["pull_request"]}
wf1   node2    hook2   {}                    {"integration": "integration1", "repository": "repository1", "events": ["push"]}
wf1   node3    hook3   {}                    {"integration": "integration1", "repository": "repository1"}
wf1   node4    hook3   {}                    {"integration": "integration1", "repository": "repository1"}
```

webhooks:
```
ID        SECRET     STATE    CONFIGURATION                INTEGRATION_ID EXTERNAL_RESOURCE_ID
webhook1  secret1    ready    {"events": ["pull_request"]} integration1   repository1
webhook2  secret2    pending  {"events": ["push"]}         integration1   repository1
webhook3  secret3    ready    {"events": ["workflow_run"]} integration1   repository1
```

## Components can have dynamic configuration, output channels, ...

- The OutputChannels() method of the switch component returns a list of output channels based on the configuration.
- The Actions() exposed by a component could also change depending on the configuration. Not only that, but the parameters for the actions might also change depending on the configuration.

Right now, these things are static. We use the /components/{name} and /triggers/{name} endpoint to get the configuration, output channels, and actions.

But, it seems like we should only use that endpoint for the configuration, because the configuration is always static. The output channels and actions could change depending on the configuration, so we should probably expose them differently.

Maybe in:
  /workflows/{workflow_id}/nodes/{node_id}/output_channels
  /workflows/{workflow_id}/nodes/{node_id}/actions

Or maybe that information is returned as part of the WorkflowNode information given when describing the workflow, in the /workflows/{workflow_id} endpoint.