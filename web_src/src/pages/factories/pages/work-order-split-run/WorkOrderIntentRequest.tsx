import type { FormEvent } from "react";

import type { FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Loader2 } from "lucide-react";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import { planningSessionPhase } from "../planningSessionActivity";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { useFollowLogScroll } from "./useFollowLogScroll";
import { WorkOrderIntentSurvey } from "./WorkOrderIntentSurvey";
import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";

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

type WorkOrderIntentRequestProps = {
  title: string;
  description: string;
  files?: FilesFile[];
  analysis?: IntentAnalysisChat;
};

export function WorkOrderIntentRequest({ title, description, files, analysis }: WorkOrderIntentRequestProps) {
  if (analysis) {
    return <AnalysisRequestChat title={title} description={description} files={files} analysis={analysis} />;
  }
  return (
    <>
      <RequestHeader title={title} />
      <RequestMessage description={description} files={files} />
    </>
  );
}

function RequestHeader({ title }: { title: string }) {
  return (
    <header className="sticky top-0 z-10 shrink-0 px-5 py-3" data-testid="split-run-intent-session">
      <h2 className="truncate text-[13px] leading-5 font-medium text-foreground">{title}</h2>
    </header>
  );
}

function analysisRequestChatState(analysis: IntentAnalysisChat) {
  const stopped = analysis.view.machineStatus === "failed" || analysis.view.machineStatus === "passed";
  const active = analysis.view.machineStatus === "starting" || analysis.view.machineStatus === "running";
  return {
    followKey: analysis.view.executionId || analysis.view.canvasId || "analysis",
    active,
    showSurvey: Boolean(analysis.view.survey && analysis.canSend),
    placeholder:
      !analysis.canSend && stopped ? ANALYSIS_PLANNING_COPY.stopped : ANALYSIS_PLANNING_COPY.composerPlaceholder,
  };
}

function AnalysisRequestChat({
  title,
  description,
  files,
  analysis,
}: WorkOrderIntentRequestProps & { analysis: IntentAnalysisChat }) {
  const state = analysisRequestChatState(analysis);
  const follow = useFollowLogScroll<HTMLDivElement>(state.followKey, analysis.view.messages.length, {
    resumeOnBottom: true,
  });
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (analysis.canSend) {
      analysis.onSend();
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="split-run-intent-chat">
      <RequestHeader title={title} />
      <div className="relative min-h-0 flex-1">
        <div
          ref={follow.scrollRef}
          onScroll={follow.onScroll}
          className="absolute inset-0 overflow-y-auto px-3 py-3"
          data-testid="split-run-intent-chat-log"
        >
          <RequestMessage description={description} files={files} asChat />
          <WorkOrderIntentTranscript messages={analysis.view.messages} />
          {state.active && analysis.view.canvasId && analysis.view.executionId ? (
            <PhaseLogCard
              phase={planningSessionPhase(analysis.view)}
              expanded
              collapsible={false}
              organizationId={analysis.organizationId}
              canvasId={analysis.view.canvasId}
              compactSessionLog
            />
          ) : state.active ? (
            <div className="flex items-center gap-2 px-2 py-1.5 text-[13px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
              <p>{ANALYSIS_PLANNING_COPY.writing}</p>
            </div>
          ) : null}
        </div>
        {follow.following ? null : (
          <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-intent-older" />
        )}
      </div>
      {state.showSurvey && analysis.view.survey ? (
        <WorkOrderIntentSurvey survey={analysis.view.survey} onSubmit={analysis.onSubmitSurvey} />
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
            placeholder={state.placeholder}
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
