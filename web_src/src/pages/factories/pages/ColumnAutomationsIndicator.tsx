import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationIconButton, type ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

/** One header icon for each configured automation. Click opens settings. */
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
          <ColumnAutomationIconButton automation={automation} onClick={() => onRowAction(automation, "settings")} />
        </div>
      ))}
    </div>
  );
}
