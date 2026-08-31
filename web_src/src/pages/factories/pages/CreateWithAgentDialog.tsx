import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { cn } from "@/lib/utils";
import { Loader2, Sparkles } from "lucide-react";
import type { FormEvent } from "react";

import type { WorkOrderSurveyAnswerInput } from "../lib/workOrderSurvey";
import { WorkOrderSurveyCard } from "../WorkOrderSurveyCard";
import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";

export type CreateWithAgentDialogProps = {
  open: boolean;
  workspaceName: string;
  view: CreateWithAgentView;
  onComposerChange: (value: string) => void;
  onSend: () => void;
  onAnswerSurvey: (surveyId: string, answers: WorkOrderSurveyAnswerInput[]) => void;
  onDraftTitleChange: (title: string) => void;
  onDraftDescriptionChange: (description: string) => void;
  onCreateDraft: () => void;
  onSkipDraft: () => void;
  onWorkOnNew: () => void;
  onSelectCreated: (orderId: string) => void;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
  onBackToList: () => void;
  onRequestClose: () => void;
  onCancelEnd: () => void;
  onConfirmEnd: () => void;
};

export function CreateWithAgentDialog({
  open,
  workspaceName,
  view,
  onComposerChange,
  onSend,
  onAnswerSurvey,
  onDraftTitleChange,
  onDraftDescriptionChange,
  onCreateDraft,
  onSkipDraft,
  onWorkOnNew,
  onSelectCreated,
  onOpenCreated,
  onBackToList,
  onRequestClose,
  onCancelEnd,
  onConfirmEnd,
}: CreateWithAgentDialogProps) {
  return (
    <>
      <Dialog open={open} onOpenChange={(nextOpen) => (nextOpen ? undefined : onRequestClose())}>
        <DialogContent
          showCloseButton={false}
          size="large"
          className="flex h-[min(90vh,720px)] w-[calc(100%-2rem)] max-w-6xl flex-col gap-0 overflow-hidden p-0 sm:rounded-xl"
          data-testid="create-with-agent-dialog"
        >
          <CreateWithAgentHeader
            workspaceName={workspaceName}
            repository={view.repository}
            machineStatus={view.machineStatus}
            onEndSession={onRequestClose}
            onClose={onRequestClose}
          />
          <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-2">
            <CreateWithAgentChat
              view={view}
              onComposerChange={onComposerChange}
              onSend={onSend}
              onAnswerSurvey={onAnswerSurvey}
            />
            <CreateWithAgentWorkPane
              view={view}
              onDraftTitleChange={onDraftTitleChange}
              onDraftDescriptionChange={onDraftDescriptionChange}
              onCreateDraft={onCreateDraft}
              onSkipDraft={onSkipDraft}
              onWorkOnNew={onWorkOnNew}
              onSelectCreated={onSelectCreated}
              onOpenCreated={onOpenCreated}
              onBackToList={onBackToList}
            />
          </div>
        </DialogContent>
      </Dialog>
      <AlertDialog open={view.endConfirmOpen} onOpenChange={(nextOpen) => (nextOpen ? undefined : onCancelEnd())}>
        <AlertDialogContent data-testid="create-with-agent-end-confirm">
          <AlertDialogHeader>
            <AlertDialogTitle>{CREATE_WITH_AGENT_COPY.endSessionAsk}</AlertDialogTitle>
            <AlertDialogDescription>{CREATE_WITH_AGENT_COPY.endSessionBody}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={onCancelEnd}>{CREATE_WITH_AGENT_COPY.keepSession}</AlertDialogCancel>
            <AlertDialogAction onClick={onConfirmEnd}>{CREATE_WITH_AGENT_COPY.endSessionConfirm}</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function CreateWithAgentHeader({
  workspaceName,
  repository,
  machineStatus,
  onEndSession,
  onClose,
}: {
  workspaceName: string;
  repository: string;
  machineStatus: CreateWithAgentView["machineStatus"];
  onEndSession: () => void;
  onClose: () => void;
}) {
  const running = machineStatus === "running";
  return (
    <div
      className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5"
      data-testid="create-with-agent-header"
    >
      <div className="flex min-w-0 items-center gap-2 text-[13px] text-muted-foreground">
        <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted">
          <Sparkles className="size-3" aria-hidden />
        </span>
        <span className="truncate text-foreground">{workspaceName}</span>
        <span aria-hidden>/</span>
        <DialogTitle className="truncate text-[13px] font-medium text-foreground">
          {CREATE_WITH_AGENT_COPY.title}
        </DialogTitle>
        <DialogDescription className="sr-only">Create tasks with an agent in this workspace.</DialogDescription>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span
          className="hidden items-center gap-1.5 text-[12px] text-muted-foreground sm:flex"
          data-testid="create-with-agent-machine"
        >
          {running ? null : <Loader2 className="size-3 animate-spin" aria-hidden />}
          {running
            ? `${repository} · ${CREATE_WITH_AGENT_COPY.machineRunning}`
            : CREATE_WITH_AGENT_COPY.machineStarting}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 px-2.5 text-[12px]"
          data-testid="create-with-agent-end"
          onClick={onEndSession}
        >
          {CREATE_WITH_AGENT_COPY.endSession}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2.5 text-[12px]"
          data-testid="create-with-agent-close"
          onClick={onClose}
        >
          {CREATE_WITH_AGENT_COPY.close}
        </Button>
      </div>
    </div>
  );
}

function CreateWithAgentChat({
  view,
  onComposerChange,
  onSend,
  onAnswerSurvey,
}: {
  view: CreateWithAgentView;
  onComposerChange: (value: string) => void;
  onSend: () => void;
  onAnswerSurvey: (surveyId: string, answers: WorkOrderSurveyAnswerInput[]) => void;
}) {
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    onSend();
  };

  return (
    <section
      className="flex min-h-0 flex-col border-b border-border md:border-r md:border-b-0"
      data-testid="create-with-agent-chat"
    >
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-4">
        {view.machineStatus === "starting" ? (
          <p className="text-[13px] text-muted-foreground">{CREATE_WITH_AGENT_COPY.machineStarting}.</p>
        ) : null}
        {view.messages.map((message) => (
          <ChatMessageBlock key={message.id} message={message} onAnswerSurvey={onAnswerSurvey} />
        ))}
      </div>
      <form className="border-t border-border p-3" onSubmit={handleSubmit}>
        <label htmlFor="create-with-agent-composer" className="sr-only">
          {CREATE_WITH_AGENT_COPY.composerPlaceholder}
        </label>
        <div className="flex items-end gap-2">
          <Textarea
            id="create-with-agent-composer"
            data-testid="create-with-agent-composer"
            value={view.composer}
            disabled={view.machineStatus !== "running"}
            placeholder={CREATE_WITH_AGENT_COPY.composerPlaceholder}
            onChange={(event) => onComposerChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                onSend();
              }
            }}
            className="min-h-[44px] resize-none text-[13px]"
            rows={2}
          />
          <Button type="submit" size="sm" disabled={view.machineStatus !== "running" || !view.composer.trim()}>
            {CREATE_WITH_AGENT_COPY.send}
          </Button>
        </div>
      </form>
    </section>
  );
}

