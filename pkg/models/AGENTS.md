# Models and database access

Read this file when you edit `pkg/models` or add database access around
models. For repository setup, commands, and generated-file rules, read
[AGENTS.md](../../AGENTS.md) first.

## Database Transaction Guidelines

We are moving away from `database.Conn()` inside `pkg/models` and from the
`FindX` / `FindXInTransaction` dual API. CI tracks remaining legacy usage via
`make check.models.tx.debt`; do not add new `*InTransaction` definitions or
`database.Conn()` call sites in `pkg/models`.

**Why:**

- Calling `database.Conn()` inside model code breaks transaction isolation when
  the caller already holds a `tx`.
- Conn wrappers plus `*InTransaction` methods duplicate API surface without
  adding behavior.

**Preferred pattern:** pass an explicit `*gorm.DB` as the first parameter.
Callers outside `pkg/models` obtain it with `database.DB(ctx)` (request-scoped,
attaches OpenTelemetry trace context).

```go
func FindCanvas(tx *gorm.DB, orgID, id uuid.UUID) (*Canvas, error) {
    var canvas Canvas
    err := tx.Where("organization_id = ? AND id = ?", orgID, id).First(&canvas).Error
    if err != nil {
        return nil, err
    }
    return &canvas, nil
}

// Handler (no surrounding transaction):
canvas, err := models.FindCanvas(database.DB(ctx), orgID, canvasID)

// Inside an existing transaction:
err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
    canvas, err := models.FindCanvas(tx, orgID, canvasID)
    return err
})
```

Rules:

- **NEVER** call `database.Conn()` inside `pkg/models` — pass the `*gorm.DB` from
  the caller instead.
- **NEVER** call a model function that uses `database.Conn()` internally while you
  already hold a `tx`.
- **Always propagate** the `*gorm.DB` through the entire call chain — pass it as
  the first parameter to functions that need database access.
- **Do not add** new `FindX` + `FindXInTransaction` pairs or conn wrappers; use a
  single function with an explicit `*gorm.DB` parameter.
- **Context constructors** that perform database queries must accept `tx *gorm.DB`
  as their first parameter.

When touching legacy `*InTransaction` or conn-wrapper code, migrate to the
explicit-parameter pattern when practical and update the debt baseline with
`make check.models.tx.debt.baseline.update`.

### Model file layout (`pkg/models`)

Order declarations in each model file as follows:

1. **Struct** — package constants used by the model, then the struct type.
2. **Constructors** — `New…` functions that build values for the model (including
   name/ID helpers).
3. **Getters** — methods on the struct (e.g. `TableName()`, computed accessors).
4. **Database access** — functions whose first parameter is `tx *gorm.DB` (or
   `db *gorm.DB`).

Place private helpers after the public API in the file.

### Models API shape (`pkg/models`)

Choose one style per concern and stick to it. Prefer object style when you
already have a model handle; do not invent free functions that re-take IDs you
already hold.

| Situation | Prefer | Example |
| --- | --- | --- |
| Operation on a loaded model | Method on the struct | `node.HardDelete(tx)` |
| Multi-step / configurable DB work for a model | Package constructor + collaborator/builder | `NewNodeResourceCleaner(tx, node).ForUnreferenced().WithLimit(n).Run()` |
| Lookup / list when you do **not** have a handle | Package function with `tx` first | `ListDeletedCanvasNodes(tx, …)`, `FindCanvas(tx, …)` |

Rules:

- **Do not** add `models.HardDeleteCanvasNode(tx, orgID, nodeID)` (or similar)
  when the caller already has `*CanvasNode` — that forces an extra find and mixes
  procedural style with OO for the same concern.
- **Do not** hang multi-step cleanup/publish logic as a thick method chain on the
  aggregate when a dedicated collaborator is clearer (`NodeResourceCleaner`,
  canvas publisher patterns).
- Keep SQL / GORM deletes and queries in `pkg/models`. Workers and gRPC actions
  **orchestrate** (lock → clean → hard-delete); they do not own batched delete
  queries.
- Receivers on model methods should use a short name consistent with the type
  (`c` for `*CanvasNode`, etc.), matching nearby code in the file.

```go
// Good: handle already loaded
if err := node.HardDelete(tx); err != nil {
    return err
}

// Good: multi-step cleanup as a collaborator
n, err := NewNodeResourceCleaner(tx, node).ForUnreferenced().WithLimit(batchSize).Run()

// Good: no handle yet — package function
nodes, err := ListDeletedCanvasNodes(tx, before, limit)

// Avoid: free function that re-keys a node you already have
_ = HardDeleteCanvasNode(tx, node.OrganizationID, node.ID)
```
