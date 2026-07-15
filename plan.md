# Plan: Remove "Pin App" from Homepage

## 1. Summary

The homepage currently shows two inline action buttons on each app card: **Pin** (push-pin icon) and **Star** (star icon). This change removes the Pin button from app cards and the "Pinned" section header that groups pinned apps at the top of the homepage. Only the Star functionality is retained.

The backend API and data model retain `pinned` / `pinnedAt` support unchanged — this is purely a UI removal. Existing pinned canvases will no longer appear in a dedicated section and will no longer be pinnable from the UI.

---

## 2. Ordered Implementation Steps

### Step 1 — Remove the Pin button from `CanvasCardsGrid.tsx`

**File:** `web_src/src/pages/home/CanvasCardsGrid.tsx`

- Remove the `CanvasPreferenceButton` block that renders the `Pin` icon (lines 116–124).
- Remove the `onTogglePin` prop from `CanvasCardsGridProps`, `CanvasCardsGrid`, and `CanvasCard` interfaces and function signatures.
- Remove the `Pin` import from `lucide-react` (line 6) — it is no longer referenced in this file.

### Step 2 — Remove the "Pinned" section and `onTogglePin` from `index.tsx`

**File:** `web_src/src/pages/home/index.tsx`

- Remove the `onTogglePin` prop from the `Content` component's props interface and its call sites.
- Remove the `pinnedCanvases` computation and the `<section>` block that renders the "Pinned" heading and `CanvasCardsGrid` for pinned canvases (lines 140–182).
- Remove the `isPinned` check from `buildFolderedLayout` (line 252 — the early `continue` that skips pinned canvases from the "unfiled" list; pinned canvases should now fall through to the normal layout).
- Remove the `onTogglePin` handler passed from `HomePage` to `Content` (line 91).
- Remove the `Pin` import from `lucide-react` (line 5) — it is no longer used in this file.

### Step 3 — Remove `onTogglePin` from `CanvasFolderSection.tsx`

**File:** `web_src/src/pages/home/CanvasFolderSection.tsx`

- Remove `onTogglePin` from `CanvasFolderSectionProps` interface.
- Remove the `onTogglePin` parameter from `CanvasFolderSection` function signature.
- Remove the `onTogglePin` prop passed to `CanvasCardsGrid` inside `CanvasFolderSection`.

### Step 4 — Update tests in `index.spec.tsx`

**File:** `web_src/src/pages/home/index.spec.tsx`

- Remove or update the `"orders preferred canvases and requests preference updates"` test, which tests that clicking "Unpin app Z Free Canvas" calls `updateCanvasPreference` with `{ pinned: false }`. Since the Pin button no longer exists:
  - Remove the pin-related assertions and the `Unpin app Z Free Canvas` button interaction.
  - Keep (or simplify) the star-related assertions, since Star is preserved.
- Update the test's setup canvases if it relied on the "Pinned" section heading — the `Pinned` heading will no longer render.

### Step 5 — Update `canvasAppPreferencePresentation.ts` and its spec

**File:** `web_src/src/pages/home/canvasAppPreferencePresentation.ts`

The ordering function currently ranks pinned canvases first (rank 0), starred second (rank 1), then all others (rank 2). With Pin removed from the UI:
- Remove the `isPinned` / `pinnedAt` branch from `canvasPreferenceRank` and `preferenceTime`, making starred canvases rank 0 and all others rank 1.
- Update `preferenceTime` to only consider `starredAt`.

**File:** `web_src/src/pages/home/canvasAppPreferencePresentation.spec.ts`

- Remove or update assertions that reference `isPinned: true` / `pinnedAt`. Starred canvases should now sort to the top; previously-pinned canvases are treated as regular canvases.

### Step 6 — (Optional) Clean up `types.ts`

**File:** `web_src/src/pages/home/types.ts`

