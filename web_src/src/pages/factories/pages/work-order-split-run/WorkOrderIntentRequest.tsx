import type { FormEvent } from "react";
import { ArrowUp } from "lucide-react";

import type { FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import { AnalysisLiveWork } from "./IntentAnalysisLiveWork";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { useFollowLogScroll } from "./useFollowLogScroll";
import { SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME } from "./splitRunPopupModel";
import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderIntentSurvey } from "./WorkOrderIntentSurvey";
import { WorkOrderIntentTranscript } from "./WorkOrderIntentTranscript";
import { WorkOrderSplitRunSource } from "./WorkOrderSplitRunSource";

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
  source?: SplitRunSource;
};

export function WorkOrderIntentRequest({ title, description, files, analysis, source }: WorkOrderIntentRequestProps) {
  if (analysis) {
    return (
      <AnalysisRequestChat title={title} description={description} files={files} analysis={analysis} source={source} />
    );
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
  const latestMessage = analysis.view.messages.at(-1);
  return {
    followKey: analysis.view.executionId || analysis.view.canvasId || "analysis",
    active,
    showSurvey: Boolean(analysis.view.survey && analysis.canSend && !active && latestMessage?.role === "agent"),
    placeholder:
      !analysis.canSend && stopped ? ANALYSIS_PLANNING_COPY.stopped : ANALYSIS_PLANNING_COPY.composerPlaceholder,
  };
}

function AnalysisRequestChat({
  description,
  files,
  analysis,
  source,
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
      <div className="relative min-h-0 flex-1">
        <div
          ref={follow.scrollRef}
          onScroll={follow.onScroll}
          className="absolute inset-0 overflow-y-auto px-3 py-3"
          data-testid="split-run-intent-chat-log"
        >
          <RequestMessage description={description} files={files} source={source} asChat />
          <WorkOrderIntentTranscript
            messages={analysis.view.messages}
            organizationId={analysis.organizationId}
            streaming={state.active}
            files={files}
            activities={analysis.view.activities}
          />
          {state.active ? (
            <AnalysisLiveWork
              machineStatus={analysis.view.machineStatus}
              organizationId={analysis.organizationId}
              canvasId={analysis.view.canvasId}
              executionId={analysis.view.executionId}
              activities={analysis.view.activities}
            />
          ) : null}
          {state.showSurvey && analysis.view.survey ? (
            <WorkOrderIntentSurvey survey={analysis.view.survey} onSubmit={analysis.onSubmitSurvey} />
          ) : null}
        </div>
        {follow.following ? null : (
          <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-intent-older" />
        )}
      </div>
      <form
        className={cn(SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME, "w-full flex-col items-stretch justify-center")}
        onSubmit={handleSubmit}
      >
        <label htmlFor="split-run-intent-composer" className="sr-only">
          {ANALYSIS_PLANNING_COPY.composerPlaceholder}
        </label>
        <div className="sp-user-note flex min-h-[3.5rem] w-full items-center gap-2 rounded-2xl border">
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
            className="min-h-[3.5rem] flex-1 resize-none rounded-2xl border-0 bg-transparent px-3.5 py-2.5 text-[13px] text-foreground shadow-none placeholder:text-muted-foreground focus-visible:ring-0"
            rows={2}
          />
          <Button
            type="submit"
            size="icon"
            className="mr-2 size-8 shrink-0 rounded-full"
            disabled={!analysis.canSend || !analysis.composer.trim()}
            aria-label={ANALYSIS_PLANNING_COPY.send}
          >
            <ArrowUp className="size-4" aria-hidden />
            <span className="sr-only">{ANALYSIS_PLANNING_COPY.send}</span>
          </Button>
        </div>
        {analysis.composerError ? (
          <p className="sp-error-shake mt-2 text-[12px] text-destructive" data-testid="split-run-intent-chat-error">
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
  source,
  asChat = false,
}: {
  description: string;
  files?: FilesFile[];
  source?: SplitRunSource;
  asChat?: boolean;
}) {
  const body = description.trim() ? (
    <WorkOrderDescription
      description={description}
      files={files}
      previewHeight={FALLBACK_COLLAPSED_MAX_HEIGHT_PX}
      fadeClassName="sp-user-note-fade"
    />
  ) : (
    <p className="text-[13px] text-muted-foreground">No request yet.</p>
  );

  if (!asChat) {
    return (
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <div className="max-w-[92%]">
          <div
            className="sp-user-note rounded-2xl border px-3.5 py-3"
            data-testid="split-run-description"
            aria-label="Request"
          >
            {body}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mb-4 flex w-full justify-end" data-testid="split-run-description">
      <div className="sp-user-note max-w-[92%] rounded-2xl border px-3.5 py-2.5">
        {source ? (
          <div className="mb-1">
            <WorkOrderSplitRunSource source={source} compact />
          </div>
        ) : null}
        {body}
      </div>
    </div>
  );
}
