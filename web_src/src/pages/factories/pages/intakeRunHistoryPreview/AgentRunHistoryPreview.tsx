import { useState } from "react";
import { AutomationRunDetails } from "@/ui/AutomationRunHistory/AutomationRunDetails";
import { AutomationRunHistory } from "@/ui/AutomationRunHistory/AutomationRunHistory";
import type { AutomationRun } from "@/ui/AutomationRunHistory/types";
import { FactoryAutomationSettingsTabs } from "../FactoryAutomationSettingsTabs";
import { PopupHeader } from "../work-order-popup-redesign/popupShared";

export function AgentRunHistoryPreview({
  title,
  runs,
  onClose,
  onOpenSettings,
}: {
  title: string;
  runs: AutomationRun[];
  onClose: () => void;
  onOpenSettings: () => void;
}) {
  const [selectedId, setSelectedId] = useState<string>();
  const selected = runs.find((run) => run.id === selectedId);
  return (
    <main className="isolate flex h-dvh min-w-0 flex-col bg-background text-foreground antialiased">
      <PopupHeader title={title} onClose={onClose}>
        <FactoryAutomationSettingsTabs
          tab="runs"
          onTabChange={onOpenSettings}
          hasAgent={false}
          generalLabel="Configuration"
          historyEnabled
          label="Automation settings"
          testId="column-automation-view"
        />
      </PopupHeader>
      <div className="min-h-0 flex-1 overflow-auto p-4 sm:p-6">
        {selected ? (
          <AutomationRunDetails run={selected} onBack={() => setSelectedId(undefined)} onRetryLogs={() => {}} />
        ) : null}
        <div hidden={Boolean(selected)}>
          <AutomationRunHistory runs={runs} onSelectRun={(run) => setSelectedId(run.id)} onRetry={() => {}} />
        </div>
      </div>
    </main>
  );
}
