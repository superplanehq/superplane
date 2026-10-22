import type { AutomationRun } from "@/ui/AutomationRunHistory/types";
import type { IntakeRunPreview } from "./intakeRunPreviewModel";

export function intakePreviewRun(run: IntakeRunPreview): AutomationRun {
  return {
    ...run,
    title: run.source?.title,
    status: run.executionStatus,
    intake: { outcome: run.outcome, event: run.event, summary: run.summary },
  };
}
