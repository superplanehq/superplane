import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown } from "lucide-react";

import type { IntentDocument } from "../../lib/intentDocument";

const PLAN_PLACEHOLDER = "The analysis is writing the plan.";
const EMPTY_SUMMARY = "The analysis has not written a summary yet.";
const EMPTY_DOCUMENT = "The analysis has not written a plan yet.";

type WorkOrderIntentPlanProps = {
  document: IntentDocument;
  expanded: boolean;
  onToggle: () => void;
  isAnalyzing?: boolean;
};

export function WorkOrderIntentPlan({
  document,
  expanded,
  onToggle,
  isAnalyzing = false,
}: WorkOrderIntentPlanProps) {
  const hasPlan = Boolean(document.plan.trim());
  if (!document.summary.trim() && !hasPlan) {
    return (
      <p
        className={cn("text-[13px] leading-5 text-muted-foreground", isAnalyzing && "sp-ai-thinking")}
        data-text={isAnalyzing ? PLAN_PLACEHOLDER : undefined}
      >
        {isAnalyzing ? PLAN_PLACEHOLDER : EMPTY_DOCUMENT}
      </p>
    );
  }

  return (
    <div>
      {document.summary.trim() ? (
        <div key={document.summary} className="sp-text-reveal">
          <MarkdownContent content={document.summary} variant="workspace" data-testid="split-run-intent-summary" />
        </div>
      ) : (
        <p className="text-[13px] text-muted-foreground">{EMPTY_SUMMARY}</p>
      )}
      {hasPlan ? (
        <div className="mt-6">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-8 px-2 text-[13px] text-muted-foreground"
            onClick={onToggle}
            aria-expanded={expanded}
            data-testid="split-run-intent-plan-toggle"
          >
            <ChevronDown className={cn("size-3.5 transition-transform", expanded && "rotate-180")} aria-hidden />
            {expanded ? "Hide full plan" : "Show full plan"}
          </Button>
          {expanded ? (
            <div key={document.plan} className="sp-stream-text" data-testid="split-run-intent-plan-panel">
              <MarkdownContent content={document.plan} variant="workspace" data-testid="split-run-intent-plan" />
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
