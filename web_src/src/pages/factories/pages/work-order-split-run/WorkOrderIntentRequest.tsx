import { useRef, useState, type FormEvent, type KeyboardEvent, type ReactNode } from "react";
import { ArrowUp } from "lucide-react";

import type { FilesFile } from "@/api-client";
import { SkillSlashFieldOverlay } from "@/components/AgentSidebar/SkillSlashFieldOverlay";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupTextarea } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Kbd } from "@/components/ui/kbd";
import type { UseSpeechDictationResult } from "@/hooks/useSpeechDictation";
import { useSpokenPhraseDictation, type SpokenPhraseField } from "@/hooks/useSpokenPhraseDictation";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { cn } from "@/lib/utils";
import { WORK_ORDER_FILE_ACCEPT } from "@/lib/workOrderFiles";
import { CreateWorkOrderRequestAttachButton } from "../../CreateWorkOrderRequestAttachButton";
import { CreateWorkOrderRequestAttachments } from "../../CreateWorkOrderRequestAttachments";
import { DictateButton } from "../../DictateButton";
import { PendingWorkOrderFileChips } from "../../PendingWorkOrderFileChips";
import { appendUploadedWorkOrderImages } from "../../lib/createWorkOrderRequestImages";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { FALLBACK_COLLAPSED_MAX_HEIGHT_PX } from "../../workOrderDescriptionOverflow";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import { REQUEST_CARD_CLASSNAME, REQUEST_CARD_FADE_CLASSNAME } from "./chatBubbleStyle";
import { ComposerPlanStack, type ComposerScore } from "./ComposerPlanControls";
import type { CreatedTaskHref } from "./CreatedTaskCard";
import { AnalysisLiveWork } from "./IntentAnalysisLiveWork";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { composerChipsWorking, type PlanChipStatus } from "./planChipStatus";
import { mergeAnalysisTranscriptFiles, useAnalysisComposerImages } from "./useAnalysisComposerImages";
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
  factoryId?: string;
  view: CreateWithAgentView;
  composer: string;
  composerError?: string;
  canSend: boolean;
  isUploading?: boolean;
  onComposerChange: (value: string) => void;
  onSend: (text?: string) => void | Promise<boolean>;
  onUploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
  onSubmitSurvey: (text: string) => void;
  planPaneOpen?: boolean;
  onTogglePlan?: () => void;
  canTogglePlan?: boolean;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  showClarity?: boolean;
  showConfidence?: boolean;
  planStatus?: PlanChipStatus;
  isAnalyzing?: boolean;
  closedDecision?: ReactNode;
  /** Model select for Start. The strip shows it on the settings row. */
  modelSelect?: ReactNode;
  /** Permalink for a task the agent split off this draft. */
  taskHref?: CreatedTaskHref;
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
  const chipsWorking = composerChipsWorking({
    isAnalyzing: analysis.isAnalyzing,
    score: analysis.clarity?.score ?? analysis.confidence?.score,
    machineStatus: analysis.view.machineStatus,
  });
  const images = useAnalysisComposerImages({
    disabled: !analysis.canSend,
    onUploadFiles: analysis.onUploadFiles,
  });
  const transcriptFiles = mergeAnalysisTranscriptFiles(files, images.transcriptFiles);

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
              files={transcriptFiles}
              activities={analysis.view.activities}
              taskHref={analysis.taskHref}
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
        {follow.showJumpToLatest ? (
          <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-intent-older" />
        ) : null}
      </div>
      <AnalysisComposer
        analysis={analysis}
        images={images}
        placeholder={state.placeholder}
        chipsWorking={chipsWorking}
        chatSolo={chatSolo}
        chatColumnClass={chatColumnClass}
      />
    </div>
  );
}

function useAnalysisComposerDictation(composer: string, onComposerChange: (next: string) => void) {
  const composerRef = useRef(composer);
  composerRef.current = composer;
  const fieldRef = useRef<SpokenPhraseField>({
    getValue: () => composerRef.current,
    setValue: onComposerChange,
  });
  fieldRef.current = {
    getValue: () => composerRef.current,
    setValue: (next) => {
      composerRef.current = next;
      onComposerChange(next);
    },
  };
  return { composerRef, dictation: useSpokenPhraseDictation(fieldRef) };
}

