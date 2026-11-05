import type { ReactNode } from "react";

import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationRows } from "./ColumnAutomationRows";
import type { ColumnAutomationRowAction } from "./ColumnAutomationsPopup";

/** Lane subheader. Undefined while the rows are off or the column hides automations. */
export function columnAutomationRowsSubheader(args: {
  title: string;
  automations?: ColumnAutomation[];
  rowCount?: number;
  onRowAction?: (automation: ColumnAutomation, action: ColumnAutomationRowAction) => void;
  onRun?: (automation: ColumnAutomation) => void;
  runPending?: boolean;
  runningAutomationId?: string;
  onAdd?: () => void;
  testId: string;
}): ReactNode {
  if (!args.rowCount || !args.automations) {
    return undefined;
  }
  return (
    <ColumnAutomationRows
      title={args.title}
      automations={args.automations}
      rowCount={args.rowCount}
      onRowAction={args.onRowAction}
      onRun={args.onRun}
      runPending={args.runPending}
      runningAutomationId={args.runningAutomationId}
      onAdd={args.onAdd}
      testId={args.testId}
    />
  );
}