The `CanvasCardData` type still has `isPinned`, `pinnedAt` fields. These flow in from `useHomePageCanvasList.ts` → API data. Since the API still returns these fields and no code in other parts of the UI depends on them (they were only used in the homepage), they can be safely removed from `CanvasCardData` to avoid dead fields.

- Remove `isPinned?: boolean` and `pinnedAt?: string` from `CanvasCardData`.
- Remove `isPinned` and `pinnedAt` mappings from `toCanvasCardData` in `useHomePageCanvasList.ts`.

> **Note:** If removing from the type causes TypeScript errors elsewhere (e.g., the spec files that make canvases with `isPinned`), those call sites must be cleaned up too.

---

## 3. Files Changed and Why

| File | Change | Reason |
|---|---|---|
| `web_src/src/pages/home/CanvasCardsGrid.tsx` | Remove Pin button, `onTogglePin` prop, `Pin` import | Pin UI element lives here |
| `web_src/src/pages/home/index.tsx` | Remove "Pinned" section, `onTogglePin` wiring, `Pin` import | Homepage layout and data flow |
| `web_src/src/pages/home/CanvasFolderSection.tsx` | Remove `onTogglePin` prop thread | Passes `onTogglePin` down to `CanvasCardsGrid` |
| `web_src/src/pages/home/canvasAppPreferencePresentation.ts` | Remove pinned-first rank, keep starred-first | Sorting now irrelevant to pin |
| `web_src/src/pages/home/canvasAppPreferencePresentation.spec.ts` | Update ordering assertions | Reflects new sort logic |
| `web_src/src/pages/home/index.spec.tsx` | Remove pin button interactions from test | Button no longer in DOM |
| `web_src/src/pages/home/types.ts` | Remove `isPinned`, `pinnedAt` from `CanvasCardData` | Clean up dead fields (optional but clean) |
| `web_src/src/pages/home/useHomePageCanvasList.ts` | Remove `isPinned`/`pinnedAt` from `toCanvasCardData` | Follows type cleanup above |

**No backend changes required.** The API endpoint (`canvasesUpdateCanvasPreference`) and its `pinned` parameter remain intact. Existing data with `pinned: true` is ignored by the UI.

---

## 4. Verification

### Automated checks

```bash
# TypeScript build check — must pass with no errors
make check.build.ui

# JavaScript formatting
make format.js

# Unit tests (vitest, runs in the app container)
# Vitest picks up index.spec.tsx and canvasAppPreferencePresentation.spec.ts automatically
make check.build.ui   # also type-checks everything
```

### Manual UI check

1. Run `make dev.server` and open `http://localhost:8000`.
2. Navigate to the home page for an org that has apps.
3. Confirm no pin (push-pin) icon appears on any app card.
4. Confirm the Star button still appears and toggles correctly.
5. Confirm apps that were previously pinned now appear in the regular (unfiled or folder) section, not in a "Pinned" banner section.
6. Confirm previously-starred apps still float to the top of their section.

---

## 5. Risks / Out-of-Scope Notes

- **No backend changes** — the API, database, and protobuf definitions for `pinned` / `pinnedAt` are left intact. This is intentional: removing the UI feature does not require a database migration, and future re-enablement is easy.
- **Existing pinned canvases** — users who previously pinned apps will see those apps fall back into the regular/folder layout. No data is lost.
- **`CanvasPreferenceButton` component** — this shared helper in `CanvasCardsGrid.tsx` is still used for the Star button. Do not delete it; only remove the Pin invocation.
- **`useUpdateCanvasPreference` hook** — still used for starring. Its `pinned` parameter type can be left in place since the hook signature is shared. The call site in `index.tsx` simply stops passing `onTogglePin`.
- **Out of scope:** Removing the `pinned` field from the API response, protobuf definitions, database schema, or backend models. That is a separate, larger change.
- **Out of scope:** Adding any alternative "sort to top" affordance using pin. The goal is removal only.
