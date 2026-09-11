import { useState, type FormEvent, type ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown, Loader2 } from "lucide-react";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CONFIDENCE_CHECK_NAME, CONFIDENCE_SCORE_MAX } from "../../lib/confidenceScore";
import { INTENT_DOCUMENT_TITLE, type IntentDocument } from "../../lib/intentDocument";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { ConfidenceAnalyzingIndicator, ConfidenceMeter } from "../../workOrders/ConfidenceMeter";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import { planningSessionPhase } from "../planningSessionActivity";
import { PlanningSessionSurveyForm } from "../PlanningSessionSurveyForm";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { splitRunIntentDocument } from "./splitRunPopupModel";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { DEFAULT_INTENT_LEFT_PERCENT, useSplitRunPanePercent } from "./useSplitRunPanePercent";
import { useFollowLogScroll } from "./useFollowLogScroll";

const SESSION_TITLE_FALLBACK = "Task";

/**
 * Description-tab reading pane. Drafts keep analysis chat on the left and
 * the plan plus confidence on the right. After Start, the left pane shows
 * source context and confidence. The plan stays on the right.
 */
export type IntentAnalysisChat = {
  organizationId: string;
  view: CreateWithAgentView;
  composer: string;
  composerError?: string;
  canSend: boolean;
  onComposerChange: (value: string) => void;
  onSend: () => void;
  onSubmitSurvey: (text: string) => void;
};

export function WorkOrderIntentDocument({
  title,
  description,
  artifacts,
  confidence,
  isAnalyzing = false,
  files,
  resultAfterBody,
  resultFooter,
  analysis,
  contextSidebar,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing?: boolean;
  files?: FilesFile[];
  resultAfterBody?: ReactNode;
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
  contextSidebar?: ReactNode;
}) {
  const [showPlan, setShowPlan] = useState(false);
  const split = useSplitRunPanePercent({ defaultPercent: DEFAULT_INTENT_LEFT_PERCENT, minPercent: 28, maxPercent: 68 });
  const document = splitRunIntentDocument({ artifacts, description });
  const hasPlan = Boolean(document.plan.trim());
  const sessionTitle = title.trim() || SESSION_TITLE_FALLBACK;

  return (
    <article className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="split-run-intent-document">
      <div ref={split.containerRef} className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <div
          className="flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[var(--intent-left)] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0"
          style={{ ["--intent-left" as string]: `${split.percent}%` }}
          data-testid="split-run-intent-request"
        >
          {contextSidebar ? (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-hidden">{contextSidebar}</div>
              <IntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />
            </div>
          ) : analysis ? (
            <AnalysisRequestChat title={sessionTitle} description={description} files={files} analysis={analysis} />
          ) : (
            <>
              <header className="sticky top-0 z-10 shrink-0 px-5 py-3" data-testid="split-run-intent-session">
                <h2 className="truncate text-[13px] leading-5 font-medium text-foreground">{sessionTitle}</h2>
              </header>
              <RequestMessage description={description} files={files} />
            </>
          )}
        </div>

        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize the request and plan"
          data-testid="split-run-intent-resize-handle"
          onPointerDown={split.startResize}
          className="group relative z-10 hidden w-2 shrink-0 cursor-col-resize bg-transparent lg:block"
        >
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-border",
              split.isResizing && "bg-border",
            )}
          />
        </div>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col lg:min-w-[16rem]" data-testid="split-run-intent-result">
          <header className="flex shrink-0 items-start justify-between gap-3 px-5 pt-5 pb-2">
            <h2 className="min-w-0 text-[17px] leading-6 font-semibold tracking-tight text-foreground">
              {document.title || INTENT_DOCUMENT_TITLE}
            </h2>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-5" data-testid="split-run-intent-body">
            <IntentDocumentBody
              document={document}
              showPlan={showPlan}
              hasPlan={hasPlan}
              onTogglePlan={() => setShowPlan((current) => !current)}
            />
            {resultAfterBody}
          </div>
          {contextSidebar ? null : <IntentConfidenceFooter confidence={confidence} isAnalyzing={isAnalyzing} />}
          {resultFooter}
        </div>
      </div>
    </article>
  );
}

