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
import { Loader2, Sparkles } from "lucide-react";
import type { FormEvent } from "react";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentView } from "./createWithAgentTypes";
import { planningSessionPhase } from "./planningSessionActivity";
import { PhaseLogCard } from "./work-order-split-run/PhaseLogCard";

export type CreateWithAgentDialogProps = {
  open: boolean;
  workspaceName: string;
  organizationId?: string;
  view: CreateWithAgentView;
  onComposerChange: (value: string) => void;
  onSend: () => void;
  onDraftTitleChange: (title: string) => void;
  onDraftDescriptionChange: (description: string) => void;
  onCreateDraft: () => void;
  onSkipDraft: () => void;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
  onRequestClose: () => void;
  onCancelEnd: () => void;
  onConfirmEnd: () => void;
};

export function CreateWithAgentDialog({
  open,
  workspaceName,
  organizationId = "",
  view,
  onComposerChange,
  onSend,
  onDraftTitleChange,
  onDraftDescriptionChange,
  onCreateDraft,
  onSkipDraft,
  onOpenCreated,
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
          />
          <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-2">
            <CreateWithAgentStream
              organizationId={organizationId}
              view={view}
              onComposerChange={onComposerChange}
              onSend={onSend}
            />
            <CreateWithAgentWorkPane
              view={view}
              onDraftTitleChange={onDraftTitleChange}
              onDraftDescriptionChange={onDraftDescriptionChange}
              onCreateDraft={onCreateDraft}
              onSkipDraft={onSkipDraft}
              onOpenCreated={onOpenCreated}
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
}: {
  workspaceName: string;
  repository: string;
  machineStatus: CreateWithAgentView["machineStatus"];
  onEndSession: () => void;
}) {
  const starting = machineStatus === "starting";
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
          {starting ? <Loader2 className="size-3 animate-spin" aria-hidden /> : null}
          {machineStatusLabel(repository, machineStatus)}
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
      </div>
    </div>
  );
}

function CreateWithAgentStream({
  organizationId,
  view,
  onComposerChange,
  onSend,
}: {
  organizationId: string;
  view: CreateWithAgentView;
  onComposerChange: (value: string) => void;
  onSend: () => void;
}) {
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    onSend();
  };

  return (
    <section
      className="flex min-h-0 flex-col border-b border-border bg-muted/25 md:border-r md:border-b-0"
      data-testid="create-with-agent-stream"
    >
      <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        <PhaseLogCard
          phase={planningSessionPhase(view)}
          expanded
          collapsible={false}
          organizationId={organizationId}
          canvasId={view.canvasId}
        />
      </div>
      <form className="border-t border-border bg-background p-3" onSubmit={handleSubmit}>
        <label htmlFor="create-with-agent-composer" className="sr-only">
          {CREATE_WITH_AGENT_COPY.composerPlaceholder}
        </label>
        <div className="flex items-end gap-2">
          <Textarea
            id="create-with-agent-composer"
            data-testid="create-with-agent-composer"
            value={view.composer}
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
          <Button type="submit" size="sm" disabled={!view.composer.trim()}>
            {CREATE_WITH_AGENT_COPY.send}
          </Button>
        </div>
      </form>
    </section>
  );
}

function machineStatusLabel(repository: string, machineStatus: CreateWithAgentView["machineStatus"]): string {
  if (machineStatus === "starting") {
    return CREATE_WITH_AGENT_COPY.machineStarting;
  }
  const label =
    machineStatus === "waiting" ? CREATE_WITH_AGENT_COPY.machineWaiting : CREATE_WITH_AGENT_COPY.machineRunning;
  return repository ? `${repository} · ${label}` : label;
}

function CreateWithAgentWorkPane({
  view,
  onDraftTitleChange,
  onDraftDescriptionChange,
  onCreateDraft,
  onSkipDraft,
  onOpenCreated,
}: {
  view: CreateWithAgentView;
  onDraftTitleChange: (title: string) => void;
  onDraftDescriptionChange: (description: string) => void;
  onCreateDraft: () => void;
  onSkipDraft: () => void;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
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
      {view.right.kind === "preview" ? (
        <PreviewWorkPane order={view.right.order} onOpenCreated={onOpenCreated} />
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

function PreviewWorkPane({
  order,
  onOpenCreated,
}: {
  order: CreateWithAgentCreatedOrder;
  onOpenCreated: (order: CreateWithAgentCreatedOrder) => void;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col px-5 py-4" data-testid="create-with-agent-preview">
      <p className="text-[12px] font-medium text-muted-foreground">{order.key}</p>
      <h2 className="mt-2 text-[22px] font-semibold tracking-[-0.02em]">{order.title}</h2>
      <p className="mt-3 min-h-0 flex-1 overflow-y-auto text-[13px] leading-relaxed text-muted-foreground">
        {order.description}
      </p>
      <div className="mt-4 flex justify-end">
        <Button type="button" variant="outline" onClick={() => onOpenCreated(order)}>
          {CREATE_WITH_AGENT_COPY.openTask}
        </Button>
      </div>
    </div>
  );
}
