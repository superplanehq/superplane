import { useEffect } from "react";
import { create } from "zustand";
import {
  AUX_SIDEBAR_MIN_WIDTH,
  computeRecomputeForViewport,
  computeResizeAuxLeft,
  computeResizeLeft,
  computeResizeRight,
  SIDEBAR_MIN_WIDTH,
  type SidebarLayoutSnapshot,
} from "./sidebarLayoutConstraints";

/**
 * Shared layout store for the canvas page's left and right sidebars.
 *
 * Both sidebars enforce a minimum width and the middle (canvas) section is
 * never allowed to shrink below {@link MIDDLE_MIN_WIDTH}. When the user drags
 * one sidebar past the point where its growth would violate that, this store
 * pushes the opposite sidebar down to {@link SIDEBAR_MIN_WIDTH} before
 * refusing to grow further. Dragging back does NOT restore the pushed
 * sidebar — that is intentional, matching how most resizable splitters work.
 *
 * Runs/Versions panels register as an auxiliary left sidebar so their combined
 * width with the agent sidebar respects the same canvas minimum.
 *
 * Persistence keys are kept identical to the pre-refactor implementation so
 * existing local storage values are honored.
 */

export { AUX_SIDEBAR_MIN_WIDTH, MIDDLE_MIN_WIDTH, SIDEBAR_MIN_WIDTH } from "./sidebarLayoutConstraints";

const LEFT_STORAGE_KEY = "agent-sidebar-width";
const RIGHT_STORAGE_KEY = "componentSidebarWidth";

const DEFAULT_LEFT_WIDTH = 380;
const DEFAULT_RIGHT_WIDTH = 380;
const FALLBACK_VIEWPORT = 1280;

function getViewportWidth(): number {
  return typeof window === "undefined" ? FALLBACK_VIEWPORT : window.innerWidth;
}

function readPersistedWidth(key: string, fallback: number, minWidth = SIDEBAR_MIN_WIDTH): number {
  if (typeof window === "undefined") return fallback;
  try {
    const saved = window.localStorage.getItem(key);
    const parsed = saved ? Number.parseInt(saved, 10) : Number.NaN;
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(minWidth, parsed);
  } catch {
    return fallback;
  }
}

function persistWidth(key: string, value: number): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, String(value));
  } catch {
    // ignore storage errors (private mode, quota, etc.)
  }
}

interface SidebarLayoutState extends SidebarLayoutSnapshot {
  auxLeftStorageKey: string | null;
  isLeftResizing: boolean;
  isRightResizing: boolean;
  isAuxLeftResizing: boolean;

  registerLeft: () => () => void;
  registerRight: () => () => void;
  registerAuxLeft: (storageKey: string, defaultWidth: number) => () => void;
  setLeftResizing: (resizing: boolean) => void;
  setRightResizing: (resizing: boolean) => void;
  setAuxLeftResizing: (resizing: boolean) => void;
  resizeLeft: (target: number) => void;
  resizeRight: (target: number) => void;
  resizeAuxLeft: (target: number) => void;
  recomputeForViewport: () => void;
  hydrateFromStorage: () => void;
}

function applyWidthUpdate(
  state: SidebarLayoutState,
  set: (partial: Partial<SidebarLayoutState>) => void,
  next: { nextLeft: number; nextRight: number; nextAuxLeft: number },
) {
  if (
    next.nextLeft === state.leftWidth &&
    next.nextRight === state.rightWidth &&
    next.nextAuxLeft === state.auxLeftWidth
  ) {
    return;
  }

  if (next.nextLeft !== state.leftWidth) persistWidth(LEFT_STORAGE_KEY, next.nextLeft);
  if (next.nextRight !== state.rightWidth) persistWidth(RIGHT_STORAGE_KEY, next.nextRight);
  if (next.nextAuxLeft !== state.auxLeftWidth && state.auxLeftStorageKey) {
    persistWidth(state.auxLeftStorageKey, next.nextAuxLeft);
  }

  set({ leftWidth: next.nextLeft, rightWidth: next.nextRight, auxLeftWidth: next.nextAuxLeft });
}