function ChatMessageBlock({
  message,
  onAnswerSurvey,
}: {
  message: CreateWithAgentMessage;
  onAnswerSurvey: (surveyId: string, answers: WorkOrderSurveyAnswerInput[]) => void;
}) {
  if (message.kind === "survey") {
    if (message.answered) {
      return (
        <p className="text-[12px] text-muted-foreground" data-testid={`create-with-agent-survey-done-${message.id}`}>
          Answer sent.
        </p>
      );
    }
    return (
      <WorkOrderSurveyCard
        survey={message.survey}
        help={CREATE_WITH_AGENT_COPY.surveyHelp}
        onSubmit={(answers) => onAnswerSurvey(message.survey.id, answers)}
      />
    );
  }

  return (
    <div
      className={cn(
        "max-w-[92%] rounded-lg px-3 py-2 text-[13px] leading-snug",
        message.role === "user" ? "ml-auto bg-primary text-primary-foreground" : "bg-muted text-foreground",
      )}
      data-testid={`create-with-agent-message-${message.id}`}
    >
      {message.text}
    </div>
  );
}

function CreateWithAgentWorkPane({
  view,
  onDraftTitleChange,
  onDraftDescriptionChange,
  onCreateDraft,
  onSkipDraft,
  onWorkOnNew,
  onSelectCreated,
  onOpenCreated,
  onBackToList,
}: {
  view: CreateWithAgentView;
  onDraftTitleChange: (title: string) => void;
  onDraftDescriptionChange: (description: string) => void;
  onCreateDraft: () => void;
  onSkipDraft: () => void;
  onWorkOnNew: () => void;
  onSelectCreated: (orderId: string) => void;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
  onBackToList: () => void;
}) {
  return (
    <section className="flex min-h-0 flex-col bg-muted/20" data-testid="create-with-agent-work">
      {view.right.kind === "empty" ? <EmptyWorkPane /> : null}
      {view.right.kind === "draft" ? (
        <DraftWorkPane
          title={view.right.draft.title}
          description={view.right.draft.description}
          onTitleChange={onDraftTitleChange}
          onDescriptionChange={onDraftDescriptionChange}
          onCreate={onCreateDraft}
          onSkip={onSkipDraft}
        />
      ) : null}
      {view.right.kind === "list" ? (
        <ListWorkPane created={view.created} onWorkOnNew={onWorkOnNew} onSelectCreated={onSelectCreated} />
      ) : null}
      {view.right.kind === "preview" ? (
        <PreviewWorkPane
          order={view.right.order}
          showList={view.created.length > 0}
          onBackToList={onBackToList}
          onOpenCreated={onOpenCreated}
          onWorkOnNew={onWorkOnNew}
        />
      ) : null}
    </section>
  );
}

