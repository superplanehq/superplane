# Run View Feature -- Implementation Progress

> This document tracks incremental progress across context restarts.
> Each batch is independently testable. Do not proceed to the next batch
> until the current one is verified by the developer on a local instance.

## Status Key

- [ ] Not started
- [~] In progress
- [x] Implemented
- [v] Verified by developer

---

## Batch 1: DescribeRun API + Canvas Version Stamping [x]

**Goal**: A working `GET /api/v1/canvases/{canvas_id}/runs/{event_id}` endpoint that returns run details with a canvas version snapshot. New executions get stamped with the live canvas version ID.

**How to verify**:
1. Run `make db.migrate DB_NAME=superplane_dev` -- migrations apply cleanly
2. Start the server with `make dev.start`
3. Trigger a canvas run, wait for it to complete
4. Call the DescribeRun endpoint via curl/browser:
   `GET /api/v1/canvases/<canvasId>/runs/<eventId>`
5. Response should include: `run` (event with executions), `snapshot_version` (canvas version), `executions` (all node executions)
6. Check that new executions in the DB have `canvas_version_id` populated

**Tasks**:
- [x] Migration: add `canvas_version_id` column to `workflow_node_executions`
- [x] Migration: add `report_entry` column to `workflow_node_executions` and `workflow_events`
- [x] Model: add `CanvasVersionID *uuid.UUID` to `CanvasNodeExecution` struct
- [x] Model: add `ReportEntry` fields to `CanvasNodeExecution` and `CanvasEvent` structs
- [x] Proto: add `DescribeRun` RPC, request/response messages, `report_entry` fields
- [x] Proto: regenerate (`make pb.gen && make openapi.spec.gen && make openapi.client.gen && make openapi.web.client.gen`)
- [x] Action: create `pkg/grpc/actions/canvases/describe_run.go`
- [x] Service: wire `DescribeRun` in `pkg/grpc/canvas_service.go`
- [x] Auth: register in `pkg/authorization/interceptor.go`
- [x] Serialization: add `ReportEntry` to `SerializeCanvasEventWithExecutions` and `SerializeNodeExecutions`
- [x] Worker: stamp `CanvasVersionID` in `process_queue_context.go` execution creation
- [x] Worker: stamp `CanvasVersionID` in `node_queue_worker.go` error path
- [x] Verify: `go build ./pkg/...` passes

**Files changed**:
- `db/migrations/*_add-canvas-version-id-to-executions.{up,down}.sql` (new)
- `db/migrations/*_add-report-entry.{up,down}.sql` (new)
- `pkg/models/canvas_node_execution.go`
- `pkg/models/canvas_event.go`
- `protos/canvases.proto`
- `pkg/grpc/actions/canvases/describe_run.go` (new)
- `pkg/grpc/actions/canvases/list_canvas_events.go`
- `pkg/grpc/actions/canvases/list_node_executions.go`
- `pkg/grpc/canvas_service.go`
- `pkg/authorization/interceptor.go`
- `pkg/workers/contexts/process_queue_context.go`
- `pkg/workers/node_queue_worker.go`
- Auto-generated proto/SDK files

---

## Batch 2: Runs Mode Tab + RunsSidebar [x]

**Goal**: A third "Runs" tab in the canvas mode toggle. Clicking it shows a left sidebar with a scrollable list of runs. Selecting a run highlights it and syncs `?run=<eventId>` in the URL. The canvas area shows the snapshot version's graph (read-only). No RunSummary overlay yet.

**How to verify**:
1. Open a canvas in the browser
2. See "Editor | Live Canvas | Runs" in the secondary header
3. Click "Runs" -- left sidebar appears with run list, bottom console hides
4. Click a run -- it highlights, URL updates to `?run=<eventId>`
5. Canvas shows the snapshot graph (read-only, no add controls)
6. Reload page with `?run=<id>` -- run is pre-selected
7. Switch back to "Live Canvas" or "Editor" -- normal behavior resumes

**Tasks**:
- [x] `useDescribeRun` hook + `canvasKeys.run` query key in `useCanvasData.ts`
- [x] WebSocket: invalidate run query on execution events for selected run in `useCanvasWebsocket.ts`
- [x] Extend `CanvasModeToggle` to support `"runs"` value with notification badge
- [x] Create `RunsSidebar` component (`web_src/src/ui/RunsSidebar/index.tsx`)
- [x] `WorkflowPageV2`: add runs mode state (`isRunsMode`, `selectedRunEventId`, URL sync)
- [x] `CanvasPage`: accept `runsSidebar`, `headerMode === "runs"` -- hide building blocks
- [x] Verify: `tsc --noEmit` passes

