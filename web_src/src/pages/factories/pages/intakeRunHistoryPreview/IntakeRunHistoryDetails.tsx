import { AutomationRunDetails } from "@/ui/AutomationRunHistory/AutomationRunDetails";
import type { IntakeRunPreview } from "./intakeRunPreviewModel";
import { intakePreviewRun } from "./intakePreviewRun";

export function IntakeRunHistoryDetails({ run, onBack }: { run: IntakeRunPreview; onBack: () => void }) {
  return (
    <section aria-label="Intake run details">
      <AutomationRunDetails run={intakePreviewRun(run)} onBack={onBack} onRetryLogs={() => {}} />
    </section>
  );
}
