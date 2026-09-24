import { memo } from "react";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import type { IntentDocument } from "../../lib/intentDocument";
import { StreamingText } from "./StreamingText";

const PLAN_PLACEHOLDER = "The analysis is writing the plan.";
const EMPTY_SUMMARY = "The analysis has not written a summary yet.";
const EMPTY_DOCUMENT = "The analysis has not written a plan yet.";

type WorkOrderIntentPlanProps = {
  document: IntentDocument;
  streamKey?: string;
  streamReady?: boolean;
  isAnalyzing?: boolean;
};

const SpecMarkdown = memo(function SpecMarkdown({ content, testId }: { content: string; testId: string }) {
  return <MarkdownContent content={content} variant="workspace" data-testid={testId} />;
});

export function WorkOrderIntentPlan({
  document,
  streamKey,
  streamReady = true,
  isAnalyzing = false,
}: WorkOrderIntentPlanProps) {
  const summary = document.summary.trim();
  const plan = document.plan.trim();
  if (!summary && !plan) {
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
      {summary ? (
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
      {summary && plan ? <Separator className="my-6" /> : null}
      {plan ? (
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
  );
}
