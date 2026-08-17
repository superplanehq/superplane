import { groupWorkOrderEntriesByLane, type WorkOrderListEntry } from "../lib/workOrderListModel";
import {
  WORK_ORDER_BOARD_LANES,
  type WorkOrderBoardLaneDefinition,
  type WorkOrderBoardLaneId,
} from "../lib/workOrderProgress";
import { WorkOrderBoardLane, type BoardLaneTone } from "./WorkOrderBoardChrome";
import { WorkOrderCard, type WorkOrderCardContext } from "./WorkOrderCard";

interface WorkOrdersBoardViewProps extends WorkOrderCardContext {
  entries: WorkOrderListEntry[];
}

/** Four-lane Kanban-style board mapping to the shared display statuses. */
export function WorkOrdersBoardView(props: WorkOrdersBoardViewProps) {
  const grouped = groupWorkOrderEntriesByLane(props.entries);
  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4" data-testid="work-orders-board">
      {WORK_ORDER_BOARD_LANES.map((lane) => (
        <BoardLane key={lane.id} {...props} lane={lane} entries={grouped.get(lane.id) ?? []} />
      ))}
    </div>
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
      <ul className="flex flex-col gap-2">
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
