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

## "ConnectionGroup" component

Execute() can:
  get() / set() metadata for the execution
  pass() / fail() the execution

Issue:
- the next item in the queue for the node will not be processed until the current execution finishes

Two ways:

### 1. The processing engine knows it should "wait until all data is collected" and then Execute() is only called once with all the data

This is kinda of what the concept of "input channel" is, but I'm having a hard time reasoning about it from this angle.

I don't like the idea of introducing extra logic into the processing engine. This whole idea of components as first-class citizens was that logic was pushed from the processing engine to the component itself. The processing engine provides capabilities that the component execution can use to do what it needs, and this goes against that, so I'm exploring option 2 first.

### 2. Execute() can receive new events from the queue without completing the current execution

What if, from the Execute(), the execution could also request the next item in the queue?

Maybe it could do that through scheduling a procedure for the processing engine to execute in the future. This procedure can be:
  - ActionCall - invoke an action for the execution
  - FetchNextInQueue - fetch the next event in the queue and forward it to the current execution

I really like this idea of "scheduling work for the processing engine to do in the future"

We would need to change a few things in the current processing engine to support this:

  1. More context needs to be given to the component. In this particular example, the component execution needs to know about the connections, so it can check which connections have emitted something, and which connections haven't. This is easily addressable, since we already have ExecutionContext - just put this information in there too.

  2. An item "in the queue" is already a pending execution. This should be addressable as well, since now the data is stored in WorkflowEvent, so we could have WorkflowNodeQueueItem records for "items in the queue", and now "find next item in queue" means "look for records in workflow_node_queue_items" table. Actually, this is probably better than looking for pending WorkflowNodeExecution records, performance wise.

  3. Currently, blueprint nodes have no queue, so FetchNextInQueue wouldn't work for blueprint nodes. The solution here would probably be to use WorkflowNode records for all nodes - workflow and blueprint nodes.

  4. We would need this new "scheduled component request worker". We could probably do that through a `scheduled_component_requests` table. That table has a column `type` - `action-call` and `fetch-next-in-queue` for now, but we could add more types as we see needs for it going forward. It also has state and a `run_at` column which tells the worker when to run the request, so the worker can only load records that need to be executed.

I think I like this
  [1] is good idea anyway, since it gives more power to components
  [2] improves current processing engine performance
  [3] still unclear, but I'm leaning towards liking the idea of having parent/child relationship in the workflow nodes, the same way we have in the workflow_node_executions. We could probably take this idea even further and also model the left/right relationship between nodes using that table. Now, we have `workflow_nodes` table, but edges are still recorded in the `workflows` table, which is a bit weird.
  [4] seems like a good idea because we can re-use this mechanism for several different things - scheduled and polling triggers, polling Semaphore workflows created by Semaphore component, ...

### Conclusion

Approach [2] seems to be the way to go:
- We avoid introducing extra complexity with new "input channel" concept
- We push even more logic into the component, which is what we wanted
- We extend the current processing engine capabilities that could already be used by other components
- We improve the current data structure

### Representing the workflow only with workflow_node records

One of the foundational requirements for this idea to work is to be able represent the entire workflow graph (top-level and blueprint-level) as `workflow_node` records. We would need to represent the parent/child relationship between top-level blueprint nodes and their child nodes, and also the left/right relationship between nodes. So, let's consider this workflow:

```
http1 -> test1 -- up --> http2
               - down -> http3 -> test2 -- up --> noop1
                                        - down -> noop2
```

Where test1 is a blueprint node with this structure:

```
filter1 -> approval1 -> if1 - true --> (up)
                            - false -> (down)
```

The workflow nodes would be:

```
ID    WF_ID   NODE_ID   PARENT_ID  PREVIOUS   PREVIOUS_CHANNEL
wn1   wf1     http1     -          -          -
wn2   wf1     test1     -          wn1        default
wn3   wf1     http2     -          wn2        up
wn4   wf1     http3     -          wn2        down
wn5   wf1     test2     -          wn4        default
wn6   wf1     noop1     -          wn5        up
wn7   wf1     noop2     -          wn5        down
wn8   wf1     filter1   wn2        -          -
wn9   wf1     approval1 wn2        wn8        default
wn10  wf1     if1       wn2        wn9        default
wn11  wf1     filter1   wn5        -          -
wn12  wf1     approval1 wn5        wn11       default
wn13  wf1     if1       wn5        wn12       default
```

One thing that gets more complicated though is what happens when a blueprint gets updated. When that happens, we need to update all `workflow_node` records that use that blueprint - and its children.

One idea of how to this is through `blueprint_update_requests`: when a blueprint is updated, we don't update it right away. We wait until the current execution finishes, and before creating another one, we update the blueprint definition.

How would the we represent the workflow graph entirely through `workflow_node` records?

### 1 - grouping outputs that look different

What if I don't have any fields to group by? For example, the case where you have this:

```
          | ----> time_window ----- |
noop ---- |                         | -----> group_by
          | -----> approval ------- |
```

Here, the approval component has output with one structure and time_window has output with another structure. In this case, group_by means only "wait until both connections emit something" to continue.

How do we solve this?

One idea here is to use the `root_event_id`. If no fields are specified, we only match on `root_event_id`.

### 2 - Queue check request needs special treatment

If I handle queue-check requests using `run_at` + new queue-check request being created by the action being executed, we could end up having way too many request processing. A better to do this is to not have a `run_at` at all queue-check requests. Instead, those requests have a 'idle' state. When EventRouter puts something in a workflow node's queue and that workflow node has a `queue-check` request in 'idle' state, we move it to 'pending' state. This way, we only process 'queue-check' requests when something appears in the queue.

We could also come up with some way for executions to subscribe to events that happen in the system, and a 'new-queue-item' would be one type of event that component executions can subscribe to.

## Not requiring polling for Semaphore component workflow updates

We maintain a registry of integration resources. The webhook data (URL, key) is on the integration resource level.

That way, we only provision the integration resource once, and components/triggers that reference the same integration resource use the same underlying thing.

On the integration resource endpoint, we:
- Check if there are triggers associated with it. For each trigger, call trigger
- Check if there are components associated with it. For each component, call component action
