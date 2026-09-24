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
      onAdd={args.onAdd}
      testId={args.testId}
    />
  );
}