**Files changed**:
- `web_src/src/hooks/useCanvasData.ts`
- `web_src/src/hooks/useCanvasWebsocket.ts`
- `web_src/src/ui/CanvasPage/components/CanvasModeToggle.tsx`
- `web_src/src/ui/RunsSidebar/index.tsx` (new)
- `web_src/src/pages/workflowv2/index.tsx`
- `web_src/src/ui/CanvasPage/index.tsx`
- `web_src/src/ui/CanvasPage/Header.tsx`

---

## Batch 3: RunSummary Overlay [x]

**Goal**: When a run is selected, a summary panel appears overlaying the canvas area. Shows run header, stats, and activity section with inline actions (push through, cancel, approve/reject).

**How to verify**:
1. Select a run in the RunsSidebar
2. RunSummary overlay appears with: trigger info, status badge, duration, step counts
3. If a run has active steps (pending approval, waiting), the Activity section shows them with action buttons
4. Click "Push Through" on a wait component -- execution advances
5. Click "Cancel" on a running execution -- it cancels
6. Approve/Reject on an approval component -- works correctly
7. Stats update in real-time via WebSocket as the run progresses

**Tasks**:
- [x] Create `usePushThroughHandler` hook
- [x] Create `RunSummary` component with header, stats, activity sections
- [x] Wire `RunSummary` as `runViewOverlay` in `WorkflowPageV2` -> `CanvasPage`
- [x] `CanvasPage`: render `runViewOverlay` prop in the canvas area
- [x] Verify: `tsc --noEmit` passes

**Files changed**:
- `web_src/src/pages/workflowv2/usePushThroughHandler.ts` (new)
- `web_src/src/ui/RunSummary/index.tsx` (new)
- `web_src/src/pages/workflowv2/index.tsx`
- `web_src/src/ui/CanvasPage/index.tsx`

---

## Batch 4: NodeDetailPanel [x]

**Goal**: Double-clicking a node in run view opens a detail panel showing that node's execution data (outputs, config, metadata, errors). Previous/Next navigation between nodes in the execution chain.

**How to verify**:
1. Select a run, double-click a node on the canvas
2. NodeDetailPanel appears with execution status, outputs, config, metadata
3. If the execution failed, error is shown in red
4. Previous/Next buttons navigate between nodes in the chain
5. Clicking away or pressing Escape closes the panel

**Tasks**:
- [x] Create `NodeDetailPanel` component
- [x] `CanvasPage`: add `onNodeDoubleClick` support on ReactFlow
- [x] `WorkflowPageV2`: track `runDetailNodeId` state, pass `runDetailPanel` to CanvasPage
- [x] `CanvasPage`: render `runDetailPanel` prop
- [x] Verify: `tsc --noEmit` passes

**Files changed**:
- `web_src/src/ui/NodeDetailPanel/index.tsx` (new)
- `web_src/src/pages/workflowv2/index.tsx`
- `web_src/src/ui/CanvasPage/index.tsx`

---

## Batch 5: Report Template Backend [x]

**Goal**: Triggers and components gain a `reportTemplate` config field. When a trigger fires, its template is resolved and stored on the event's `report_entry`. When a component finishes, its template is resolved and stored on the execution's `report_entry`. Templates use `{{ }}` expressions with fault-tolerant resolution.

**How to verify**:
1. Open a trigger's config -- see `reportTemplate` field (togglable)
2. Set a template like `Deployed {{ root().repository.name }}`
3. Open a component's config -- same field available
4. Trigger a run
5. Call DescribeRun API -- `report_entry` fields populated on the event and executions
6. Expression errors render as inline `` `error: msg` `` in the stored markdown

**Tasks**:
- [x] `AppendGlobalComponentFields` + `appendReportTemplateField` in `common.go`
- [x] Update `AppendGlobalTriggerFields` to include `reportTemplate`
- [x] `ResolveCustomNameTemplate` standalone function in `node_configuration_builder.go`
- [x] `ResolveReportTemplateFromPayload` (fault-tolerant) in `node_configuration_builder.go`
- [x] `ResolveReportTemplate` builder method in `node_configuration_builder.go`
- [x] Skip `reportTemplate` in `resolve`/`resolveWithSchema`
- [x] Trigger gRPC path: resolve + store report in `emit_node_event.go`
- [x] Trigger worker path: resolve + store report in `event_context.go`
- [x] Component path: resolve + store report in `execution_state_context.go` (with `SetConfigBuilder`)
- [x] Persist `report_entry` in `PassInTransaction` update
- [x] Verify: `go build ./pkg/...` passes

