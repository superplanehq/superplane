# SuperPlane Performance Analysis Report

**Date**: 2026-03-28
**Scope**: Full-stack — Go backend (pkg/) + React frontend (web_src/src/)
**Findings**: 23 total | 4 CRITICAL, 10 HIGH, 8 MEDIUM, 1 LOW
**Weighted Score**: 40.50

---

## Executive Summary

This report covers a comprehensive analysis of the SuperPlane project's Go backend and React frontend. The most critical issues are **unbounded database queries in polling workers**, **a WebSocket Hub deadlock**, a **6,589-line monolithic React component**, and **lack of code splitting at the route level**.

---

## SECTION 1: DATABASE PERFORMANCE

### FINDING 1 (CRITICAL): Unbounded `ListPendingNodeExecutions()` Query

**File:** `pkg/models/canvas_node_execution.go:132-144`

Fetches **all** pending executions without any `LIMIT` clause. Polled every **60 seconds** by `NodeExecutor.Start()` at `pkg/workers/node_executor.go:64-74`. In a production backlog scenario with thousands of pending executions, this loads every row into memory on every tick.

**Impact:** Multi-second query times and GiB-level memory spikes under load.
**Recommendation:** Add a `LIMIT` clause (e.g., 500). The semaphore already limits concurrency to 25.

### FINDING 2 (CRITICAL): Unbounded `ListPendingCanvasEvents()` Query

**File:** `pkg/models/canvas_event.go:206-220`

Same pattern — fetches all pending events with no limit. Polled every **60 seconds** by `EventRouter.Start()`.

**Impact:** Same as Finding 1. Event storms can fetch thousands of rows unbounded.
**Recommendation:** Add `LIMIT` matching the semaphore capacity (e.g., 100 or 500).

### FINDING 3 (CRITICAL): Unbounded `ListNodeRequests()` Query (1s Poll)

**File:** `pkg/models/canvas_node_request.go:62-81`

Same unbounded fetch pattern, polled every **1 second** by `NodeRequestWorker.Start()`.

**Impact:** Polling every 1 second without a limit is especially dangerous — constant full-table-scan pressure under burst conditions.
**Recommendation:** Add LIMIT. Consider increasing poll interval to 5s or switching to event-driven consumption.

### FINDING 4 (HIGH): N+1 Query Pattern in `SerializeCanvas()`

**File:** `pkg/grpc/actions/canvases/serialization.go:21-139`

Issues **5+ sequential database queries** per call. Called for **every canvas** in `ListCanvases()` at `list_canvases.go:21-28`. For 50 canvases = 100-150 queries per list call.

**Impact:** Response time degrades linearly with canvas count.
**Recommendation:** Batch-load related data for all canvases at once before serialization.

### FINDING 5 (HIGH): O(n*m) String Comparison in Serialization

**File:** `pkg/grpc/actions/canvases/list_node_executions.go:414-431`

UUID-to-string comparison in nested loops: `event.ID.String() == execution.EventID.String()`. For 100 executions with 100 events = 10,000 string allocations.

**Impact:** GC pressure hotspot at scale.
**Recommendation:** Build `map[uuid.UUID]CanvasEvent` index. Compare UUIDs as bytes.

### FINDING 6 (HIGH): Missing `ConnMaxLifetime` in Database Pool

**File:** `pkg/database/connection.go:78-86`

Without `ConnMaxLifetime`, connections live indefinitely. Cloud environments with connection proxies have idle timeouts, causing intermittent "connection reset" errors.

**Recommendation:** Add `sqlDB.SetConnMaxLifetime(30 * time.Minute)`.

### FINDING 7 (HIGH): Default Pool Size of 5 Is Too Low

**File:** `pkg/database/connection.go:35-44`

6+ concurrent workers (each with semaphore concurrency of 25) share a single 5-connection pool.

**Impact:** Workers will contend for connections, leading to serialized execution.
**Recommendation:** Default should be 20-30.

### FINDING 8 (MEDIUM): Missing Composite Index on Queue Items

Query `WHERE workflow_id = ? AND node_id = ? ORDER BY created_at ASC` has no composite index.