function AnalysisRequestChat({
  title,
  description,
  files,
  analysis,
}: {
  title: string;
  description: string;
  files?: FilesFile[];
  analysis: IntentAnalysisChat;
}) {
  const follow = useFollowLogScroll<HTMLDivElement>(
    analysis.view.executionId || analysis.view.canvasId || "analysis",
    analysis.view.messages.length,
    { resumeOnBottom: true },
  );
  const stopped =
    analysis.view.machineStatus === "failed" || analysis.view.machineStatus === "passed";
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (!analysis.canSend) {
      return;
    }
    analysis.onSend();
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="split-run-intent-chat">
      <header className="sticky top-0 z-10 shrink-0 px-5 py-3" data-testid="split-run-intent-session">
        <h2 className="truncate text-[13px] leading-5 font-medium text-foreground">{title}</h2>
      </header>
      <div className="relative min-h-0 flex-1">
        <div
          ref={follow.scrollRef}
          onScroll={follow.onScroll}
          className="absolute inset-0 overflow-y-auto px-3 py-3"
          data-testid="split-run-intent-chat-log"
        >
          <RequestMessage description={description} files={files} asChat />
          {analysis.view.canvasId && analysis.view.executionId ? (
            <PhaseLogCard
              phase={planningSessionPhase(analysis.view)}
              expanded
              collapsible={false}
              organizationId={analysis.organizationId}
              canvasId={analysis.view.canvasId}
              compactSessionLog
            />
          ) : (
            <div className="flex items-center gap-2 px-2 py-1.5 text-[13px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
              <p>{ANALYSIS_PLANNING_COPY.writing}</p>
            </div>
          )}
        </div>
        {follow.following ? null : (
          <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-intent-older" />
        )}
      </div>
      {analysis.view.survey && analysis.canSend ? (
        <PlanningSessionSurveyForm
          key={analysis.view.survey.id ?? analysis.view.survey.questions[0]?.prompt ?? "survey"}
          survey={analysis.view.survey}
          onSubmit={analysis.onSubmitSurvey}
        />
      ) : null}
      <form className="border-t border-border bg-background p-3" onSubmit={handleSubmit}>
        <label htmlFor="split-run-intent-composer" className="sr-only">
          {ANALYSIS_PLANNING_COPY.composerPlaceholder}
        </label>
        <div className="flex items-end gap-2">
          <Textarea
            id="split-run-intent-composer"
            data-testid="split-run-intent-composer"
            value={analysis.composer}
            placeholder={stopped ? ANALYSIS_PLANNING_COPY.stopped : ANALYSIS_PLANNING_COPY.composerPlaceholder}
            disabled={!analysis.canSend}
            onChange={(event) => analysis.onComposerChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                if (analysis.canSend) {
                  analysis.onSend();
                }
              }
            }}
            className="min-h-[44px] resize-none text-[13px]"
            rows={2}
          />
          <Button type="submit" size="sm" disabled={!analysis.canSend || !analysis.composer.trim()}>
            {ANALYSIS_PLANNING_COPY.send}
          </Button>
        </div>
        {analysis.composerError ? (
          <p className="mt-2 text-[12px] text-destructive" data-testid="split-run-intent-chat-error">
            {analysis.composerError}
          </p>
        ) : null}
      </form>
    </div>
  );
}

function RequestMessage({
  description,
  files,
  asChat = false,
}: {
  description: string;
  files?: FilesFile[];
  asChat?: boolean;
}) {
  const body = description.trim() ? (
    <WorkOrderDescription
      description={description}
      files={files}
      previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
      fadeClassName={asChat ? "from-primary/10 via-primary/10" : "from-muted via-muted/80"}
    />
  ) : (
    <p className="text-[13px] text-muted-foreground">No request yet.</p>
  );

  if (!asChat) {
    return (
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <div className="max-w-[92%]">
          <div className="rounded-2xl bg-muted/70 px-3.5 py-3" data-testid="split-run-description" aria-label="Request">
            {body}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mb-3 flex w-full items-start" data-testid="split-run-description" aria-label="Request">
      <span className="inline-flex w-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1 whitespace-normal break-words rounded-md border-l-2 border-primary/50 bg-primary/10 px-2 py-1">
        <span className="mb-0.5 block font-sans text-[11px] font-medium leading-none text-primary">
          {CREATE_WITH_AGENT_COPY.you}
        </span>
        {body}
      </div>
    </div>
  );
}

function IntentDocumentBody({
  document,
  showPlan,
  hasPlan,
  onTogglePlan,
}: {
  document: IntentDocument;
  showPlan: boolean;
  hasPlan: boolean;
  onTogglePlan: () => void;
}) {
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
            onClick={onTogglePlan}
            aria-expanded={showPlan}
            data-testid="split-run-intent-plan-toggle"
          >
            <ChevronDown className={cn("size-3.5 transition-transform", showPlan && "rotate-180")} aria-hidden />
            {showPlan ? "Hide full plan" : "Show full plan"}
          </Button>
          {showPlan ? (
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

function IntentConfidenceFooter({
  confidence,
  isAnalyzing,
}: {
  confidence?: WorkOrderCheckPresentation;
  isAnalyzing: boolean;
}) {
  if (isAnalyzing && !confidence) {
    return (
      <footer
        className="shrink-0 border-t border-border px-5 py-3"
        data-testid="split-run-overview-checks"
        aria-label={CONFIDENCE_CHECK_NAME}
      >
        <div className="flex items-start gap-2">
          <ConfidenceAnalyzingIndicator testId="split-run-intent-confidence-meter" />
          <p className="text-[13px] leading-5 text-muted-foreground">The analysis is still running.</p>
        </div>
      </footer>
    );
  }

  if (!confidence) {
    return (
      <footer
        className="shrink-0 border-t border-border px-5 py-3"
        data-testid="split-run-overview-checks"
        aria-label={CONFIDENCE_CHECK_NAME}
      >
        <p className="text-[13px] text-muted-foreground">No confidence score yet.</p>
      </footer>
    );
  }

  return (
    <footer
      className="shrink-0 border-t border-border px-5 py-3"
      data-testid="split-run-overview-checks"
      aria-label={CONFIDENCE_CHECK_NAME}
    >
      <div className="flex items-start gap-2.5">
        <div className="flex shrink-0 flex-col items-start gap-1 pt-0.5">
          <ConfidenceMeter score={confidence.score} testId="split-run-intent-confidence-meter" />
          <span className="text-[12px] tabular-nums text-muted-foreground">
            {confidence.score}/{confidence.maxScore || CONFIDENCE_SCORE_MAX}
          </span>
        </div>
        <div className="min-w-0">
          <p className="text-[12px] text-muted-foreground">{CONFIDENCE_CHECK_NAME}</p>
          <p className="text-[13px] leading-5 text-foreground" data-testid="split-run-intent-confidence-copy">
            {confidence.summary?.trim() || "The analysis scored how clear this work is."}
          </p>
        </div>
      </div>
    </footer>
  );
}
