export const SIDEBAR_MIN_WIDTH = 300;
/** Runs and versions can be narrower than the component sidebar. */
export const AUX_SIDEBAR_MIN_WIDTH = 240;
export const MIDDLE_MIN_WIDTH = 220;

export interface SidebarLayoutSnapshot {
  rightWidth: number;
  auxLeftWidth: number;
  rightMountCount: number;
  auxLeftMountCount: number;
}

function sidebarCap(viewport: number, reservedWidth: number): number {
  return Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - reservedWidth);
}

export function computeResizeRight(state: SidebarLayoutSnapshot, target: number, viewport: number) {
  const reservedWidth = state.auxLeftMountCount > 0 ? state.auxLeftWidth : 0;
  const nextRight = Math.max(SIDEBAR_MIN_WIDTH, Math.min(sidebarCap(viewport, reservedWidth), Math.round(target)));
  return { nextRight, nextAuxLeft: state.auxLeftWidth };
}

export function computeResizeAuxLeft(state: SidebarLayoutSnapshot, target: number, viewport: number) {
  const reservedWidth = state.rightMountCount > 0 ? state.rightWidth : 0;
  const nextAuxLeft = Math.max(
    AUX_SIDEBAR_MIN_WIDTH,
    Math.min(sidebarCap(viewport, reservedWidth), Math.round(target)),
  );
  const nextRight =
    state.rightMountCount > 0 ? Math.min(state.rightWidth, sidebarCap(viewport, nextAuxLeft)) : state.rightWidth;
  return { nextRight, nextAuxLeft };
}

export function computeRecomputeForViewport(state: SidebarLayoutSnapshot, viewport: number) {
  const rightActive = state.rightMountCount > 0;
  const auxActive = state.auxLeftMountCount > 0;
  const nextRight = rightActive
    ? Math.min(state.rightWidth, sidebarCap(viewport, auxActive ? state.auxLeftWidth : 0))
    : state.rightWidth;
  const nextAuxLeft = auxActive
    ? Math.min(state.auxLeftWidth, sidebarCap(viewport, rightActive ? nextRight : 0))
    : state.auxLeftWidth;
  return { nextRight, nextAuxLeft, changed: nextRight !== state.rightWidth || nextAuxLeft !== state.auxLeftWidth };
}
