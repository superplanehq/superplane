---
description: Expr syntax, $ and node names, root/previous, common functions, {{ }} in configuration.
---

# Expressions

SuperPlane uses **Expr** for expressions—to read payload data, transform values, and evaluate conditions.

## Accessing payload data

Use **`$['Node Name']`** to read any **upstream** node’s payload in the message chain (the node name must match the **display name** on the canvas):

```
$['Node Name'].field
$['Node Name'].nested.field
$['Node Name'].array[0].value
```

Examples:

```
$['GitHub onPush'].ref
$['GitHub onPush'].head_commit.message
$['Deploy 10%'].workflow_run.html_url
```

The **data-flow** skill describes how `$` is built during a run.

## SuperPlane functions

| Function | Description | Example |
|----------|-------------|---------|
| `root()` | Root payload that started the run | `root().data.ref` |
| `previous()` | Immediate upstream node’s payload | `previous().data.status` |
| `previous(n)` | Walk *n* levels upstream | `previous(2).data.version` |

## Canvas memory

`memory` is a namespace for reading records written by the **Add / Upsert / Update Memory** components. Records are scoped to the current canvas.

| Method | Description | Example |
|--------|-------------|---------|
| `memory.find(ns, matches)` | Every record in namespace `ns` whose stored values contain all fields in `matches`. Newest first. | `memory.find("deploys", {"env": "prod"})` |
| `memory.findFirst(ns, matches)` | Newest record matching, or `null`. | `memory.findFirst("deploys", {"env": "prod"}).version` |

`matches` is matched with JSONB containment — the record must contain every key/value you pass, but may carry additional fields.

## Common Expr functions

**String:** `lower()`, `upper()`, `trim()`, `split()`, `replace()`, `indexOf()`, `hasPrefix()`, `hasSuffix()`

**Array:** `filter()`, `map()`, `first()`, `last()`, `len()`, `any()`, `all()`, `count()`, `join()`

**Date:** `now()`, `date()`, `duration()` — with methods like `.Year()`, `.Month()`, `.Day()`, `.Hour()`

**Type conversion:** `int()`, `float()`, `string()`, `toJSON()`, `fromJSON()`, `toBase64()`, `fromBase64()`

Expr provides additional built-ins beyond those listed here.

## Using expressions in configuration

Wrap expressions in **double curly braces** in supported configuration fields.

**Dynamic text:**

```
Deployment of {{$['Listen to new Releases'].data.release.name}} has failed.
```

**Filter / condition:**

```
$['GitHub onPush'].ref == "refs/heads/main"
```

**String manipulation:**

```
indexOf(lower($['Slack Message'].data.text), "p1") != -1
```

**Conditional logic:**

```
$['Check for alerts'].data.status != "clear" || $['Health Check'].data.body.healthy == false
```