**Files changed**:
- `pkg/grpc/actions/common.go`
- `pkg/workers/contexts/node_configuration_builder.go`
- `pkg/grpc/actions/canvases/emit_node_event.go`
- `pkg/workers/contexts/event_context.go`
- `pkg/workers/contexts/execution_state_context.go`
- `pkg/workers/contexts/process_queue_context.go`
- `pkg/models/canvas_node_execution.go` (Pass update includes `report_entry`)

---

## Batch 6: Report UI [x]

**Goal**: Rich markdown rendering of report entries in the RunSummary. A "Report" tab in the ComponentSidebar for editing report templates. AutoCompleteInput gains `minRows` support.

**How to verify**:
1. Open RunSummary for a run that has report entries
2. Report section renders markdown with syntax highlighting, tables, admonitions, inline badges
3. Open a component's sidebar -- "Report" tab exists
4. Edit a `reportTemplate` with `{{ }}` expressions -- autocomplete works
5. Save -- template persists in node config
6. Trigger a run -- report entries render correctly in RunSummary

**Tasks**:
- [x] Add npm deps: `rehype-highlight`, `rehype-raw`, `rehype-sanitize`
- [x] Create `ReportMarkdown` component
- [x] Add Report section to `RunSummary` (renders `ReportMarkdown` per step)
- [x] Add `minRows` prop to `AutoCompleteInput`
- [x] Create `ReportTab` component
- [x] Add "Report" tab to `ComponentSidebar`
- [x] Verify: `make format.js && make check.build.ui`

**Files changed**:
- `web_src/package.json`
- `web_src/src/ui/RunSummary/ReportMarkdown.tsx` (new)
- `web_src/src/ui/RunSummary/index.tsx`
- `web_src/src/components/AutoCompleteInput/AutoCompleteInput.tsx`
- `web_src/src/ui/componentSidebar/ReportTab.tsx` (new)
- `web_src/src/ui/componentSidebar/index.tsx`

---

## Architecture Reference

```
Current Canvas Page Layout:
+-----------------------------------------------------------+
| PageHeader (logo, canvas name, settings)                  |
+-----------------------------------------------------------+
| SecondaryHeader                                           |
|  [Agent] [Editor | Live Canvas | *Runs*] [VC/Save/Pub]   |
+-----------------------------------------------------------+
| +----------+------------------------------+-----------+   |
| | Left     |  ReactFlow Canvas            | Right     |   |
| | sidebar  |  (in runs mode: snapshot     | sidebar   |   |
| | (Runs    |   version, read-only,        | (hidden   |   |
| |  Sidebar |   execution status overlays) |  in runs  |   |
| |  in runs |                              |  mode)    |   |
| |  mode)   |  +------------------------+  |           |   |
| |          |  | RunSummary overlay     |  |           |   |
| |          |  | (header, stats,        |  |           |   |
| |          |  |  activity, report)     |  |           |   |
| |          |  +------------------------+  |           |   |
| |          |                              |           |   |
| |          |  +------------------------+  |           |   |
| |          |  | NodeDetailPanel        |  |           |   |
| |          |  | (on double-click)      |  |           |   |
| |          |  +------------------------+  |           |   |
| +----------+------------------------------+-----------+   |
| (bottom console hidden in runs mode)                      |
+-----------------------------------------------------------+
```

## Key Design Decisions

1. **Entry point**: Third tab "Runs" in `CanvasModeToggle`
2. **Left sidebar**: `RunsSidebar` replaces version control sidebar in runs mode
3. **Bottom console**: Hidden in runs mode (RunsSidebar takes over its function)
4. **Canvas**: Shows snapshot version's graph, read-only, with execution status overlays
5. **RunSummary**: Overlay panel on top of the canvas area
6. **NodeDetailPanel**: Appears on double-click of a node in run view
7. **Report templates**: Fault-tolerant `{{ }}` expression resolution, stored at event/execution time
8. **URL sync**: `?run=<eventId>` persists selected run
9. **Real-time**: WebSocket invalidates run queries when execution events arrive for the selected run
