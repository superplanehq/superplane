import { arrayMove } from "@dnd-kit/sortable";

/**
 * A manual reorder expressed as the two neighbors the moved task should end
 * up between (see the `ReorderWorkOrder` API). Either neighbor is omitted
 * when the task moves to that end of its column.
 */
export interface WorkOrderReorderMove {
  orderId: string;
  previousOrderId?: string;
  nextOrderId?: string;
}

/**
 * Turns a drag-end inside one board column into a `WorkOrderReorderMove`.
 * `laneIds` is that column's card order before the move; `activeId` is the
 * dragged card and `overId` the card it was dropped on. Returns `null` when
 * the drop doesn't change anything (dropped on itself, or either id isn't
 * in this column — the caller already keeps drags inside a single column,
 * this is a defensive fallback).
 */
export function buildWorkOrderReorderMove(
  laneIds: string[],
  activeId: string,
  overId: string,
): WorkOrderReorderMove | null {
  if (activeId === overId) {
    return null;
  }
  const fromIndex = laneIds.indexOf(activeId);
  const toIndex = laneIds.indexOf(overId);
  if (fromIndex === -1 || toIndex === -1) {
    return null;
  }

  const reordered = arrayMove(laneIds, fromIndex, toIndex);
  const movedIndex = reordered.indexOf(activeId);
  return {
    orderId: activeId,
    previousOrderId: reordered[movedIndex - 1],
    nextOrderId: reordered[movedIndex + 1],
  };
}

/**
 * Applies a `WorkOrderReorderMove` to a raw, position-ordered list for
 * optimistic UI updates — moving `orderId` next to its new neighbors
 * without waiting for the server response. A no-op when `orderId` isn't in
 * `items` (stale cache, or the mutation already landed).
 */
export function applyWorkOrderReorderMove<T extends { id?: string }>(items: T[], move: WorkOrderReorderMove): T[] {
  const target = items.find((item) => item.id === move.orderId);
  if (!target) {
    return items;
  }

  const withoutTarget = items.filter((item) => item.id !== move.orderId);
  const insertAt = reorderInsertIndex(withoutTarget, move);

  const next = withoutTarget.slice();
  next.splice(insertAt, 0, target);
  return next;
}

function reorderInsertIndex<T extends { id?: string }>(withoutTarget: T[], move: WorkOrderReorderMove): number {
  if (move.previousOrderId) {
    const index = withoutTarget.findIndex((item) => item.id === move.previousOrderId);
    return index === -1 ? withoutTarget.length : index + 1;
  }
  if (move.nextOrderId) {
    const index = withoutTarget.findIndex((item) => item.id === move.nextOrderId);
    return index === -1 ? 0 : index;
  }
  return 0;
}