function AnalysisComposer({
  analysis,
  images,
  placeholder,
  chipsWorking,
  chatSolo,
  chatColumnClass,
}: {
  analysis: IntentAnalysisChat;
  images: ReturnType<typeof useAnalysisComposerImages>;
  placeholder: string;
  chipsWorking: boolean;
  chatSolo: boolean;
  chatColumnClass: string;
}) {
  const canSubmit = analysis.canSend && Boolean(analysis.composer.trim() || images.pending.length);
  const { composerRef, dictation } = useAnalysisComposerDictation(analysis.composer, analysis.onComposerChange);
  const send = async () => {
    if (!canSubmit) {
      return;
    }
    dictation.stop();
    const pending = images.takePending();
    const result = await analysis.onSend(appendUploadedWorkOrderImages(analysis.composer, pending));
    if (result === false) {
      images.restorePending(pending);
    }
  };
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    void send();
  };

  return (
    <div
      className={cn(chatColumnClass, SPLIT_RUN_CHAT_SCROLLBAR_GUTTER_CLASSNAME, "shrink-0", chatSolo ? "pb-4" : "pb-2")}
      data-testid="split-run-intent-chat-column"
    >
      <form
        className={cn(
          SPLIT_RUN_INTENT_PANE_FOOTER_CLASSNAME,
          "w-full flex-col items-stretch justify-center border-0 bg-transparent px-0 pt-0 pb-0",
        )}
        onSubmit={handleSubmit}
      >
        <Label htmlFor="split-run-intent-composer" className="sr-only">
          {ANALYSIS_PLANNING_COPY.composerPlaceholder}
        </Label>
        <div className="flex flex-col gap-2">
          <ComposerPlanStack
            open={Boolean(analysis.planPaneOpen)}
            clarity={analysis.clarity}
            confidence={analysis.confidence}
            showClarity={analysis.showClarity !== false}
            showConfidence={analysis.showConfidence !== false}
            isAnalyzing={chipsWorking}
            canTogglePlan={Boolean(analysis.canTogglePlan)}
            planStatus={analysis.planStatus}
            onToggle={analysis.onTogglePlan}
            actions={analysis.closedDecision}
            modelSelect={analysis.modelSelect}
          />
          <AnalysisComposerField
            analysis={analysis}
            images={images}
            dictation={dictation}
            placeholder={placeholder}
            canSubmit={canSubmit}
            composerRef={composerRef}
            onSend={() => void send()}
          />
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

function AnalysisComposerField({
  analysis,
  images,
  dictation,
  placeholder,
  canSubmit,
  composerRef,
  onSend,
}: {
  analysis: IntentAnalysisChat;
  images: ReturnType<typeof useAnalysisComposerImages>;
  dictation: UseSpeechDictationResult;
  placeholder: string;
  canSubmit: boolean;
  composerRef: { current: string };
  onSend: () => void;
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const skillKeyboardRef = useRef<((event: KeyboardEvent) => boolean) | null>(null);
  const [cursor, setCursor] = useState(0);
  const insertSkill = (next: { value: string; cursor: number }) => {
    composerRef.current = next.value;
    analysis.onComposerChange(next.value);
    setCursor(next.cursor);
    requestAnimationFrame(() => {
      const textarea = textareaRef.current;
      if (!textarea) {
        return;
      }
      textarea.focus();
      textarea.setSelectionRange(next.cursor, next.cursor);
    });
  };

  return (
    <div className="relative">
      {analysis.factoryId ? (
        <SkillSlashFieldOverlay
          organizationId={analysis.organizationId}
          factoryId={analysis.factoryId}
          value={analysis.composer}
          cursor={cursor}
          onInsert={insertSkill}
          keyboardRef={skillKeyboardRef}
        />
      ) : null}
      <InputGroup className="h-auto overflow-visible rounded-xl" data-testid="split-run-intent-composer-card">
        <InputGroupTextarea
          ref={textareaRef}
          id="split-run-intent-composer"
          data-testid="split-run-intent-composer"
          value={analysis.composer}
          placeholder={placeholder}
          disabled={!analysis.canSend}
          onChange={(event) => {
            composerRef.current = event.target.value;
            analysis.onComposerChange(event.target.value);
            setCursor(event.target.selectionStart);
          }}
          onSelect={(event) => setCursor(event.currentTarget.selectionStart)}
          onPaste={images.handlePaste}
          onKeyDown={(event) => {
            if (skillKeyboardRef.current?.(event)) {
              return;
            }
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              onSend();
            }
          }}
          className="min-h-[4.2rem] py-2 text-[13px]"
          rows={2}
        />
        <InputGroupAddon align="block-end" className="items-end justify-between gap-3 overflow-visible pb-1.5">
          <AnalysisComposerAddons analysis={analysis} images={images} dictation={dictation} />
          <div className="flex items-center gap-1.5">
            <Kbd className="hidden sm:inline-flex" data-testid="split-run-intent-composer-kbd">
              {ANALYSIS_PLANNING_COPY.sendShortcut}
            </Kbd>
            <InputGroupButton
              type="submit"
              variant="default"
              size="icon-sm"
              className="rounded-full"
              disabled={!canSubmit}
              aria-label={ANALYSIS_PLANNING_COPY.send}
              data-testid="split-run-intent-composer-send"
            >
              <ArrowUp className="size-4" aria-hidden />
            </InputGroupButton>
          </div>
        </InputGroupAddon>
      </InputGroup>
    </div>
  );
}

function AnalysisComposerAddons({
  analysis,
  images,
  dictation,
}: {
  analysis: IntentAnalysisChat;
  images: ReturnType<typeof useAnalysisComposerImages>;
  dictation: UseSpeechDictationResult;
}) {
  return (
    <div className="create-work-order-request-attachments flex min-w-0 items-end gap-2 overflow-visible">
      {analysis.onUploadFiles ? (
        <CreateWorkOrderRequestAttachButton
          accept={WORK_ORDER_FILE_ACCEPT}
          disabled={!images.canAttach}
          onAttach={(files) => void images.attach(files)}
        />
      ) : null}
      <DictateButton dictation={dictation} copy={ANALYSIS_PLANNING_COPY} disabled={!analysis.canSend} />
      {images.previewImages.length > 0 ? (
        <CreateWorkOrderRequestAttachments images={images.previewImages} onRemove={images.remove} />
      ) : null}
      {images.pendingFiles.length > 0 ? (
        <PendingWorkOrderFileChips files={images.pendingFiles} onRemove={images.remove} />
      ) : null}
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
      fadeClassName={asChat ? REQUEST_CARD_FADE_CLASSNAME : undefined}
    />
  ) : (
    <p className="text-[13px] text-muted-foreground">No request yet.</p>
  );

  if (!asChat) {
    return (
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <div className="max-w-[92%]">
          <div
            className="rounded-2xl border bg-card px-3.5 py-3"
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
      <div className={cn(REQUEST_CARD_CLASSNAME, "max-w-[85%]")}>
        {source ? (
          <div className="mb-2">
            <WorkOrderSplitRunSource source={source} compact />
          </div>
        ) : null}
        {body}
      </div>
    </div>
  );
}
