import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { ConsoleLayoutItem, ConsolePage, ConsolePanel } from "@/hooks/useCanvasData";

import { DEFAULT_CONSOLE_PAGE_ID, DEFAULT_CONSOLE_PAGE_NAME, MAX_CONSOLE_PAGES } from "./consoleYaml";
import { templateForPanelType, type PanelType } from "./panelTypes";

const SAVE_DEBOUNCE_MS = 500;

type UseConsolePagesStateOptions = {
  pages: ConsolePage[];
  onChange: (next: { pages: ConsolePage[] }) => void;
  onEffectiveChange?: (next: { pages: ConsolePage[] }) => void;
  activePageId: string | null;
  onActivePageIdChange: (id: string) => void;
  // Identifies the canvas whose console this hook is editing. React
  // Router reuses this route component across canvases, so a change
  // here (rather than an unmount) is our signal that any in-flight
  // debounced save from the outgoing canvas must be discarded before
  // it can be misrouted onto the destination canvas's mutation. Pass
  // `undefined` in tests / storybook where canvas identity is not
  // meaningful.
  canvasId?: string;
};

/**
 * Centralized state for the multi-page console editor. Manages the list
 * of pages, the active page id, and per-page panel/layout edits with a
 * debounced save that mirrors the pre-pages behavior.
 *
 * Callers pass:
 * - `pages`: canonical pages (persisted or dematerialized from YAML).
 * - `onChange`: commits the full pages document. Debounced.
 * - `onEffectiveChange`: immediate local-effective callback for diff /
 *   staging indicators (fires without the debounce).
 * - `activePageId` / `onActivePageIdChange`: controlled by the parent
 *   so URL sync and last-visited persistence stay outside this hook.
 */
export function useConsolePagesState({
  pages,
  onChange,
  onEffectiveChange,
  activePageId,
  onActivePageIdChange,
  canvasId,
}: UseConsolePagesStateOptions) {
  const [localPages, setLocalPages, queueSave] = useDebouncedPages({ pages, onChange, onEffectiveChange, canvasId });

  const activePage = useMemo(
    () => localPages.find((page) => page.id === activePageId) ?? localPages[0] ?? null,
    [localPages, activePageId],
  );

  const activePanels = activePage?.panels ?? [];
  const activeLayout = activePage?.layout ?? [];

  const activePageIdForUpdates = activePage?.id ?? null;
  const panelHandlers = usePanelHandlers({
    activePageId: activePageIdForUpdates,
    setLocalPages,
    queueSave,
    onActivePageIdChange,
  });
  const pageHandlers = usePageHandlers({ setLocalPages, queueSave, activePageId, onActivePageIdChange });

  return {
    localPages,
    activePage,
    activePanels,
    activeLayout,
    ...panelHandlers,
    ...pageHandlers,
  };
}

