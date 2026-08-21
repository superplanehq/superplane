import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
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
}

/** Four-lane Kanban-style board mapping to the shared display statuses. */
export function WorkOrdersBoardView(props: WorkOrdersBoardViewProps) {
  const grouped = groupWorkOrderEntriesByLane(props.entries);
  return (
    <WorkOrderKanbanBoard testId="work-orders-board">
      {WORK_ORDER_BOARD_LANES.map((lane) => (
        <BoardLane key={lane.id} {...props} lane={lane} entries={grouped.get(lane.id) ?? []} />
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
          <li key={entry.id}>
            <WorkOrderCard entry={entry} {...rest} />
          </li>
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
