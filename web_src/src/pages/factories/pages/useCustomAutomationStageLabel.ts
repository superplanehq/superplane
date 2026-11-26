import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { customAutomationColumnLabel, type CustomAutomationColumn } from "../lib/customAutomation";

export function useCustomAutomationStageLabel(column: CustomAutomationColumn): string {
  const layout = useOptionalFactoriesLayout();
  const stage = layout?.factory?.lines?.flatMap((line) => line.stages ?? []).find((entry) => entry.key === column);
  return stage?.name || customAutomationColumnLabel(column);
}
