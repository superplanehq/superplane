The plan has been written to `/workspace/superplane/plan.md`. Here's a summary of what I found and documented:

---

## What I Found

The "Pin App" feature spans four files in `web_src/src/pages/home/`:

1. **`CanvasCardsGrid.tsx`** — renders the thumbtack `<Pin>` button on each card alongside the star button
2. **`index.tsx`** — wires the `onTogglePin` callback, renders the **"Pinned" section** at the top of the page, and has a guard in `buildFolderedLayout` that hides pinned canvases from the main grid
3. **`CanvasFolderSection.tsx`** — passes `onTogglePin` through to the grid
4. **`index.spec.tsx`** — has a test that clicks "Unpin app" and asserts on the Pinned section

---

## Key Risks Called Out

- The `buildFolderedLayout` function has an `if (canvas.isPinned) continue` guard that **must be removed** along with the Pinned section — otherwise previously-pinned canvases would vanish from the UI entirely.
- The backend API still stores `pinned`/`pinnedAt`; this is intentionally out of scope.
- `canvasAppPreferencePresentation.ts` sorting logic references `isPinned` but is left unchanged since it's harmless and keeps the scope minimal.