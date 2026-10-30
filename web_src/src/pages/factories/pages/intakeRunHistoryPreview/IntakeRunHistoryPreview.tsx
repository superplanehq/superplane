import { PopupHeader } from "../work-order-popup-redesign/popupShared";
import { FactoryAutomationSettingsTabs } from "../FactoryAutomationSettingsTabs";
import { Clock } from "lucide-react";
import { useState } from "react";
import { IntakeSettingsHeader } from "../IntakeSettingsHeader";
import { lineIntakeSourceById } from "../lineIntakeModel";
import { AutomationRunHistory } from "@/ui/AutomationRunHistory/AutomationRunHistory";
import { intakePreviewRun } from "./intakePreviewRun";
import { IntakeRunHistoryDetails } from "./IntakeRunHistoryDetails";
import type { IntakePreviewSource, IntakeRunPreview } from "./intakeRunPreviewModel";

/** Interactive page examples with the production history components. */
export function IntakeRunHistoryPreview({
  sourceId,
  runs,
  initialRunId,
  loading,
  error,
  onRetry,
  onOpenSettings,
  onClose,
}: {
  sourceId: IntakePreviewSource;
  runs: IntakeRunPreview[];
  initialRunId?: string;
  loading?: boolean;
  error?: string;
  onRetry: () => void;
  onOpenSettings: () => void;
  onClose: () => void;
}) {
  const [selectedId, setSelectedId] = useState(initialRunId);
  const selected = runs.find((run) => run.id === selectedId);
  return (
    <main
      className="isolate flex h-dvh min-w-0 flex-col bg-background text-foreground antialiased"
      data-testid="intake-history-preview"
    >
      {sourceId === "scheduled-intake" ? (
        <PopupHeader title="Scheduled intake" onClose={onClose} leading={<Clock aria-hidden className="size-5" />}>
          <FactoryAutomationSettingsTabs
            tab="runs"
            onTabChange={onOpenSettings}
            hasAgent={false}
            historyEnabled
            label="Intake settings"
            testId="intake-settings"
          />
        </PopupHeader>
      ) : (
        <IntakeSettingsHeader
          name={lineIntakeSourceById(sourceId)?.name ?? "Intake"}
          sourceId={sourceId}
          history={{ tab: "runs", enabled: true, setTab: onOpenSettings }}
          hasAgent={false}
          onClose={onClose}
        />
      )}
      <div className="min-h-0 flex-1 overflow-auto p-4 sm:p-6">
        {selected ? (
          <IntakeRunHistoryDetails key={selected.id} run={selected} onBack={() => setSelectedId(undefined)} />
        ) : null}
        <div hidden={Boolean(selected)}>
          <AutomationRunHistory
            runs={runs.map(intakePreviewRun)}
            loading={loading}
            error={error}
            onRetry={onRetry}
            onSelectRun={(run) => setSelectedId(run.id)}
          />
        </div>
      </div>
    </main>
  );
}
