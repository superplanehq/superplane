import { emptyColumnAutomationActivity } from "../lib/columnAutomationActivity";
import type { ColumnAutomation } from "../lib/columnAutomations";
import {
  automationOffersAgentEdit,
  ColumnAutomationsPopup,
  type ColumnAutomationRowAction,
} from "./ColumnAutomationsPopup";

/** One header icon for each configured automation. */
export function ColumnAutomationsHeaderSlot({
  title,
  automations,
  onRowAction,
  canEditAgent,
  testId,
}: {
  title: string;
  automations?: ColumnAutomation[];
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  /** When set, overrides the default agent-edit offer for every icon in this slot. */
  canEditAgent?: boolean;
  testId: string;
}) {
  if (!onRowAction || !automations || automations.length === 0) {
    return null;
  }
  return (
    <div className="flex shrink-0 items-center" role="list" aria-label={`${title} automations`} data-testid={testId}>
      {automations.map((automation) => (
        <div key={automation.id} role="listitem">
          <ColumnAutomationsPopup
            automation={automation}
            activity={automation.activity ?? emptyColumnAutomationActivity(automation.runningCount)}
            onAction={(action) => onRowAction(automation, action)}
            showEditAgent={canEditAgent ?? automationOffersAgentEdit(automation)}
            showEditAutomation={Boolean(automation.canvasId)}
          />
        </div>
      ))}
    </div>
  );
}
