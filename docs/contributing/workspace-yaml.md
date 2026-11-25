# Workspace YAML stage kinds

The v2 Factory manifest at `.superplane/main.yml` configures an existing
workspace. Export includes every stage and omits stage `kind` fields. Keep
the document-level `kind: Factory`.

For example, an exported stage entry can contain:

```yaml
key: verify
name: Verify
automations: []
```

SuperPlane matches each stage to the live workspace by its stable `key` and
uses the live stage kind. A stage name does not determine its kind. Keep
the exported keys when you rename stages.

Stage `kind` is not a supported YAML field. SuperPlane uses the live kind for
stage order, phase settings, and runner validation.

Include every existing stage exactly once. YAML cannot add or remove stages.
Backlog must remain first, and Done must remain last. Keep existing `run`
bindings. Only phases support `max_parallelism`. Existing permissions for
stage names and order still apply.

Export followed by plan produces no changes when the workspace has not changed.