export const useSidebarLayoutStore = create<SidebarLayoutState>((set, get) => ({
  leftWidth: readPersistedWidth(LEFT_STORAGE_KEY, DEFAULT_LEFT_WIDTH),
  rightWidth: readPersistedWidth(RIGHT_STORAGE_KEY, DEFAULT_RIGHT_WIDTH),
  auxLeftWidth: DEFAULT_LEFT_WIDTH,
  auxLeftStorageKey: null,
  leftMountCount: 0,
  rightMountCount: 0,
  auxLeftMountCount: 0,
  isLeftResizing: false,
  isRightResizing: false,
  isAuxLeftResizing: false,

  registerLeft: () => {
    set((state) => ({ leftMountCount: state.leftMountCount + 1 }));
    get().recomputeForViewport();
    return () => set((state) => ({ leftMountCount: Math.max(0, state.leftMountCount - 1) }));
  },

  registerRight: () => {
    set((state) => ({ rightMountCount: state.rightMountCount + 1 }));
    get().recomputeForViewport();
    return () => set((state) => ({ rightMountCount: Math.max(0, state.rightMountCount - 1) }));
  },

  registerAuxLeft: (storageKey, defaultWidth) => {
    set((state) => ({
      auxLeftMountCount: state.auxLeftMountCount + 1,
      auxLeftStorageKey: storageKey,
      auxLeftWidth: readPersistedWidth(storageKey, defaultWidth, AUX_SIDEBAR_MIN_WIDTH),
    }));
    get().recomputeForViewport();
    return () =>
      set((state) => ({
        auxLeftMountCount: Math.max(0, state.auxLeftMountCount - 1),
        auxLeftStorageKey: state.auxLeftMountCount <= 1 ? null : state.auxLeftStorageKey,
      }));
  },

  setLeftResizing: (resizing) => set({ isLeftResizing: resizing }),
  setRightResizing: (resizing) => set({ isRightResizing: resizing }),
  setAuxLeftResizing: (resizing) => set({ isAuxLeftResizing: resizing }),

  resizeLeft: (target) => {
    const state = get();
    applyWidthUpdate(state, set, computeResizeLeft(state, target, getViewportWidth()));
  },

  resizeRight: (target) => {
    const state = get();
    applyWidthUpdate(state, set, computeResizeRight(state, target, getViewportWidth()));
  },

  resizeAuxLeft: (target) => {
    const state = get();
    applyWidthUpdate(state, set, computeResizeAuxLeft(state, target, getViewportWidth()));
  },

  hydrateFromStorage: () => {
    set({
      leftWidth: readPersistedWidth(LEFT_STORAGE_KEY, DEFAULT_LEFT_WIDTH),
      rightWidth: readPersistedWidth(RIGHT_STORAGE_KEY, DEFAULT_RIGHT_WIDTH),
      auxLeftWidth: DEFAULT_LEFT_WIDTH,
      auxLeftStorageKey: null,
      leftMountCount: 0,
      rightMountCount: 0,
      auxLeftMountCount: 0,
      isLeftResizing: false,
      isRightResizing: false,
      isAuxLeftResizing: false,
    });
  },

  recomputeForViewport: () => {
    const state = get();
    const result = computeRecomputeForViewport(state, getViewportWidth());
    if (!result.changed) return;
    applyWidthUpdate(state, set, result);
  },
}));

/**
 * Subscribe a sidebar's mount lifecycle to the layout store. Use `"left"` from
 * the canvas tool sidebar and `"right"` from each of the right-side sidebars
 * (component / building-blocks). When unmounted the store decrements the mount
 * count so its constraints reflect the actual on-screen geometry.
 */
export function useSidebarMount(side: "left" | "right", active = true): void {
  const registerLeft = useSidebarLayoutStore((s) => s.registerLeft);
  const registerRight = useSidebarLayoutStore((s) => s.registerRight);
  useEffect(() => {
    if (!active) return;
    return side === "left" ? registerLeft() : registerRight();
  }, [active, side, registerLeft, registerRight]);
}

/**
 * Subscribe the runs/versions auxiliary sidebar to the layout store. Only
 * registers while {@link isOpen} is true so closed panels do not consume width.
 */
export function useAuxLeftSidebarMount(isOpen: boolean, storageKey: string, defaultWidth: number): void {
  const registerAuxLeft = useSidebarLayoutStore((s) => s.registerAuxLeft);
  useEffect(() => {
    if (!isOpen) return;
    return registerAuxLeft(storageKey, defaultWidth);
  }, [isOpen, storageKey, defaultWidth, registerAuxLeft]);
}

/**
 * Recompute widths whenever the viewport changes so the middle-section
 * minimum is honored even if the user resizes their browser window.
 */
export function useSidebarLayoutViewport(): void {
  const recompute = useSidebarLayoutStore((s) => s.recomputeForViewport);
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = () => recompute();
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, [recompute]);
}

/**
 * Horizontal inset for absolutely positioned canvas overlays (Console, Memory)
 * so they sit beside the tool sidebar instead of underneath it.
 */
export function useEffectiveLeftSidebarWidth(): number {
  const leftWidth = useSidebarLayoutStore((state) => state.leftWidth);
  const auxLeftWidth = useSidebarLayoutStore((state) => state.auxLeftWidth);
  const leftMounted = useSidebarLayoutStore((state) => state.leftMountCount > 0);
  const auxLeftMounted = useSidebarLayoutStore((state) => state.auxLeftMountCount > 0);
  return (leftMounted ? leftWidth : 0) + (auxLeftMounted ? auxLeftWidth : 0);
}