**Recommendation:** `CREATE INDEX idx_wf_node_queue_items_node_created ON workflow_node_queue_items (workflow_id, node_id, created_at ASC);`

### FINDING 9 (MEDIUM): Missing Index for Ready Nodes Query

`ListCanvasNodesReady()` filters on `(state, type)` — no composite index exists.

**Recommendation:** `CREATE INDEX idx_workflow_nodes_state_type ON workflow_nodes (state, type) WHERE deleted_at IS NULL;`

---

## SECTION 2: CONCURRENCY AND MEMORY MANAGEMENT

### FINDING 10 (HIGH): `context.Background()` in Semaphore Acquisition

**Files:** `pkg/workers/node_queue_worker.go:81`, `pkg/workers/node_executor.go:82`, `pkg/workers/event_router.go:64`

Workers use `context.Background()` for semaphore acquisition. During shutdown, goroutines hang indefinitely.

**Recommendation:** Pass parent `ctx` to `semaphore.Acquire()`.

### FINDING 11 (HIGH): WebSocket Hub Deadlock

**File:** `pkg/public/ws/ws_hub.go:127-142`

`BroadcastToWorkflow` holds a **read lock** and calls `unregisterClient()` which requires a **write lock**. When a client's 4096-message buffer fills up, the entire Hub permanently deadlocks.

**Impact:** Full system deadlock killing all WebSocket connections.
**Recommendation:** Collect clients to unregister and handle after releasing the read lock.

### FINDING 12 (MEDIUM): Non-Thread-Safe DB Singleton Init

**File:** `pkg/database/connection.go:26-33`

Lazy initialization without mutex or `sync.Once`. Multiple goroutines calling `Conn()` simultaneously during startup can create multiple pools.

**Recommendation:** Use `sync.Once`.

---

## SECTION 3: API PERFORMANCE

### FINDING 13 (HIGH): Sequential DB Calls in DescribeCanvas

**File:** `pkg/grpc/actions/canvases/serialization.go:71-107`

4 independent database queries executed sequentially when `includeStatus = true`. Could be parallelized.

**Impact:** Response time = sum of all 4 queries instead of max. 50-200ms wasted.
**Recommendation:** Wrap in goroutines with `sync.WaitGroup`.

### FINDING 14 (MEDIUM): Per-Canvas DB Calls in ListCanvases

**File:** `pkg/grpc/actions/canvases/list_canvases.go:14-34`

Even with `includeStatus=false`, each canvas makes 2-3 DB queries.

**Recommendation:** Batch-load all versions and nodes in single queries.

---

## SECTION 4: FRONTEND RENDERING PERFORMANCE

### FINDING 15 (CRITICAL): 6,589-Line Monolithic Component

**File:** `web_src/src/pages/workflowv2/index.tsx`

124 `useMemo`/`useCallback`, 21 `useEffect` calls, 100+ imports. Any state change forces React to re-evaluate all memoization hooks.

**Impact:** Every WebSocket message triggers re-evaluation of the entire component tree.
**Recommendation:** Extract into `<CanvasEditor>`, `<CanvasSidebar>`, `<CanvasToolbar>`, etc.

### FINDING 16 (HIGH): No Route-Level Code Splitting

**File:** `web_src/src/App.tsx:1-142`

All pages imported eagerly. No `React.lazy()` used anywhere. Entire application bundled into a single chunk.

**Impact:** Initial page load downloads everything regardless of which page the user visits.
**Recommendation:** Use `React.lazy()` for route-level components with Vite code splitting.

### FINDING 17 (HIGH): Zustand Store Triggers Broad Re-Renders

**File:** `web_src/src/stores/nodeExecutionStore.ts:336-352`

Every update creates a new `Map` and increments a global `version` counter. Any component subscribing re-renders on every update regardless of which node changed.

**Impact:** N nodes on a canvas = N unnecessary re-renders per single node update.
**Recommendation:** Use Zustand selectors with `shallow` equality. Remove global `version` counter.

### FINDING 18 (MEDIUM): `staleTime: 0` on Canvas Query

