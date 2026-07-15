Everything looks clean. Here's a summary of all changes made:

## Summary

**Files changed** (8 files, all in `web_src/src/pages/home/`):

1. **`CanvasCardsGrid.tsx`** — Removed `Pin` import from lucide-react; removed `onTogglePin` from `CanvasCardsGridProps`, `CanvasCardsGrid`, `CanvasCardProps`, and `CanvasCard`; removed the Pin `CanvasPreferenceButton` block from the card UI.

2. **`index.tsx`** — Removed `Pin` import; removed `onTogglePin` prop from `Content`'s signature and call site; removed `pinnedCanvases` computation and the entire "Pinned" `<section>` block; removed the `isPinned` early-continue guard in `buildFolderedLayout` so formerly-pinned canvases fall into the regular unfiled layout; removed `onTogglePin` from all `CanvasCardsGrid` and `CanvasFolderSection` usages inside `Content`.

3. **`CanvasFolderSection.tsx`** — Removed `onTogglePin` from `CanvasFolderSectionProps`, function signature, and the `CanvasCardsGrid` call inside it.

4. **`canvasAppPreferencePresentation.ts`** — Simplified ranking: starred canvases are rank 0 (top), all others rank 1. Removed all `isPinned`/`pinnedAt` branches.

5. **`canvasAppPreferencePresentation.spec.ts`** — Rewrote test to verify starred-first ordering without any pin references.

6. **`index.spec.tsx`** — Replaced the pin/star test with a star-only test: verifies starred canvases sort to the top and the Unstar button calls `updateCanvasPreference` correctly.

7. **`types.ts`** — Removed `isPinned?: boolean` and `pinnedAt?: string` from `CanvasCardData`.

8. **`useHomePageCanvasList.ts`** — Removed `isPinned` and `pinnedAt` mappings from `toCanvasCardData`.

No backend changes. The API/database `pinned`/`pinnedAt` fields remain intact.