By default, every single event from every connection in a stage will generate a new event in its queue. But sometimes you don't want that.

That's where connection groups come in handy. Connection groups allow you to group events coming in from multiple connections using certain fields. Once the connection group has received events with the same grouping fields for all the connections, it will emit an event for itself.

This is how you define a connection group:

```yaml
apiVersion: v1
kind: ConnectionGroup
metadata:
  name: preprod
spec:

  #
  # Define your connections, just like you do for a stage.
  #
  connections:
    - type: TYPE_STAGE
      name: preprod1
    - type: TYPE_STAGE
      name: preprod2

  groupBy:

    #
    # The fields in the connection events we use to group things by.
    # Multiple fields can be used.
    # All the members of this group must have send
    # these fields in their events.
    #
    fields:
      - name: version
        expression: outputs.version

    #
    # Controls when an event is emitted from this group:
    # - EMIT_ON_ALL: after events for all group members have been received for the same grouping keys
    # - EMIT_ON_MAJORITY: after events for >50% of group members have been received for the same grouping keys
    #
    emitOn: EMIT_ON_ALL | EMIT_ON_MAJORITY
```

And this is how you use it as a connection for another stage:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: prod
spec:
  connections:
    - type: TYPE_CONNECTION_GROUP
      name: preprod
```

The events emitted by the connection group will have all the grouping keys and all the events that it grouped in it. For our example above, it would look like this:

```json
{
  "version": "v1",
  "events": {
    "preprod1": {...},
    "preprod2": {...}
  }
}
```
