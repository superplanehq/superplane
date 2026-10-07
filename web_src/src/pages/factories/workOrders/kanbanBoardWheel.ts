/**
 * Lane lists use overflow-y: auto, which captures the wheel even when the
 * list does not overflow. The board then shows an x scrollbar that never
 * moves. Redirect vertical wheel to the board when no child can scroll y.
 */
export function shouldRedirectWheelToHorizontalScroll(
  event: Pick<WheelEvent, "deltaX" | "deltaY" | "target">,
  board: HTMLElement,
): boolean {
  if (board.scrollWidth <= board.clientWidth + 1) {
    return false;
  }
  if (Math.abs(event.deltaX) > Math.abs(event.deltaY) || event.deltaY === 0) {
    return false;
  }
  return !verticalAncestorCanConsumeDelta(event.target, board, event.deltaY);
}

function verticalAncestorCanConsumeDelta(target: EventTarget | null, board: HTMLElement, deltaY: number): boolean {
  let node = target instanceof Element ? target : null;
  while (node && node !== board) {
    if (node instanceof HTMLElement && canScrollVertically(node, deltaY)) {
      return true;
    }
    node = node.parentElement;
  }
  return false;
}

function canScrollVertically(node: HTMLElement, deltaY: number): boolean {
  const overflowY = getComputedStyle(node).overflowY;
  if (overflowY !== "auto" && overflowY !== "scroll") {
    return false;
  }
  if (node.scrollHeight <= node.clientHeight + 1) {
    return false;
  }
  if (deltaY < 0) {
    return node.scrollTop > 0;
  }
  return node.scrollTop + node.clientHeight < node.scrollHeight - 1;
}
