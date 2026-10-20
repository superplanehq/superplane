import { AutomationRunHistory } from "@/ui/AutomationRunHistory/AutomationRunHistory";
import { FactoryAutomationRunHistory } from "./automationRunHistory/FactoryAutomationRunHistory";

export function FactoryAutomationHistory({
  organizationId,
  factoryId,
  canvasId,
  selectedRunId,
  onSelectRun,
  loading,
  error,
  onRetry,
}: {
  organizationId?: string;
  factoryId?: string;
  canvasId?: string;
  selectedRunId: string | null;
  onSelectRun: (id: string | null) => void;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
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
      selectedRunId={selectedRunId}
      onSelectRun={onSelectRun}
    />
  );
}
