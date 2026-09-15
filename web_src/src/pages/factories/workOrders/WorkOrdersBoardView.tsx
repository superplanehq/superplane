import { useMemo } from "react";
import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
import {
  WORK_ORDER_BOARD_LANES,
  type WorkOrderBoardLaneDefinition,
  type WorkOrderBoardLaneId,
} from "../lib/workOrderProgress";
import { KanbanCardMotionItem } from "./KanbanCardMotionItem";
import { tasksBoardCardPlacements, useKanbanDisplayedBoard } from "./kanbanCardMotion";
import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  type BoardLaneTone,
} from "./WorkOrderBoardChrome";
import { WorkOrderCard, type WorkOrderCardContext } from "./WorkOrderCard";

interface WorkOrdersBoardViewProps extends WorkOrderCardContext {
  entries: WorkOrderListEntry[];
}

/** Four-lane Kanban-style board mapping to the shared display statuses. */
export function WorkOrdersBoardView(props: WorkOrdersBoardViewProps) {
  const incomingPlacements = useMemo(() => tasksBoardCardPlacements(props.entries), [props.entries]);
  const entries = useKanbanDisplayedBoard(props.entries, incomingPlacements);
  const grouped = groupWorkOrderEntriesByLane(entries);
  return (
    <WorkOrderKanbanBoard testId="work-orders-board">
      {WORK_ORDER_BOARD_LANES.map((lane) => (
        <BoardLane key={lane.id} {...props} entries={grouped.get(lane.id) ?? []} lane={lane} />
      ))}
    </WorkOrderKanbanBoard>
  );
}

interface BoardLaneProps extends WorkOrdersBoardViewProps {
  lane: WorkOrderBoardLaneDefinition;
  entries: WorkOrderListEntry[];
}

function BoardLane({ lane, entries, ...rest }: BoardLaneProps) {
  return (
    <WorkOrderBoardLane
      title={lane.title}
      count={entries.length}
      tone={laneTone(lane.id)}
      emptyDescription={lane.description}
      testId={`work-orders-board-lane-${lane.id}`}
    >
      <ul className={workOrderKanbanLaneScrollClassName}>
        {entries.map((entry) => (
          <KanbanCardMotionItem key={entry.id} id={entry.id}>
            <WorkOrderCard entry={entry} {...rest} />
          </KanbanCardMotionItem>
        ))}
      </ul>
    </WorkOrderBoardLane>
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