function useDebouncedPages({
  pages,
  onChange,
  onEffectiveChange,
  canvasId,
}: {
  pages: ConsolePage[];
  onChange: (next: { pages: ConsolePage[] }) => void;
  onEffectiveChange?: (next: { pages: ConsolePage[] }) => void;
  canvasId?: string;
}) {
  const [localPages, setLocalPages] = useState<ConsolePage[]>(pages);
  const lastPropsHashRef = useRef<string>("");

  // Reset local editor state on canvas switch. React Router reuses
  // this route component across `apps/:id`, so a canvas switch would
  // otherwise leave the outgoing canvas's diverged local state in
  // `localPages`. The props-sync effect below only fires when the
  // JSON hash of `pages` actually changes; two canvases with the
  // same pages payload (commonly, two empty consoles) slip past
  // that check. Combined with `activePage = localPages.find(...) ??
  // localPages[0]` upstream, canvas A's unsaved local pages then
  // render on canvas B, and a subsequent edit stages A's content
  // onto B via `queueSave`.
  //
  // Standard React "reset state on prop change during render"
  // pattern (https://react.dev/reference/react/useState#storing-information-from-previous-renders):
  // running the reset in-render ensures `localPages` is aligned with
  // the incoming canvas before any effect (including the effective-
  // change callback that drives staging indicators) observes stale
  // state on this render. Mirrors the equivalent reset in
  // `useConsoleActivePageInitial`.
  const [trackedCanvasId, setTrackedCanvasId] = useState(canvasId);
  if (trackedCanvasId !== canvasId) {
    setTrackedCanvasId(canvasId);
    setLocalPages(pages);
    lastPropsHashRef.current = JSON.stringify(pages);
  }

  useEffect(() => {
    const next = JSON.stringify(pages);
    if (next !== lastPropsHashRef.current) {
      lastPropsHashRef.current = next;
      setLocalPages(pages);
    }
  }, [pages]);

  useEffect(() => {
    onEffectiveChange?.({ pages: localPages });
  }, [localPages, onEffectiveChange]);

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // `pendingRef` also captures the canvas the payload was produced
  // for. Every fire path (debounce timer, unmount flush) checks that
  // the pending canvas still matches the current one before invoking
  // `onChange`. Without this, an in-flight debounce (or an unmount
  // flush that runs after a canvas switch happened to leave the
  // component mounted long enough) can hand the outgoing canvas's
  // pages to `updateConsoleMutationRef.current.mutate`, which by
  // that point points at the destination canvas — quietly staging
  // canvas A's console onto canvas B. The tests below (`does not
  // stage a pending save from a previous canvas`) lock this in.
  const pendingRef = useRef<{ pages: ConsolePage[]; canvasId: string | undefined } | null>(null);
  const canvasIdRef = useRef(canvasId);

  const queueSave = useCallback(
    (nextPages: ConsolePage[]) => {
      pendingRef.current = { pages: nextPages, canvasId: canvasIdRef.current };
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        const pending = pendingRef.current;
        pendingRef.current = null;
        saveTimerRef.current = null;
        if (!pending) return;
        if (pending.canvasId !== canvasIdRef.current) return;
        onChange({ pages: pending.pages });
      }, SAVE_DEBOUNCE_MS);
    },
    [onChange],
  );

  // Eagerly discard any pending debounced save on canvas switch.
  // React Router keeps this component mounted when the user
  // navigates between canvases (same `apps/:id` route), so absent
  // this effect the timer above would still fire ~500ms later and
  // route the outgoing canvas's pages through the current mutation,
  // which is already the destination canvas's. The pending-canvas
  // guard inside `queueSave`'s timer is a belt-and-suspenders for
  // the narrow window where the timer's `setTimeout` scheduler
  // could theoretically fire before this effect runs.
  useEffect(() => {
    const previousCanvasId = canvasIdRef.current;
    canvasIdRef.current = canvasId;
    if (previousCanvasId === canvasId) return;
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = null;
    }
    pendingRef.current = null;
  }, [canvasId]);

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      const pending = pendingRef.current;
      pendingRef.current = null;
      if (!pending) return;
      if (pending.canvasId !== canvasIdRef.current) return;
      onChange({ pages: pending.pages });
    };
  }, [onChange]);

  return [localPages, setLocalPages, queueSave] as const;
}

type PanelHandlersOptions = {
  activePageId: string | null;
  setLocalPages: React.Dispatch<React.SetStateAction<ConsolePage[]>>;
  queueSave: (next: ConsolePage[]) => void;
  onActivePageIdChange: (id: string) => void;
};

