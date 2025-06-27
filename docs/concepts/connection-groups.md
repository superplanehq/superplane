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
    # The timeout for the connection group.
    # drop: drop the events received, and don't do anything
    # emit: emit the events received, but the resulting event will indicate that events from some connections were missing
    #
    timeout:
      after: 24h
      behavior: drop | emit
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

The events emitted by the connection group will have all the grouping fields and all the events that were grouped for that field set in it. For our example above, it would look like this:

```json
{
  "result": "received-all",
  "fields": {
    "version": "v1",
  },
  "events": {
    "preprod1": {...},
    "preprod2": {...}
  }
}
```

If the timeout is reached, the connection group will emit an event with the same fields, but with a `result` field set to `timed-out`, and only the events from the connections that have sent events will be included.

```json
{
  "result": "timed-out",
  "fields": {
    "version": "v1",
  },
  "events": {
    "preprod1": {...}
  }
}
```

So, when defining a stage input from a connection group event data, I would do it like this:

```yaml
apiVersion: v1
kind: Stage
metadata:
  name: prod
spec:
  inputs:
    - name: VERSION

  inputMappings:
    - values:
        - name: VERSION
          valueFrom:
            eventData:
              connection: preprod
              expression: fields.version
```
