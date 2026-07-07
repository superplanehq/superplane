import { Loader2 } from "lucide-react";
import { Accordion } from "@/ui/accordion";
import { RunInspectorErrorSummaryCard } from "./RunInspectorErrorSummaryCard";
import { RunInspectorNodeAccordion } from "./RunInspectorNodeAccordion";
import { RunInspectorStepsHeader } from "./RunInspectorStepsHeader";
import type { RunInspectorErrorSummary, RunInspectorNodeSection } from "./runNodeDetailModel";
import type { RUN_STATUS_META } from "./runPresentation";

export function RunInspectorStepsList({
  errorSummaries,
  status,
  sections,
  isLoading,
  selectedValue,
  componentIconMap,
  onValueChange,
  onJumpToError,
  onRerun,
  rerunPending,
}: {
  errorSummaries: RunInspectorErrorSummary[];
  status: keyof typeof RUN_STATUS_META;
  sections: RunInspectorNodeSection[];
  isLoading: boolean;
  selectedValue: string;
  componentIconMap: Record<string, string>;
  onValueChange: (value: string) => void;
  onJumpToError: (nodeId: string) => void;
  onRerun: () => void;
  rerunPending: boolean;
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
              key={section.nodeId}
              section={section}
              componentIconMap={componentIconMap}
              isOpen={selectedValue === section.nodeId}
              onRerun={onRerun}
              rerunPending={rerunPending}
            />
          ))}
        </Accordion>
      )}
    </div>
  );
}