function usePanelHandlers({ activePageId, setLocalPages, queueSave, onActivePageIdChange }: PanelHandlersOptions) {
  const withActivePage = useCallback(
    (updater: (page: ConsolePage) => ConsolePage) => {
      setLocalPages((prev) => {
        if (!activePageId) return prev;
        const next = prev.map((page) => (page.id === activePageId ? updater(page) : page));
        queueSave(next);
        return next;
      });
    },
    [activePageId, queueSave, setLocalPages],
  );

  const handleAddPanel = useCallback(
    (name: string, type: PanelType = "markdown") => {
      const trimmedName = name.trim();
      const baseId = panelBaseIdFromName(trimmedName, type);
      let createdId: string = baseId;

      setLocalPages((prev) => {
        // An empty console auto-adopts the default page on the first
        // panel add. Without this, the "Add panel" button in the header
        // and the empty-state CTA both fail silently because there is
        // no active page to update.
        const wasEmpty = prev.length === 0;
        const base = wasEmpty
          ? [{ id: DEFAULT_CONSOLE_PAGE_ID, name: DEFAULT_CONSOLE_PAGE_NAME, panels: [], layout: [] }]
          : prev;
        const targetId = wasEmpty ? base[0]!.id : (activePageId ?? base[0]!.id);
        // Note: there is intentionally no client-side per-page panel
        // cap guard here. `useUpdateCanvasConsole` runs
        // `validateConsolePagesDelta` against the committed baseline,
        // which correctly allows grandfathered over-cap pages to
        // regrow up to their previously committed count. A blanket
        // `>= MAX_CONSOLE_PANELS_PER_PAGE` guard would over-eagerly
        // freeze grandfathered pages at 20 even when the mutation
        // would accept adding more. Rapid clicks that push past the
        // cap on a fresh page are caught at mutation time (rollback +
        // error toast). The UI disables the control at the cap for
        // fresh pages, which handles the common case.
        const next = base.map((page) => {
          if (page.id !== targetId) return page;
          const id = uniquePanelId(page.panels, baseId);
          createdId = id;
          const newPanel: ConsolePanel = { id, type, content: templateForPanelType(type, trimmedName) };
          const maxBottom = page.layout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
          const newLayoutItem: ConsoleLayoutItem = { i: id, x: 0, y: maxBottom, w: 12, h: 6, minW: 2, minH: 2 };
          return {
            ...page,
            panels: [...page.panels, newPanel],
            layout: [...page.layout, newLayoutItem],
          };
        });
        queueSave(next);
        if (wasEmpty) onActivePageIdChange(targetId);
        return next;
      });
      return createdId;
    },
    [activePageId, onActivePageIdChange, queueSave, setLocalPages],
  );

  const handleDeletePanel = useCallback(
    (id: string) => {
      withActivePage((page) => ({
        ...page,
        panels: page.panels.filter((p) => p.id !== id),
        layout: page.layout.filter((l) => l.i !== id),
      }));
    },
    [withActivePage],
  );

  const handlePanelContentChange = useCallback(
    (id: string, content: Record<string, unknown>) => {
      withActivePage((page) => ({
        ...page,
        panels: page.panels.map((p) => {
          if (p.id !== id) return p;
          const nextType = migratedPanelType(p.type, content);
          return { ...p, type: nextType, content };
        }),
      }));
    },
    [withActivePage],
  );

  const handleLayoutChange = useCallback(
    (nextLayout: ConsoleLayoutItem[]) => {
      withActivePage((page) => {
        const validIds = new Set(page.panels.map((p) => p.id));
        const filtered = nextLayout.filter((item) => validIds.has(item.i));
        if (layoutsEqual(page.layout, filtered)) return page;
        return { ...page, layout: filtered };
      });
    },
    [withActivePage],
  );

  return { handleAddPanel, handleDeletePanel, handlePanelContentChange, handleLayoutChange };
}