function EmptyWorkPane() {
  return (
    <div
      className="flex flex-1 flex-col items-center justify-center px-8 text-center"
      data-testid="create-with-agent-empty"
    >
      <p className="text-[15px] font-medium tracking-[-0.01em] text-foreground">
        {CREATE_WITH_AGENT_COPY.emptyHeadline}
      </p>
      <p className="mt-1 max-w-sm text-[13px] text-muted-foreground">{CREATE_WITH_AGENT_COPY.emptyBody}</p>
    </div>
  );
}

function DraftWorkPane({
  title,
  description,
  onTitleChange,
  onDescriptionChange,
  onCreate,
  onSkip,
}: {
  title: string;
  description: string;
  onTitleChange: (title: string) => void;
  onDescriptionChange: (description: string) => void;
  onCreate: () => void;
  onSkip: () => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col px-5 py-4" data-testid="create-with-agent-draft">
      <p className="text-[12px] font-medium text-muted-foreground">{CREATE_WITH_AGENT_COPY.draftLabel}</p>
      <Input
        value={title}
        onChange={(event) => onTitleChange(event.target.value)}
        aria-label="Task title"
        data-testid="create-with-agent-draft-title"
        className="mt-3 h-auto border-0 bg-transparent p-0 text-[22px] font-semibold tracking-[-0.02em] shadow-none focus-visible:ring-0"
      />
      <Textarea
        value={description}
        onChange={(event) => onDescriptionChange(event.target.value)}
        aria-label="Task description"
        data-testid="create-with-agent-draft-description"
        className="mt-3 min-h-0 flex-1 resize-none border-0 bg-transparent p-0 text-[13px] shadow-none focus-visible:ring-0"
      />
      <div className="mt-4 flex justify-end gap-2">
        <Button type="button" variant="ghost" onClick={onSkip}>
          {CREATE_WITH_AGENT_COPY.skip}
        </Button>
        <Button type="button" data-testid="create-with-agent-create" disabled={!title.trim()} onClick={onCreate}>
          {CREATE_WITH_AGENT_COPY.create}
        </Button>
      </div>
    </div>
  );
}

function ListWorkPane({
  created,
  onWorkOnNew,
  onSelectCreated,
}: {
  created: CreateWithAgentCreatedOrder[];
  onWorkOnNew: () => void;
  onSelectCreated: (orderId: string) => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col px-5 py-4" data-testid="create-with-agent-list">
      <div className="flex items-center justify-between gap-2">
        <p className="text-[13px] font-medium text-foreground">{CREATE_WITH_AGENT_COPY.sessionListHeadline}</p>
        <Button type="button" size="sm" data-testid="create-with-agent-work-on-new" onClick={onWorkOnNew}>
          {CREATE_WITH_AGENT_COPY.workOnNew}
        </Button>
      </div>
      <ul className="mt-3 min-h-0 flex-1 space-y-1 overflow-y-auto">
        {created.map((order) => (
          <li key={order.id}>
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left hover:bg-accent"
              data-testid={`create-with-agent-created-${order.id}`}
              onClick={() => onSelectCreated(order.id)}
            >
              <span className="shrink-0 text-[12px] text-muted-foreground">{order.key}</span>
              <span className="min-w-0 flex-1 truncate text-[13px]">{order.title}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function PreviewWorkPane({
  order,
  showList,
  onBackToList,
  onOpenCreated,
  onWorkOnNew,
}: {
  order: CreateWithAgentCreatedOrder;
  showList: boolean;
  onBackToList: () => void;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
  onWorkOnNew: () => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col px-5 py-4" data-testid="create-with-agent-preview">
      <p className="text-[12px] font-medium text-muted-foreground">{order.key}</p>
      <h2 className="mt-2 text-[22px] font-semibold tracking-[-0.02em]">{order.title}</h2>
      <p className="mt-3 min-h-0 flex-1 overflow-y-auto text-[13px] leading-relaxed text-muted-foreground">
        {order.description}
      </p>
      <div className="mt-4 flex flex-wrap justify-end gap-2">
        {showList ? (
          <Button type="button" variant="ghost" onClick={onBackToList}>
            {CREATE_WITH_AGENT_COPY.sessionListHeadline}
          </Button>
        ) : null}
        <Button type="button" variant="outline" onClick={() => onOpenCreated(order)}>
          {CREATE_WITH_AGENT_COPY.openTask}
        </Button>
        <Button type="button" onClick={onWorkOnNew}>
          {CREATE_WITH_AGENT_COPY.workOnNew}
        </Button>
      </div>
    </div>
  );
}
