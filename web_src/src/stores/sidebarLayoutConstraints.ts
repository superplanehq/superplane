export const SIDEBAR_MIN_WIDTH = 300;
/** Runs/Versions auxiliary sidebar can be narrower than the agent or component sidebars. */
export const AUX_SIDEBAR_MIN_WIDTH = 240;
export const MIDDLE_MIN_WIDTH = 220;

export interface SidebarLayoutSnapshot {
  leftWidth: number;
  rightWidth: number;
  auxLeftWidth: number;
  leftMountCount: number;
  rightMountCount: number;
  auxLeftMountCount: number;
}

function leftIsMounted(state: SidebarLayoutSnapshot): boolean {
  return state.leftMountCount > 0;
}

function rightIsMounted(state: SidebarLayoutSnapshot): boolean {
  return state.rightMountCount > 0;
}

function auxLeftIsMounted(state: SidebarLayoutSnapshot): boolean {
  return state.auxLeftMountCount > 0;
}

function shrinkLeftPanelsToFit(
  leftMounted: boolean,
  auxMounted: boolean,
  leftWidth: number,
  auxLeftWidth: number,
  allowedLeftTotal: number,
): { leftWidth: number; auxLeftWidth: number } {
  const currentLeftTotal = (leftMounted ? leftWidth : 0) + (auxMounted ? auxLeftWidth : 0);
  if (currentLeftTotal <= allowedLeftTotal) {
    return { leftWidth, auxLeftWidth };
  }

  let nextLeft = leftWidth;
  let nextAux = auxLeftWidth;
  let overflow = currentLeftTotal - allowedLeftTotal;

  if (auxMounted && auxLeftWidth > AUX_SIDEBAR_MIN_WIDTH) {
    const auxShrink = Math.min(overflow, auxLeftWidth - AUX_SIDEBAR_MIN_WIDTH);
    nextAux = auxLeftWidth - auxShrink;
    overflow -= auxShrink;
  }

  if (overflow > 0 && leftMounted && leftWidth > SIDEBAR_MIN_WIDTH) {
    nextLeft = Math.max(SIDEBAR_MIN_WIDTH, leftWidth - overflow);
  }

  return { leftWidth: nextLeft, auxLeftWidth: nextAux };
}

function sidebarCap(viewport: number, reservedWidth: number): number {
  return Math.max(SIDEBAR_MIN_WIDTH, viewport - MIDDLE_MIN_WIDTH - reservedWidth);
}

export function computeResizeLeft(state: SidebarLayoutSnapshot, target: number, viewport: number) {
  const otherMounted = rightIsMounted(state);
  const auxMounted = auxLeftIsMounted(state);
  const auxFloor = auxMounted ? state.auxLeftWidth : 0;
  const otherFloor = otherMounted ? SIDEBAR_MIN_WIDTH : 0;
  const maxLeft = sidebarCap(viewport, otherFloor + auxFloor);
  const nextLeft = Math.max(SIDEBAR_MIN_WIDTH, Math.min(maxLeft, Math.round(target)));

  let nextRight = state.rightWidth;
  if (otherMounted) {
    const allowedRight = sidebarCap(viewport, nextLeft + auxFloor);
    if (state.rightWidth > allowedRight) nextRight = allowedRight;
  }

  return { nextLeft, nextRight, nextAuxLeft: state.auxLeftWidth };
}

export function computeResizeRight(state: SidebarLayoutSnapshot, target: number, viewport: number) {
  const leftMounted = leftIsMounted(state);
  const auxMounted = auxLeftIsMounted(state);
  const leftTotal = (leftMounted ? state.leftWidth : 0) + (auxMounted ? state.auxLeftWidth : 0);
  const maxRight = sidebarCap(viewport, leftTotal);
  const nextRight = Math.max(SIDEBAR_MIN_WIDTH, Math.min(maxRight, Math.round(target)));

  if (!leftMounted && !auxMounted) {
    return { nextLeft: state.leftWidth, nextRight, nextAuxLeft: state.auxLeftWidth };
  }

  const minLeftTotal = leftMounted && auxMounted ? SIDEBAR_MIN_WIDTH * 2 : SIDEBAR_MIN_WIDTH;
  const allowedLeftTotal = Math.max(minLeftTotal, viewport - MIDDLE_MIN_WIDTH - nextRight);
  const shrunk = shrinkLeftPanelsToFit(leftMounted, auxMounted, state.leftWidth, state.auxLeftWidth, allowedLeftTotal);

  return { nextLeft: shrunk.leftWidth, nextRight, nextAuxLeft: shrunk.auxLeftWidth };
}

export function computeResizeAuxLeft(state: SidebarLayoutSnapshot, target: number, viewport: number) {
  const leftMounted = leftIsMounted(state);
  const rightMounted = rightIsMounted(state);
  const leftFloor = leftMounted ? state.leftWidth : 0;
  const maxAux = sidebarCap(viewport, leftFloor + (rightMounted ? state.rightWidth : 0));
  const nextAux = Math.max(AUX_SIDEBAR_MIN_WIDTH, Math.min(maxAux, Math.round(target)));

  let nextRight = state.rightWidth;
  if (rightMounted) {
    const allowedRight = sidebarCap(viewport, leftFloor + nextAux);
    if (state.rightWidth > allowedRight) nextRight = allowedRight;
  }

  let nextLeft = state.leftWidth;
  if (leftMounted) {
    const allowedLeft = sidebarCap(viewport, nextAux + (rightMounted ? nextRight : 0));
    if (state.leftWidth > allowedLeft) nextLeft = allowedLeft;
  }

  return { nextLeft, nextRight, nextAuxLeft: nextAux };
}

function mountedLeftWidth(leftActive: boolean, left: number, auxActive: boolean, aux: number): number {
  return (leftActive ? left : 0) + (auxActive ? aux : 0);
}

function capSidebarWidth(active: boolean, current: number, cap: number): number {
  return active && current > cap ? cap : current;
}

export function computeRecomputeForViewport(state: SidebarLayoutSnapshot, viewport: number) {
  const leftActive = leftIsMounted(state);
  const rightActive = rightIsMounted(state);
  const auxActive = auxLeftIsMounted(state);

  let nextLeft = state.leftWidth;
  let nextRight = state.rightWidth;
  let nextAux = state.auxLeftWidth;

  const mountedTotal = mountedLeftWidth(leftActive, nextLeft, auxActive, nextAux) + (rightActive ? nextRight : 0);
  if (mountedTotal + MIDDLE_MIN_WIDTH <= viewport) {
    return { nextLeft, nextRight, nextAuxLeft: nextAux, changed: false };
  }

  nextRight = capSidebarWidth(
    rightActive,
    nextRight,
    sidebarCap(viewport, mountedLeftWidth(leftActive, nextLeft, auxActive, nextAux)),
  );
  nextAux = capSidebarWidth(
    auxActive,
    nextAux,
    sidebarCap(viewport, (leftActive ? nextLeft : 0) + (rightActive ? nextRight : 0)),
  );
  nextLeft = capSidebarWidth(
    leftActive,
    nextLeft,
    sidebarCap(viewport, (auxActive ? nextAux : 0) + (rightActive ? nextRight : 0)),
  );

  const changed = nextLeft !== state.leftWidth || nextRight !== state.rightWidth || nextAux !== state.auxLeftWidth;

  return { nextLeft, nextRight, nextAuxLeft: nextAux, changed };
}
