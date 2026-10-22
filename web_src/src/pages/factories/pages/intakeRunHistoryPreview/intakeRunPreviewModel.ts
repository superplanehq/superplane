import type { LineIntakeSourceId } from "../lineIntakeModel";
import type { RunActivity, RunStatus } from "@/ui/AutomationRunHistory/types";

import type { IntakeRunOutcome } from "@/ui/AutomationRunHistory/intakeRunOutcome";
export type { IntakeRunOutcome } from "@/ui/AutomationRunHistory/intakeRunOutcome";

export type IntakePreviewSource = LineIntakeSourceId | "scheduled-intake";

export interface IntakeRunPreview {
  id: string;
  source?: { reference: string; title: string; url?: string };
  event: string;
  outcome: IntakeRunOutcome;
  summary: string;
  startedAt: string;
  duration: string;
  finishedAt?: string;
  executionStatus: RunStatus;
  task?: { id: string; title: string };
  error?: string;
  activities: RunActivity[];
}
