import type { SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { isRunnerComponent } from "../streamNotesFromLiveLog";

export type FailedNodeError = {
  id: string;
  name: string;
  message: string;
  iconSlug?: string;
  executionId?: string;
};

export function failedNodeErrors(lines: SplitRunStreamLine[], runStatus?: SplitRunPhaseStatus): FailedNodeError[] {
  if (runStatus === "cancelled") {
    return [];
  }
  return lines
    .filter(
      (line) =>
        !line.note && line.status === "failed" && (!isRunnerComponent(line.component) || Boolean(line.detail?.trim())),
    )
    .map((line) => ({
      id: line.id,
      name: line.componentName,
      message: line.detail?.trim() || "This node failed.",
      iconSlug: line.iconSlug,
      executionId: line.executionId,
    }));
}
