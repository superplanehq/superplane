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

List all executions for a node
  GET /workflows/{workflow_id}/nodes/{node_id}/executions

List all child executions for a parent execution
  GET /workflows/{workflow_id}/executions/{execution_id}/children

Invoke action on an execution
  POST /workflows/{workflow_id}/executions/{execution_id}/actions/{action_name}

List all root events for a workflow - workflow_events records with execution_id set
  GET /workflows/{workflow_id}/events

List all executions for a root event
  GET /workflows/{workflow_id}/events/{event_id}/executions

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