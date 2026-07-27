import { Loader2 } from "lucide-react";
import { Accordion } from "@/ui/accordion";
import { RunInspectorErrorSummaryCard } from "./RunInspectorErrorSummaryCard";
import { RunInspectorNodeAccordion } from "./RunInspectorNodeAccordion";
import { RunInspectorStepsHeader } from "./RunInspectorStepsHeader";
import type { RunInspectorCurrentUser, RunInspectorErrorSummary, RunInspectorNodeSection } from "./types";
import type { RUN_STATUS_META } from "./runPresentation";
import type { useRunInspectorActions } from "./useRunInspectorActions";

export function RunInspectorStepsList({
  errorSummaries,
  status,
  sections,
  isLoading,
  selectedValue,
  componentIconMap,
  organizationId,
  canShowExpressionTemplates,
  onValueChange,
  onJumpToError,
  onRerun,
  onEditNode,
  rerunPending,
  actions,
  currentUser,
  errorScrollRequest,
  onErrorScrolled,
}: {
  errorSummaries: RunInspectorErrorSummary[];
  status: keyof typeof RUN_STATUS_META;
  sections: RunInspectorNodeSection[];
  isLoading: boolean;
  selectedValue: string;
  componentIconMap: Record<string, string>;
  organizationId?: string;
  canShowExpressionTemplates?: boolean;
  onValueChange: (value: string) => void;
  onJumpToError: (nodeId: string) => void;
  onRerun: () => void;
  onEditNode?: (nodeId: string) => void;
  rerunPending: boolean;
  actions: ReturnType<typeof useRunInspectorActions>;
  currentUser?: RunInspectorCurrentUser;
  errorScrollRequest?: { nodeId: string; requestId: number } | null;
  onErrorScrolled?: () => void;
}) {
  return (
    <div className="min-h-0 flex-1 overflow-y-auto" data-testid="run-panel-step-list">
      {errorSummaries.length > 0 ? (
        <div className="space-y-2 px-4 py-3">
          {errorSummaries.map((summary) => (
            <RunInspectorErrorSummaryCard
              key={summary.nodeId}
              nodeName={summary.nodeName}
              message={summary.message}
              onJump={() => onJumpToError(summary.nodeId)}
            />
          ))}
        </div>
      ) : null}

      <RunInspectorStepsHeader status={status} errorCount={errorSummaries.length} stepCount={sections.length} />

      {isLoading ? (
        <div className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500 dark:text-gray-400">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading run steps...
        </div>
      ) : sections.length === 0 ? (
        <div className="px-4 py-8 text-sm text-slate-500 dark:text-gray-400">No executed nodes in this run.</div>
      ) : (
        <Accordion type="single" collapsible value={selectedValue} onValueChange={onValueChange}>
          {sections.map((section) => (
            <RunInspectorNodeAccordion
              key={section.sectionValue}
              section={section}
              componentIconMap={componentIconMap}
              organizationId={organizationId}
              canShowExpressionTemplates={canShowExpressionTemplates}
              isOpen={selectedValue === section.sectionValue}
              onRerun={onRerun}
              onEditNode={onEditNode}
              rerunPending={rerunPending}
              actions={actions}
              currentUser={currentUser}
              errorScrollRequestId={errorScrollRequest?.nodeId === section.nodeId ? errorScrollRequest.requestId : null}
              onErrorScrolled={onErrorScrolled}
              onSelectSection={onValueChange}
            />
          ))}
        </Accordion>
      )}
    </div>
  );
}
