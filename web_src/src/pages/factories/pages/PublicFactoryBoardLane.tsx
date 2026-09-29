import type { LineBoardColumnColorView } from "../lib/lineBoardColumnColorViewPreference";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderBoardLane, workOrderKanbanLaneScrollClassName } from "../workOrders/WorkOrderBoardChrome";
import { WorkOrderCard } from "../workOrders/WorkOrderCard";
import { columnAutomationRowsSubheader } from "./columnAutomationRowsSubheader";
import { LineBoardColumnCardList } from "./LineBoardOrderCard";
import { lineBoardColumnLaneProps } from "./lineBoardColumnColors";
import {
  columnAutomations,
  columnEmptyDescription,
  publicBoardOrder,
  type PublicBoard,
  type PublicBoardCard,
  type PublicBoardColumn,
} from "./publicBoardModel";

const IDLE_ORDER_IDS: ReadonlySet<string> = new Set();

export function PublicBoardColumnLane({
  column,
  narrowed,
  board,
  automationRowCount,
  colorView,
}: {
  column: PublicBoardColumn;
  narrowed: boolean;
  board: PublicBoard;
  automationRowCount: number;
  colorView: LineBoardColumnColorView;
}) {
  const phase = column.key.startsWith("phase-");
  const lane = lineBoardColumnLaneProps(column.color, colorView, phase ? undefined : { mutedFallback: true });

  return (
    <WorkOrderBoardLane
      title={column.title}
      label={phase ? `${column.title} phase` : column.title}
      count={column.cards.length}
      tone={column.key === "done" ? "done" : "neutral"}
      surfaceClassName={lane.surfaceClassName}
      className={lane.className}
      emptyDescription={columnEmptyDescription(column, narrowed)}
      subheader={columnAutomationRowsSubheader({
        title: column.title,
        automations: columnAutomations(column),
        rowCount: automationRowCount,
        testId: `public-board-automations-${column.key}`,
      })}
      testId={`public-board-column-${column.key}`}
    >
      <LineBoardColumnCardList pending={false} className={workOrderKanbanLaneScrollClassName}>
        {column.cards.map((card, index) => (
          <li key={card.id ?? `${column.key}-${index}`}>
            <PublicBoardTaskCard card={card} board={board} index={index} />
          </li>
        ))}
      </LineBoardColumnCardList>
    </WorkOrderBoardLane>
  );
}

function PublicBoardTaskCard({ card, board, index }: { card: PublicBoardCard; board: PublicBoard; index: number }) {
  const order = publicBoardOrder(card, `${board.workspaceKey ?? "board"}-${index}`);
  const entry = buildWorkOrderListEntry(order, { key: board.workspaceKey });

  return (
    <WorkOrderCard
      entry={entry}
      organizationId=""
      factoryKey={board.workspaceKey ?? ""}
      factoryLines={[]}
      canDispatch={false}
      canAssign={false}
      dispatchingOrderIds={IDLE_ORDER_IDS}
      isAssigneesSaving={false}
      onDispatch={async () => undefined}
      onAssigneesSave={async () => undefined}
      pullRequests={order.pullRequests ?? []}
      clarityScore={card.clarity}
      confidenceScore={card.confidence}
      showClarity={board.showClarity}
      showConfidenceScore={board.showConfidence}
      hasAgentQuestion={Boolean(card.agentQuestion)}
      interactive={false}
      showOwner
    />
  );
}
