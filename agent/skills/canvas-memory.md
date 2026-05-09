# Canvas Memory

Canvas Memory provides persistent key-value storage scoped to the canvas. Use it to share state across nodes, accumulate values across runs, or cache expensive lookups.

## Components

### addMemory

Stores a new value for a key. Fails if the key already exists — use `upsertMemory` if you want create-or-update semantics.

```yaml
type: addMemory
key: "{{ $['Trigger'].run_id }}"
value: "{{ $['Trigger'].payload | toJSON }}"
```

### readMemory

Reads the current value for a key. Emits on the `found` channel if the key exists, `not_found` otherwise. You can also reference memory values inline using the `memory()` expression function (see [expressions.md](expressions.md)).

```yaml
type: readMemory
key: "last_deployed_sha"
```

Channels: `found`, `not_found`

### upsertMemory

Creates the key if it does not exist; updates it if it does.

```yaml
type: upsertMemory
key: "deploy_count"
value: "{{ int(memory('deploy_count')) + 1 }}"
```

### deleteMemory

Removes a key. No-ops silently if the key does not exist.

```yaml
type: deleteMemory
key: "{{ $['Trigger'].run_id }}"
```

## Using memory() in expressions

The `memory()` function reads a Canvas Memory key inline without a dedicated `readMemory` node:

```
{{ memory("last_deployed_sha") }}
{{ memory("deploy_count") != nil ? int(memory("deploy_count")) : 0 }}
```

Returns `nil` if the key does not exist. See [expressions.md](expressions.md) for the full expression reference.
