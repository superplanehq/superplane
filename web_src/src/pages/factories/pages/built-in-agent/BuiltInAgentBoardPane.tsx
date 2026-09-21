import { formatRelative } from "@/lib/datetime";
import { cn } from "@/lib/utils";

import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  type BoardLaneTone,
} from "../../workOrders/WorkOrderBoardChrome";
import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import { WORK_ORDER_CARD_HOVER_SURFACE_CLASS } from "../../workOrders/workOrderCardSurface";
import {
  WORK_ORDER_BOARD_LANES,
  getWorkOrderDisplayStatusMeta,
  type WorkOrderBoardLaneId,
} from "../../lib/workOrderProgress";
import { BUILT_IN_AGENT_COPY, type BuiltInAgentTask } from "./builtInAgentMocks";

export function BuiltInAgentBoardPane({ tasks }: { tasks: BuiltInAgentTask[] }) {
  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background px-4 py-3"
      aria-label={BUILT_IN_AGENT_COPY.tasksHeading}
    >
      <h1 className="workspace-section-title mb-3 shrink-0">{BUILT_IN_AGENT_COPY.tasksHeading}</h1>
      <WorkOrderKanbanBoard testId="built-in-agent-board">
        {WORK_ORDER_BOARD_LANES.map((lane) => {
          const laneTasks = tasks.filter((task) => task.lane === lane.id);
          return (
            <WorkOrderBoardLane
              key={lane.id}
              title={lane.title}
              count={laneTasks.length}
              tone={laneTone(lane.id)}
              emptyDescription={lane.description}
              testId={`built-in-agent-board-lane-${lane.id}`}
            >
              <ul className={workOrderKanbanLaneScrollClassName}>
                {laneTasks.map((task) => (
                  <li key={task.id}>
                    <PrototypeTaskCard task={task} />
                  </li>
                ))}
              </ul>
            </WorkOrderBoardLane>
          );
        })}
      </WorkOrderKanbanBoard>
    </section>
  );
}

function PrototypeTaskCard({ task }: { task: BuiltInAgentTask }) {
  const meta = getWorkOrderDisplayStatusMeta(task.state);
  const ownerName = task.owner?.trim().split(/\s+/)[0];

  return (
    <article
      className={cn(
        "w-full rounded-md border border-border bg-card p-2.5 shadow-sm",
        WORK_ORDER_CARD_HOVER_SURFACE_CLASS,
      )}
      data-testid={`built-in-agent-card-${task.id}`}
    >
      <div className="flex min-w-0 items-center gap-2">
        <WorkOrderStatusIcon status={task.state} title={meta.label} aria-label={meta.label} />
        <h3 className="min-w-0 flex-1 truncate text-[13px] font-medium leading-snug text-foreground">{task.title}</h3>
      </div>
      <div className="mt-2 flex items-center justify-between gap-2">
        <span className="truncate text-[11px] leading-4 text-muted-foreground">{formatRelative(task.updatedAt)}</span>
        {ownerName ? <span className="truncate text-[11px] leading-4 text-muted-foreground">{ownerName}</span> : null}
      </div>
    </article>
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
