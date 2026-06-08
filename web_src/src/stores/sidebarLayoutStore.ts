import { useEffect } from "react";
import { create } from "zustand";

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
 * Persistence keys are kept identical to the pre-refactor implementation so
 * existing local storage values are honored.
 */

export const SIDEBAR_MIN_WIDTH = 300;
export const MIDDLE_MIN_WIDTH = 220;

const LEFT_STORAGE_KEY = "agent-sidebar-width";
const RIGHT_STORAGE_KEY = "componentSidebarWidth";

const DEFAULT_LEFT_WIDTH = 380;
const DEFAULT_RIGHT_WIDTH = 380;
const FALLBACK_VIEWPORT = 1280;

function getViewportWidth(): number {
  return typeof window === "undefined" ? FALLBACK_VIEWPORT : window.innerWidth;
}

function readPersistedWidth(key: string, fallback: number): number {
  if (typeof window === "undefined") return fallback;
  try {
    const saved = window.localStorage.getItem(key);
    const parsed = saved ? Number.parseInt(saved, 10) : Number.NaN;
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(SIDEBAR_MIN_WIDTH, parsed);
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

interface SidebarLayoutState {
  leftWidth: number;
  rightWidth: number;
  leftMountCount: number;
  rightMountCount: number;
  isLeftResizing: boolean;
  isRightResizing: boolean;

  registerLeft: () => () => void;
  registerRight: () => () => void;
  setLeftResizing: (resizing: boolean) => void;
  setRightResizing: (resizing: boolean) => void;

  /**
   * Drive the left sidebar to {@link target} px. Pushes the right sidebar
   * down to {@link SIDEBAR_MIN_WIDTH} when needed; will not violate the
   * middle-section minimum.
   */
  resizeLeft: (target: number) => void;
  /**
   * Mirrors {@link resizeLeft} for the right sidebar.
   */
  resizeRight: (target: number) => void;

  /**
   * Recompute widths against the current viewport, shrinking sidebars (right
   * first, then left) only if necessary. Called on window resize.
   */
  recomputeForViewport: () => void;

  /**
   * Re-read both widths from localStorage and reset transient flags. Useful
   * from tests that mutate storage between renders.
   */
  hydrateFromStorage: () => void;
}

function leftIsMounted(state: SidebarLayoutState): boolean {
  return state.leftMountCount > 0;
}

function rightIsMounted(state: SidebarLayoutState): boolean {
  return state.rightMountCount > 0;
}

export const useSidebarLayoutStore = create<SidebarLayoutState>((set, get) => ({
  leftWidth: readPersistedWidth(LEFT_STORAGE_KEY, DEFAULT_LEFT_WIDTH),
  rightWidth: readPersistedWidth(RIGHT_STORAGE_KEY, DEFAULT_RIGHT_WIDTH),
  leftMountCount: 0,
  rightMountCount: 0,
  isLeftResizing: false,
  isRightResizing: false,

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

  setLeftResizing: (resizing) => set({ isLeftResizing: resizing }),
  setRightResizing: (resizing) => set({ isRightResizing: resizing }),

  resizeLeft: (target) => {
    const state = get();
    const viewport = getViewportWidth();
    const otherMounted = rightIsMounted(state);

    const otherFloor = otherMounted ? SIDEBAR_MIN_WIDTH : 0;
    const maxLeft = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - otherFloor);
    const nextLeft = Math.max(SIDEBAR_MIN_WIDTH, Math.min(maxLeft, Math.round(target)));

    let nextRight = state.rightWidth;
    if (otherMounted) {
      const allowedRight = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - nextLeft);
      if (state.rightWidth > allowedRight) nextRight = allowedRight;
    }

    if (nextLeft === state.leftWidth && nextRight === state.rightWidth) return;

    persistWidth(LEFT_STORAGE_KEY, nextLeft);
    if (nextRight !== state.rightWidth) persistWidth(RIGHT_STORAGE_KEY, nextRight);
    set({ leftWidth: nextLeft, rightWidth: nextRight });
  },

  resizeRight: (target) => {
    const state = get();
    const viewport = getViewportWidth();
    const otherMounted = leftIsMounted(state);

    const otherFloor = otherMounted ? SIDEBAR_MIN_WIDTH : 0;
    const maxRight = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - otherFloor);
    const nextRight = Math.max(SIDEBAR_MIN_WIDTH, Math.min(maxRight, Math.round(target)));

    let nextLeft = state.leftWidth;
    if (otherMounted) {
      const allowedLeft = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - nextRight);
      if (state.leftWidth > allowedLeft) nextLeft = allowedLeft;
    }

    if (nextRight === state.rightWidth && nextLeft === state.leftWidth) return;

    persistWidth(RIGHT_STORAGE_KEY, nextRight);
    if (nextLeft !== state.leftWidth) persistWidth(LEFT_STORAGE_KEY, nextLeft);
    set({ leftWidth: nextLeft, rightWidth: nextRight });
  },

  hydrateFromStorage: () => {
    set({
      leftWidth: readPersistedWidth(LEFT_STORAGE_KEY, DEFAULT_LEFT_WIDTH),
      rightWidth: readPersistedWidth(RIGHT_STORAGE_KEY, DEFAULT_RIGHT_WIDTH),
      leftMountCount: 0,
      rightMountCount: 0,
      isLeftResizing: false,
      isRightResizing: false,
    });
  },

  recomputeForViewport: () => {
    const state = get();
    const viewport = getViewportWidth();
    const leftActive = leftIsMounted(state);
    const rightActive = rightIsMounted(state);
    const effectiveLeft = leftActive ? state.leftWidth : 0;
    const effectiveRight = rightActive ? state.rightWidth : 0;

    if (effectiveLeft + effectiveRight + MIDDLE_MIN_WIDTH <= viewport) return;

    let nextLeft = state.leftWidth;
    let nextRight = state.rightWidth;

    // Shrink the right sidebar first since it overlays the canvas.
    if (rightActive) {
      const cap = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - (leftActive ? nextLeft : 0));
      if (nextRight > cap) nextRight = cap;
    }
    if (leftActive) {
      const cap = Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - (rightActive ? nextRight : 0));
      if (nextLeft > cap) nextLeft = cap;
    }

    if (nextLeft === state.leftWidth && nextRight === state.rightWidth) return;
    if (nextLeft !== state.leftWidth) persistWidth(LEFT_STORAGE_KEY, nextLeft);
    if (nextRight !== state.rightWidth) persistWidth(RIGHT_STORAGE_KEY, nextRight);
    set({ leftWidth: nextLeft, rightWidth: nextRight });
  },
}));

/**
 * Subscribe a sidebar's mount lifecycle to the layout store. Use `"left"` from
 * the canvas tool sidebar and `"right"` from each of the right-side sidebars
 * (component / building-blocks). When unmounted the store decrements the mount
 * count so its constraints reflect the actual on-screen geometry.
 */
export function useSidebarMount(side: "left" | "right"): void {
  const registerLeft = useSidebarLayoutStore((s) => s.registerLeft);
  const registerRight = useSidebarLayoutStore((s) => s.registerRight);
  useEffect(() => (side === "left" ? registerLeft() : registerRight()), [side, registerLeft, registerRight]);
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
  const leftMounted = useSidebarLayoutStore((state) => state.leftMountCount > 0);
  return leftMounted ? leftWidth : 0;
}
