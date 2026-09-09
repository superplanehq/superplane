import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToParentElement, restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { cn } from "@/lib/utils";
import { useCallback } from "react";
import { buildWorkOrderReorderMove, type WorkOrderReorderMove } from "../lib/workOrderReorder";
import {
  groupWorkOrderEntriesByLane,
  type WorkOrderListEntry,
  type WorkOrderOrdering,
} from "../lib/workOrderListModel";
import {
  WORK_ORDER_BOARD_LANES,
  type WorkOrderBoardLaneDefinition,
  type WorkOrderBoardLaneId,
} from "../lib/workOrderProgress";
import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  type BoardLaneTone,
} from "./WorkOrderBoardChrome";
import { WorkOrderCard, type WorkOrderCardContext } from "./WorkOrderCard";

interface WorkOrdersBoardViewProps extends WorkOrderCardContext {
  entries: WorkOrderListEntry[];
  /**
   * The active Display-menu ordering. Dragging to reorder is only enabled
   * in "manual" — under any other ordering a card's position is computed,
   * so a drag has nothing stable to save. Omit on read-only boards (for
   * example the Missions board) to disable dragging outright.
   */
  ordering?: WorkOrderOrdering;
  /** Persists a drag-and-drop reorder. Required to enable dragging. */
  onReorder?: (move: WorkOrderReorderMove) => void;
}

/** Four-lane Kanban-style board mapping to the shared display statuses. */
export function WorkOrdersBoardView(props: WorkOrdersBoardViewProps) {
  const { ordering, onReorder, entries, ...rest } = props;
  const grouped = groupWorkOrderEntriesByLane(entries);
  const dragEnabled = ordering === "manual" && Boolean(onReorder);

  const sensors = useSensors(
    // A small activation distance keeps a plain click opening the task
    // (dnd-kit only starts a drag once the pointer clears this threshold).
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    // Lets keyboard users reorder with Space to pick up and arrow keys to
    // move, same as dnd-kit's standard sortable list pattern.
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over) {
        return;
      }

      const laneId = active.data.current?.laneId as WorkOrderBoardLaneId | undefined;
      const overLaneId = over.data.current?.laneId as WorkOrderBoardLaneId | undefined;
      // Cards are confined to their own column by the modifiers below, so
      // this only trips on a stale drag target; keep it as a hard backstop
      // against ever saving a cross-column move.
      if (!laneId || laneId !== overLaneId) {
        return;
      }

      const laneIds = (grouped.get(laneId) ?? []).map((entry) => entry.id);
      const move = buildWorkOrderReorderMove(laneIds, String(active.id), String(over.id));
      if (move) {
        onReorder?.(move);
      }
    },
    [grouped, onReorder],
  );

  const board = (
    <WorkOrderKanbanBoard testId="work-orders-board">
      {WORK_ORDER_BOARD_LANES.map((lane) => (
        <BoardLane key={lane.id} {...rest} lane={lane} entries={grouped.get(lane.id) ?? []} draggable={dragEnabled} />
      ))}
    </WorkOrderKanbanBoard>
  );

  if (!dragEnabled) {
    return board;
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalAxis, restrictToParentElement]}
      onDragEnd={handleDragEnd}
    >
      {board}
    </DndContext>
  );
}

interface BoardLaneProps extends Omit<WorkOrdersBoardViewProps, "entries" | "ordering" | "onReorder"> {
  lane: WorkOrderBoardLaneDefinition;
  entries: WorkOrderListEntry[];
  draggable: boolean;
}

function BoardLane({ lane, entries, draggable, ...rest }: BoardLaneProps) {
  return (
    <WorkOrderBoardLane
      title={lane.title}
      count={entries.length}
      tone={laneTone(lane.id)}
      emptyDescription={lane.description}
      testId={`work-orders-board-lane-${lane.id}`}
    >
      {draggable ? (
        <SortableContext id={lane.id} items={entries.map((entry) => entry.id)} strategy={verticalListSortingStrategy}>
          <ul className={workOrderKanbanLaneScrollClassName}>
            {entries.map((entry) => (
              <SortableWorkOrderCard key={entry.id} entry={entry} laneId={lane.id} {...rest} />
            ))}
          </ul>
        </SortableContext>
      ) : (
        <ul className={workOrderKanbanLaneScrollClassName}>
          {entries.map((entry) => (
            <li key={entry.id}>
              <WorkOrderCard entry={entry} {...rest} />
            </li>
          ))}
        </ul>
      )}
    </WorkOrderBoardLane>
  );
}

interface SortableWorkOrderCardProps extends WorkOrderCardContext {
  entry: WorkOrderListEntry;
  laneId: WorkOrderBoardLaneId;
}

/** One draggable card inside a manual-ordered column. */
function SortableWorkOrderCard({ entry, laneId, ...rest }: SortableWorkOrderCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: entry.id,
    data: { laneId },
  });

  return (
    <li
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(isDragging && "z-10 opacity-60")}
      {...attributes}
      {...listeners}
    >
      <WorkOrderCard entry={entry} {...rest} />
    </li>
  );
}

function laneTone(laneId: WorkOrderBoardLaneId): BoardLaneTone {
  if (laneId === "running") {
    return "running";
  }
  if (laneId === "done") {
    return "done";
  }
  return "neutral";
}
