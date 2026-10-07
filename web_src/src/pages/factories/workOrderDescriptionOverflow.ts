export const FALLBACK_COLLAPSED_MAX_HEIGHT_PX = 220;
export const MIN_COLLAPSED_DESCRIPTION_HEIGHT_PX = 100;
export const DESCRIPTION_TOGGLE_RESERVE_PX = 32;
const HEIGHT_SLACK_PX = 4;

export type DescriptionPaneMetrics = {
  clientHeight: number;
  paddingTop: number;
  paddingBottom: number;
};

export function descriptionPaneCapacity(pane: DescriptionPaneMetrics | null): number {
  if (!pane || pane.clientHeight <= 0) {
    return FALLBACK_COLLAPSED_MAX_HEIGHT_PX;
  }

  return Math.max(0, pane.clientHeight - pane.paddingTop - pane.paddingBottom);
}

export function descriptionLeftoverCapacity(paneCapacity: number, reservedHeight: number): number {
  const leftover = paneCapacity - reservedHeight;
  if (leftover <= 0) {
    return FALLBACK_COLLAPSED_MAX_HEIGHT_PX;
  }

  return leftover;
}

export function descriptionNeedsCollapse(contentHeight: number, paneCapacity: number): boolean {
  return contentHeight > Math.max(paneCapacity, MIN_COLLAPSED_DESCRIPTION_HEIGHT_PX) + HEIGHT_SLACK_PX;
}

export function collapsedDescriptionMaxHeight(paneCapacity: number): number {
  return Math.max(MIN_COLLAPSED_DESCRIPTION_HEIGHT_PX, paneCapacity - DESCRIPTION_TOGGLE_RESERVE_PX);
}

export function nearestScrollParent(element: HTMLElement): HTMLElement | null {
  let parent = element.parentElement;
  while (parent) {
    const overflowY = getComputedStyle(parent).overflowY;
    if (overflowY === "auto" || overflowY === "scroll") {
      return parent;
    }
    parent = parent.parentElement;
  }

  return null;
}

function paneBlockFor(content: HTMLElement, pane: HTMLElement): HTMLElement | null {
  let current: HTMLElement | null = content;
  while (current && current.parentElement !== pane) {
    current = current.parentElement;
  }

  return current;
}

function outerHeight(element: HTMLElement): number {
  const style = getComputedStyle(element);
  return (
    element.offsetHeight + (Number.parseFloat(style.marginTop) || 0) + (Number.parseFloat(style.marginBottom) || 0)
  );
}

export function reservedPaneSiblingHeight(content: HTMLElement, pane: HTMLElement): number {
  const block = paneBlockFor(content, pane);
  let reserved = 0;
  for (const child of Array.from(pane.children)) {
    if (!(child instanceof HTMLElement) || child === block) {
      continue;
    }
    reserved += outerHeight(child);
  }

  return reserved;
}

export function readScrollPaneMetrics(element: HTMLElement): DescriptionPaneMetrics | null {
  const pane = nearestScrollParent(element);
  if (!pane) {
    return null;
  }

  const style = getComputedStyle(pane);
  return {
    clientHeight: pane.clientHeight,
    paddingTop: Number.parseFloat(style.paddingTop) || 0,
    paddingBottom: Number.parseFloat(style.paddingBottom) || 0,
  };
}
