import type { FormEvent, ReactNode } from "react";
import { ArrowUp } from "lucide-react";

import type { FilesFile } from "@/api-client";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupTextarea } from "@/components/ui/input-group";
import { Kbd } from "@/components/ui/kbd";
import { cn } from "@/lib/utils";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import { ComposerPlanStack } from "./ComposerPlanControls";
import { AnalysisLiveWork } from "./IntentAnalysisLiveWork";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { composerChipsWorking, type PlanChipStatus } from "./planChipStatus";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { useFollowLogScroll } from "./useFollowLogScroll";
import {
  SPLIT_RUN_CHAT_COLUMN_CLASSNAME,
  SPLIT_RUN_CHAT_SCROLLBAR_GUTTER_CLASSNAME,
  SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME,
} from "./splitRunPopupModel";
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
  planPaneOpen?: boolean;
  onTogglePlan?: () => void;
  canTogglePlan?: boolean;
  clarityExpanded?: boolean;
  onToggleClarity?: () => void;
  latestPlanScore?: number;
  latestPlanSummary?: string;
  planStatus?: PlanChipStatus;
  isAnalyzing?: boolean;
  closedDecision?: ReactNode;
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
  const chatSolo = !analysis.planPaneOpen;
  const chatColumnClass = SPLIT_RUN_CHAT_COLUMN_CLASSNAME;
  const follow = useFollowLogScroll<HTMLDivElement>(state.followKey, analysis.view.messages.length, {
    resumeOnBottom: true,
  });
  const hasClarity = analysis.latestPlanScore != null;
  const chipsWorking = composerChipsWorking({
    isAnalyzing: analysis.isAnalyzing,
    score: analysis.latestPlanScore,
    machineStatus: analysis.view.machineStatus,
  });
  const planStack = (
    <ComposerPlanStack
      open={Boolean(analysis.planPaneOpen)}
      score={analysis.latestPlanScore}
      scoreSummary={analysis.latestPlanSummary}
      isAnalyzing={chipsWorking}
      canTogglePlan={Boolean(hasClarity && analysis.canTogglePlan)}
      planStatus={analysis.planStatus}
      onToggle={analysis.onTogglePlan}
      summaryOpen={analysis.clarityExpanded}
      onToggleSummary={analysis.onToggleClarity}
      actions={analysis.closedDecision}
    />
  );
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
          className={cn("absolute inset-0", SPLIT_RUN_CHAT_SCROLLBAR_GUTTER_CLASSNAME)}
          data-testid="split-run-intent-chat-log"
        >
          <div className={cn(chatColumnClass, chatSolo ? "py-6" : "py-3")} data-testid="split-run-intent-chat-column">
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
        </div>
        {follow.following ? null : (
          <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-intent-older" />
        )}
      </div>
      <div
        className={cn(
          chatColumnClass,
          SPLIT_RUN_CHAT_SCROLLBAR_GUTTER_CLASSNAME,
          "shrink-0",
          chatSolo ? "pb-4" : "pb-2",
        )}
        data-testid="split-run-intent-chat-column"
      >
        <form
          className={cn(
            SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME,
            "w-full flex-col items-stretch justify-center border-0 bg-transparent px-0 pt-0 pb-0",
          )}
          onSubmit={handleSubmit}
        >
          <label htmlFor="split-run-intent-composer" className="sr-only">
            {ANALYSIS_PLANNING_COPY.composerPlaceholder}
          </label>
          <div className="flex flex-col gap-2">
            {hasClarity ? planStack : null}
            <InputGroup className="h-auto rounded-xl" data-testid="split-run-intent-composer-card">
              <InputGroupTextarea
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
                className="min-h-[4.2rem] py-2 text-[13px]"
                rows={2}
              />
              <InputGroupAddon align="block-end" className="pb-1.5">
                <div className="ms-auto flex items-center gap-1.5">
                  <Kbd className="hidden sm:inline-flex" data-testid="split-run-intent-composer-kbd">
                    {ANALYSIS_PLANNING_COPY.sendShortcut}
                  </Kbd>
                  <InputGroupButton
                    type="submit"
                    variant="default"
                    size="icon-sm"
                    className="rounded-full"
                    disabled={!analysis.canSend || !analysis.composer.trim()}
                    aria-label={ANALYSIS_PLANNING_COPY.send}
                    data-testid="split-run-intent-composer-send"
                  >
                    <ArrowUp className="size-4" aria-hidden />
                  </InputGroupButton>
                </div>
              </InputGroupAddon>
            </InputGroup>
          </div>
          {analysis.composerError ? (
            <p className="sp-error-shake mt-2 text-[12px] text-destructive" data-testid="split-run-intent-chat-error">
              {analysis.composerError}
            </p>
          ) : null}
        </form>
      </div>
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
    <div className="mb-3 flex w-full justify-end" data-testid="split-run-description">
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
