The plan has been written to `plan.md`. Here's a summary of what it covers:

## Plan Summary

**Goal:** Remove the Pin button from homepage app cards and the "Pinned" grouped section. Keep the Star button as-is.

### Files changed (8 total — frontend only, no backend)

| File | What changes |
|---|---|
| `CanvasCardsGrid.tsx` | Delete the Pin `CanvasPreferenceButton`, remove `onTogglePin` prop, remove `Pin` lucide import |
| `index.tsx` | Delete the "Pinned" section rendering, remove `onTogglePin` wiring and `isPinned` filtering from `buildFolderedLayout`, remove `Pin` lucide import |
| `CanvasFolderSection.tsx` | Remove `onTogglePin` prop that it threads through to `CanvasCardsGrid` |
| `canvasAppPreferencePresentation.ts` | Remove `isPinned`/`pinnedAt` from the sort-rank logic; starred apps become rank 0 |
| `canvasAppPreferencePresentation.spec.ts` | Update assertions to match new sort order |
| `index.spec.tsx` | Remove pin button interaction from the preference test |
| `types.ts` | Remove `isPinned`/`pinnedAt` dead fields from `CanvasCardData` |
| `useHomePageCanvasList.ts` | Remove `isPinned`/`pinnedAt` from `toCanvasCardData` |

### Key decisions

- **No backend changes** — the API, protos, and DB schema for `pinned` are untouched. This keeps the change minimal and reversible.
- The `CanvasPreferenceButton` component itself is preserved (still needed for Star).
- Existing previously-pinned canvases silently fall back into the normal layout — no data loss.
- Verification: `make check.build.ui` + `make format.js` + manual UI check of the home page.