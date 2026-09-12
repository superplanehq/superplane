import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown } from "lucide-react";

import type { IntentDocument } from "../../lib/intentDocument";

type WorkOrderIntentPlanProps = {
  document: IntentDocument;
  expanded: boolean;
  onToggle: () => void;
};

export function WorkOrderIntentPlan({ document, expanded, onToggle }: WorkOrderIntentPlanProps) {
  const hasPlan = Boolean(document.plan.trim());
  if (!document.summary.trim() && !hasPlan) {
    return <p className="text-[13px] text-muted-foreground">The analysis has not written a plan yet.</p>;
  }

  return (
    <div>
      {document.summary.trim() ? (
        <MarkdownContent content={document.summary} variant="workspace" data-testid="split-run-intent-summary" />
      ) : (
        <p className="text-[13px] text-muted-foreground">The analysis has not written a summary yet.</p>
      )}
      {hasPlan ? (
        <div className="mt-4">
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
            <div
              className="mt-3 rounded-lg border border-border bg-muted/40 px-4 py-3"
              data-testid="split-run-intent-plan-panel"
            >
              <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Plan</p>
              <MarkdownContent content={document.plan} variant="workspace" data-testid="split-run-intent-plan" />
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
