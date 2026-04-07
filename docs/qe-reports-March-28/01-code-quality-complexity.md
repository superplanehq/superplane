# SuperPlane Code Quality & Complexity Analysis Report

**Date**: 2026-03-28
**Scope**: Full-stack — Go backend (944 files, 407,022 lines) + TypeScript frontend (720 files, 137,910 lines)
**Total Functions Analyzed**: 6,931

---

## Executive Summary

The SuperPlane codebase has an average maintainability index of 86.0 (Go) and 94.1 (TypeScript) — both in "Good" range. However, **severe hotspots** exist that represent outsized risk:

- **382 functions** exceed cyclomatic complexity 15 (threshold)
- **209 functions** exceed cyclomatic complexity 20
- **21 files** are in critical maintainability range (MI < 20)
- The **#1 risk**: `workflowv2/index.tsx` — a 6,589-line god component with 179 hooks and cyclomatic complexity exceeding 1,100

---

## Top 20 Most Complex Functions

| Rank | Cyc | Lines | Function | File | Lang |
|------|-----|-------|----------|------|------|
| 1 | 284 | 901 | `safeName` | `workflowv2/index.tsx:4945` | TS |
| 2 | 218 | 1090 | `pushIndexed` (inner) | `workflowv2/index.tsx:553` | TS |
| 3 | 171 | 745 | `getIncomingNodes` (inner) | `workflowv2/index.tsx:2342` | TS |
| 4 | 158 | 900 | `ComponentSidebar` | `componentSidebar/index.tsx:181` | TS |
| 5 | 115 | 542 | `tokenize` | `lib/exprEvaluator.ts:54` | TS |
| 6 | 108 | 610 | `handleBeforeUnload` (inner) | `workflowv2/index.tsx:1643` | TS |
| 7 | 101 | 715 | `componentType` (inner) | `workflowv2/index.tsx:4195` | TS |
| 8 | 81 | 575 | `CanvasContent` | `CanvasPage/index.tsx:1877` | TS |
| 9 | 81 | 417 | `handleKeyDown` | `AutoCompleteInput.tsx:1048` | TS |
| 10 | 81 | 370 | `existingNodeNames` (inner) | `workflowv2/index.tsx:3336` | TS |
| 11 | 80 | 258 | `renderField` | `configurationFieldRenderer/index.tsx:269` | TS |
| 12 | 79 | 526 | `getApprovalStatusColor` | `chainItem/ChainItem.tsx:315` | TS |
| 13 | 61 | 263 | `Sentry.ListResources` | `integrations/sentry/sentry.go:555` | Go |
| 14 | 58 | 236 | `UpdateApp.Execute` | `integrations/digitalocean/update_app.go:224` | Go |
| 15 | 47 | 179 | `Hetzner.ListResources` | `integrations/hetzner/hetzner.go:98` | Go |
| 16 | 37 | 90 | `GCP.ListResources` | `integrations/gcp/gcp.go:880` | Go |
| 17 | 36 | 102 | `validateFieldValue` | `configuration/validation.go:623` | Go |
| 18 | 36 | 131 | `ParseCanvas` | `canvases/serialization.go:197` | Go |
| 19 | 34 | 119 | `OnIncidentTimelineEvent` | `integrations/rootly/on_incident_timeline_event.go:222` | Go |
| 20 | 32 | 159 | `Server.setupOwner` | `public/setup_owner.go:41` | Go |

**Key observation**: Top 12 most complex functions are all TypeScript. Go complexity concentrates in integration `ListResources` and `Execute` methods.

---

## Top 10 Code Smells

### 1. God Component — `WorkflowPageV2` (CRITICAL)
**File**: `web_src/src/pages/workflowv2/index.tsx` (6,589 lines)
- 179 hooks (30 useState, 20 useEffect, 85 useCallback, 38 useMemo, 6 useRef)
- 48 toast notifications, 43 loading states
- 238 commits since January 2025 (highest churn of any file)
- Responsible for canvas editing, version control, change requests, node management, sidebar, websocket, drag-and-drop, YAML import/export, AI operations, approvals — all in one function

### 2. God Object — `*Client` with 598 Methods (HIGH)
Aggregated across `pkg/integrations/*/client.go` files. Single `Client` struct per integration with massive surface area (e.g., `digitalocean/client.go` at 2,333 lines).

### 3. Data Clumps / Long Parameter Lists (MEDIUM-HIGH)
48 Go functions accept 5+ parameters, 9 accept 8-9 parameters (e.g., `CreateIntegration` takes 9). Repeated `(ctx, registry, encryptor, authService, ...)` tuples indicate missing aggregate types.

### 4. Excessive Proto Conversion Boilerplate (HIGH)
`pkg/grpc/actions/common.go` (1,177 lines, 52 functions): 84 TypeOptions references with mirrored `toProto`/`protoTo*` pairs. Textbook mechanical duplication.

