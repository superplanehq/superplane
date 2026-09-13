import { memo } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown } from "lucide-react";

import type { IntentDocument } from "../../lib/intentDocument";
import { StreamingText } from "./StreamingText";

const PLAN_PLACEHOLDER = "The analysis is writing the plan.";
const EMPTY_SUMMARY = "The analysis has not written a summary yet.";
const EMPTY_DOCUMENT = "The analysis has not written a plan yet.";

type WorkOrderIntentPlanProps = {
  document: IntentDocument;
  streamKey?: string;
  streamReady?: boolean;
  expanded: boolean;
  onToggle: () => void;
  isAnalyzing?: boolean;
};

const SpecMarkdown = memo(function SpecMarkdown({ content, testId }: { content: string; testId: string }) {
  return <MarkdownContent content={content} variant="workspace" data-testid={testId} />;
});

export function WorkOrderIntentPlan({
  document,
  streamKey,
  streamReady = true,
  expanded,
  onToggle,
  isAnalyzing = false,
}: WorkOrderIntentPlanProps) {
  const hasPlan = Boolean(document.plan.trim());
  if (!document.summary.trim() && !hasPlan) {
    return (
      <StreamingText content="" memoryKey={streamKey ? `${streamKey}:summary` : undefined} ready={streamReady}>
        {() => (
          <p
            className={cn("text-[13px] leading-5 text-muted-foreground", isAnalyzing && "sp-ai-thinking")}
            data-text={isAnalyzing ? PLAN_PLACEHOLDER : undefined}
          >
            {isAnalyzing ? PLAN_PLACEHOLDER : EMPTY_DOCUMENT}
          </p>
        )}
      </StreamingText>
    );
  }

  return (
    <div>
      {document.summary.trim() ? (
        <StreamingText
          content={document.summary}
          memoryKey={streamKey ? `${streamKey}:summary` : undefined}
          ready={streamReady}
        >
          {(visible) => <SpecMarkdown content={visible} testId="split-run-intent-summary" />}
        </StreamingText>
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
            <StreamingText
              content={document.plan}
              memoryKey={streamKey ? `${streamKey}:plan` : undefined}
              ready={streamReady}
              data-testid="split-run-intent-plan-panel"
            >
              {(visible) => <SpecMarkdown content={visible} testId="split-run-intent-plan" />}
            </StreamingText>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