**File:** `web_src/src/hooks/useCanvasData.ts:152`

Every mount/focus triggers refetch despite WebSocket handling real-time updates.

**Recommendation:** Set `staleTime` to 10-30 seconds.

### FINDING 19 (MEDIUM): 3-Second Polling for Canvas Memory

**File:** `web_src/src/hooks/useCanvasData.ts:774`

`refetchInterval: 3000` polls regardless of whether memory view is active.

**Recommendation:** Only enable polling when the memory panel is visible.

---

## SECTION 5: WEBSOCKET PERFORMANCE

### FINDING 20 (HIGH): Excessive Query Invalidation per WebSocket Message

**File:** `web_src/src/hooks/useCanvasWebsocket.ts:59-120`

Every WebSocket event invalidates the infinite events list. A single execution flow generates ~20+ events = 20+ full refetches.

**Impact:** Thundering herd on the server and UI jank.
**Recommendation:** Debounce invalidation (500ms) or delay until execution chain completes.

### FINDING 21 (MEDIUM): 4096-Message WebSocket Client Buffer

**File:** `pkg/public/ws/ws_hub.go:149`

With JSON payloads of 1-5KB, this could consume 4-20MB per client.

**Impact:** 100 users = 400MB-2GB for WebSocket buffers alone.
**Recommendation:** Reduce to 256-512 with message coalescing.

---

## SECTION 6: BUILD AND CONFIGURATION

### FINDING 22 (MEDIUM): Missing Manual Chunk Strategy in Vite

**File:** `web_src/vite.config.ts:72-91`

`rollupOptions` are commented out. Large dependencies (monaco-editor, react-flow, lodash) all in main bundle.

**Recommendation:** Enable `manualChunks` for vendor splitting.

### FINDING 23 (LOW): `FailInTransaction` Reads Outside Transaction

**File:** `pkg/models/canvas_node_execution.go:515-523`

Parent execution fetched outside transaction, then modified inside it.

**Impact:** Potential lost updates under concurrent access.
**Recommendation:** Use `FindNodeExecutionInTransaction(tx, ...)`.

---

## Summary Table

| # | Severity | Category | Finding | Est. Impact |
|---|----------|----------|---------|-------------|
| 1 | CRITICAL | Database | Unbounded ListPendingNodeExecutions | OOM risk |
| 2 | CRITICAL | Database | Unbounded ListPendingCanvasEvents | OOM risk |
| 3 | CRITICAL | Database | Unbounded ListNodeRequests (1s poll) | CPU/memory |
| 11 | HIGH | Concurrency | WebSocket Hub deadlock | Full system deadlock |
| 4 | HIGH | Database | N+1 in ListCanvases | 100+ queries/list |
| 5 | HIGH | Database | O(n*m) string comparison | GC pressure |
| 6 | HIGH | Database | Missing ConnMaxLifetime | Stale connections |
| 7 | HIGH | Database | Default pool size of 5 | Contention |
| 10 | HIGH | Concurrency | context.Background() in semaphore | Shutdown hang |
| 13 | HIGH | API | Sequential DB calls in DescribeCanvas | 50-200ms wasted |
| 15 | CRITICAL | Frontend | 6,589-line monolithic component | Re-render storms |
| 16 | HIGH | Frontend | No route-level code splitting | Slow initial load |
| 17 | HIGH | Frontend | Zustand broad re-renders | Unnecessary renders |
| 20 | HIGH | WebSocket | Excessive query invalidation | Thundering herd |

## Top 5 Priority Fixes

1. **Fix WebSocket Hub deadlock** (Finding 11) — correctness bug, 30 min fix
2. **Add LIMIT to all polling queries** (Findings 1-3) — one-line changes, 15 min
3. **Increase DB pool size + add ConnMaxLifetime** (Findings 6-7) — config change, 10 min
4. **Debounce WebSocket query invalidation** (Finding 20) — 1 hour
5. **Add route-level code splitting** (Finding 16) — 2 hours

---
*Generated by AQE v3 Performance Reviewer Agent*
