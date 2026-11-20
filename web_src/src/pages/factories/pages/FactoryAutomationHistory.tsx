import type { ReactNode } from "react";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import type { RunHistoryIntakeSource } from "./automationRunHistory/intakeRunData";
import { AutomationRunHistory } from "@/ui/AutomationRunHistory/AutomationRunHistory";
import { FactoryAutomationRunHistory } from "./automationRunHistory/FactoryAutomationRunHistory";

export function FactoryAutomationHistory({
  organizationId,
  factoryId,
  canvasId,
  intakeSource,
  selectedRunId,
  onSelectRun,
  loading,
  error,
  onRetry,
  runRetryEnabled,
  renderRunSummary,
}: {
  organizationId?: string;
  factoryId?: string;
  canvasId?: string;
  intakeSource?: RunHistoryIntakeSource;
  selectedRunId: string | null;
  onSelectRun: (id: string | null) => void;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  runRetryEnabled?: boolean;
  renderRunSummary?: (run: CanvasesCanvasRun, executions: CanvasesCanvasNodeExecution[]) => ReactNode;
}) {
  if (!organizationId || !factoryId || !canvasId) {
    return (
      <div className="min-h-0 flex-1 overflow-auto p-6">
        <AutomationRunHistory
          runs={[]}
          loading={loading}
          error={error ? "Could not load run history." : undefined}
          onRetry={onRetry}
          onSelectRun={() => {}}
        />
      </div>
    );
  }
  return (
    <FactoryAutomationRunHistory
      organizationId={organizationId}
      factoryId={factoryId}
      canvasId={canvasId}
      intakeSource={intakeSource}
      runRetryEnabled={runRetryEnabled}
      renderRunSummary={renderRunSummary}
      selectedRunId={selectedRunId}
      onSelectRun={onSelectRun}
    />
  );
}