function usePageHandlers({
  setLocalPages,
  queueSave,
  activePageId,
  onActivePageIdChange,
}: {
  setLocalPages: React.Dispatch<React.SetStateAction<ConsolePage[]>>;
  queueSave: (next: ConsolePage[]) => void;
  activePageId: string | null;
  onActivePageIdChange: (id: string) => void;
}) {
  const handleAddPage = useCallback(() => {
    setLocalPages((prev) => {
      // Enforce the cap in local state as well as in the tab-strip
      // button. Rapid successive clicks and direct calls (e.g. from
      // tests or keyboard shortcuts) can bypass the disabled button
      // between renders; without this guard, staging + the local
      // editor state can drift past MAX_CONSOLE_PAGES and the commit
      // then fails while the editor still shows an invalid sixth tab.
      if (prev.length >= MAX_CONSOLE_PAGES) return prev;
      const pagesForCreation = appendConsolePage(prev);
      queueSave(pagesForCreation);
      onActivePageIdChange(pagesForCreation[pagesForCreation.length - 1]!.id);
      return pagesForCreation;
    });
  }, [onActivePageIdChange, queueSave, setLocalPages]);

  const handleRenamePage = useCallback(
    (pageId: string, name: string) => {
      setLocalPages((prev) => {
        const next = prev.map((page) => (page.id === pageId ? { ...page, name } : page));
        queueSave(next);
        return next;
      });
    },
    [queueSave, setLocalPages],
  );

  const handleReorderPages = useCallback(
    (fromIndex: number, toIndex: number) => {
      setLocalPages((prev) => {
        if (fromIndex < 0 || toIndex < 0 || fromIndex >= prev.length || toIndex >= prev.length) return prev;
        if (fromIndex === toIndex) return prev;
        const next = prev.slice();
        const [moved] = next.splice(fromIndex, 1);
        next.splice(toIndex, 0, moved!);
        queueSave(next);
        return next;
      });
    },
    [queueSave, setLocalPages],
  );

  const handleRemovePage = useCallback(
    (pageId: string) => {
      setLocalPages((prev) => {
        if (prev.length <= 1) return prev;
        const index = prev.findIndex((page) => page.id === pageId);
        if (index < 0) return prev;
        const next = prev.filter((page) => page.id !== pageId);
        queueSave(next);
        if (activePageId === pageId) {
          const fallback = next[Math.max(0, index - 1)] ?? next[0]!;
          onActivePageIdChange(fallback.id);
        }
        return next;
      });
    },
    [activePageId, onActivePageIdChange, queueSave, setLocalPages],
  );

  return { handleAddPage, handleRenamePage, handleReorderPages, handleRemovePage };
}

function appendConsolePage(prev: ConsolePage[]): ConsolePage[] {
  const baseId = "page";
  const takenIds = new Set(prev.map((page) => page.id));
  let index = prev.length + 1;
  let candidateId = `${baseId}-${index}`;
  while (takenIds.has(candidateId)) {
    index += 1;
    candidateId = `${baseId}-${index}`;
  }
  const newPage: ConsolePage = { id: candidateId, name: `Page ${index}`, panels: [], layout: [] };
  // When the console was empty, adopt the default id so the first added
  // page keeps the same on-disk shape a single-page console has always had.
  if (prev.length === 0) {
    return [{ ...newPage, id: DEFAULT_CONSOLE_PAGE_ID, name: DEFAULT_CONSOLE_PAGE_NAME }];
  }
  return [...prev, newPage];
}

function panelBaseIdFromName(name: string, type: PanelType): string {
  const slug = name
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
  return slug || `${type}-${Math.random().toString(36).slice(2, 8)}`;
}

function layoutsEqual(previous: ConsoleLayoutItem[], next: ConsoleLayoutItem[]): boolean {
  if (previous.length !== next.length) return false;
  return next.every((item, index) => {
    const before = previous[index];
    if (!before || before.i !== item.i) return false;
    return (
      before.x === item.x &&
      before.y === item.y &&
      before.w === item.w &&
      before.h === item.h &&
      before.minW === item.minW &&
      before.minH === item.minH
    );
  });
}

function migratedPanelType(currentType: string, content: Record<string, unknown>): string {
  if (currentType === "node" && Array.isArray(content.nodes)) return "nodes";
  return currentType;
}

function uniquePanelId(panels: ConsolePanel[], base: string): string {
  const taken = new Set(panels.map((p) => p.id));
  if (!taken.has(base)) return base;
  for (let i = 2; i < 1000; i += 1) {
    const candidate = `${base}-${i}`;
    if (!taken.has(candidate)) return candidate;
  }
  return `${base}-${Math.random().toString(36).slice(2, 8)}`;
}
