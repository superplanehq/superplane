import { emptyColumnAutomationActivity } from "../lib/columnAutomationActivity";
import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationsPopup, type ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

/** One header icon for each configured automation. */
export function ColumnAutomationsHeaderSlot({
  title,
  automations,
  onRowAction,
  testId,
}: {
  title: string;
  automations?: ColumnAutomation[];
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
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
          />
        </div>
      ))}
    </div>
  );
}