### 5. Monster Configuration Methods (HIGH)
`pkg/integrations/gcp/compute/create_vm.go:1376` (900 lines), `pkg/integrations/aws/ecs/service.go:247` (1,160 lines). Data files masquerading as code — 415 `configuration.Field{}` declarations total.

### 6. Feature Envy — `CanvasPage` + Sidebar (MEDIUM-HIGH)
`web_src/src/ui/CanvasPage/index.tsx` (3,234 lines): Multiple components in one file, deeply intertwined prop passing, 12 `any` type usages.

### 7. Missing Default in Switch Statements (MEDIUM)
68 Go `switch` statements lack `default` clauses, including type-dispatch code in validation and proto conversion. Unknown types silently fall through.

### 8. Duplicated ListResources Pattern (MEDIUM)
106 structurally similar `ListResources` implementations averaging 100-260 lines each, differentiated only by client calls and field mappings.

### 9. Unconstrained Goroutine Spawning (MEDIUM)
Some workers (cleanup workers) spawn goroutines without concurrency limits, while others properly use semaphores.

### 10. TypeScript `any` Proliferation (MEDIUM)
`custom-component/index.tsx` (21 usages), `CanvasPage/index.tsx` (12), `CustomComponentBuilderPage/index.tsx` (17) — type safety eroded in the most complex files.

---

## Top 10 Largest/Most Complex Files

| Rank | Lines | MI | Churn | File | Key Issues |
|------|-------|-----|-------|------|------------|
| 1 | 6,589 | 0.0 | 238 | `workflowv2/index.tsx` | God component, 179 hooks |
| 2 | 3,234 | 0.0 | 163 | `CanvasPage/index.tsx` | Multi-component, 338 complexity |
| 3 | 2,368 | 0.0 | — | `gcp/compute/create_vm.go` | 900-line Configuration |
| 4 | 2,333 | 0.0 | — | `digitalocean/client.go` | 30+ method god client |
| 5 | 1,568 | 0.0 | — | `AutoCompleteInput.tsx` | 272 complexity |
| 6 | 1,485 | 0.0 | — | `gcp/gcp.go` | 248 complexity |
| 7 | 1,434 | 0.0 | — | `AutoCompleteInput/core.ts` | Expression tokenizer |
| 8 | 1,406 | — | — | `aws/ecs/service.go` | 1,160-line function |
| 9 | 1,359 | 11.4 | 51 | `custom-component/index.tsx` | 178 complexity, 21 `any` |
| 10 | 1,325 | 4.6 | — | `gcp/compute/list_resource_handler.go` | Resource caching, 200 complexity |

---

## Churn-Complexity Hotspots (Highest Defect Risk)

| File | Commits | Complexity | Risk |
|------|---------|------------|------|
| `workflowv2/index.tsx` | 238 | 1,121 | **CRITICAL** |
| `CanvasPage/index.tsx` | 163 | 338 | **HIGH** |
| `BuildingBlocksSidebar/index.tsx` | 104 | 127 | MEDIUM |
| `server.go` + `public/server.go` | 102+97 | 122 | MEDIUM |
| `componentSidebar/index.tsx` | 79 | 112 | MEDIUM |

---

## Maintainability Index Summary

**Go Backend**: Average MI **86.0** (Good)
- 63% of files Good (MI >= 80)
- 13 files Critical (MI < 20) — concentrated in integrations and proto conversion

**TypeScript Frontend**: Average MI **94.1** (Good)
- 80% of files Good (MI >= 80)
- 8 files Critical (MI < 20) — concentrated in workflow editor and canvas page

---

## Refactoring Recommendations (Prioritized)

### Priority 1 (Critical): Decompose `WorkflowPageV2`
Extract into 8-12 focused sub-components and custom hooks. Estimated reduction: 6,589 -> 800 lines for parent orchestrator. 10x+ testability improvement.

### Priority 2 (High): Move Configuration Declarations to Data Files
Move 900-1,160 line `Configuration()` methods to YAML/JSON. Eliminates ~50,000 lines of nested struct literals.

### Priority 3 (High): Generate Proto Conversion Code
Code-generate `toProto`/`protoTo*` pairs. Eliminates ~800 lines of mechanical duplication.

### Priority 4 (High): Decompose `CanvasPage`
Extract 3 components into separate files with explicit prop interfaces.

### Priority 5 (Medium): Introduce Parameter Objects in gRPC Actions
Create `ActionContext` aggregates. Reduce parameter counts from 8-9 -> 2-3.

### Priority 6 (Medium): Standardize Worker Concurrency Limits
Add `semaphore.Weighted` to all workers, including cleanup workers.

### Priority 7 (Medium): Add Default Clauses to Switch Statements
Add to all 68 missing `default` clauses, especially type-dispatch code.

---

**Bottom Line**: 80%+ of the codebase is well-maintained. The problems are concentrated in (1) the frontend workflow editor — a 6,589-line monolith demanding immediate decomposition, and (2) integration Configuration methods embedding data as code. Addressing these two areas would dramatically improve the overall quality profile.

---
*Generated by AQE v3 Code Complexity Agent*
