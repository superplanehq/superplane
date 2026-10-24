import { useCallback, useState, type ChangeEvent, type FormEvent, type KeyboardEvent } from "react";
import { ArrowUp, ClipboardList, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { EmptyState } from "@/ui/emptyState";
import { Separator } from "@/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { cn } from "@/lib/utils";

import { FactoryAgentSidebarShell } from "../../agent/FactoryAgentSidebarShell";
import { factoryAgentIconButtonClassName, factoryAgentMessageClassName } from "../../agent/factoryAgentChrome";
import {
  BUILT_IN_AGENT_COPY,
  deleteConfirmTitle,
  type BuiltInAgentMessage,
  type BuiltInAgentProposedPlan,
  type BuiltInAgentTask,
} from "./builtInAgentMocks";

export interface BuiltInAgentPanelProps {
  tasks: BuiltInAgentTask[];
  transcript: BuiltInAgentMessage[];
  pendingDeleteId: string | null;
  pendingPlan: BuiltInAgentProposedPlan | null;
  hasError: boolean;
  onClose: () => void;
  onCreateTask: (title: string) => void;
  onDeleteTask: (taskId: string) => void;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;
  onSendMessage: (content: string) => void;
  onAcceptPlan: () => void;
  onDismissPlan: () => void;
  onRetryAfterError: () => void;
}

export function BuiltInAgentPanel({
  tasks,
  transcript,
  pendingDeleteId,
  pendingPlan,
  hasError,
  onClose,
  onCreateTask,
  onDeleteTask,
  onConfirmDelete,
  onCancelDelete,
  onSendMessage,
  onAcceptPlan,
  onDismissPlan,
  onRetryAfterError,
}: BuiltInAgentPanelProps) {
  const pendingDelete = tasks.find((task) => task.id === pendingDeleteId) ?? null;

  return (
    <FactoryAgentSidebarShell>
      <PanelHeader onClose={onClose} />
      <Separator />
      {hasError ? <ErrorBanner onRetry={onRetryAfterError} /> : null}
      <TranscriptList messages={transcript} />
      <Separator />
      <TaskSection
        tasks={tasks}
        pendingPlan={pendingPlan}
        onCreateTask={onCreateTask}
        onDeleteTask={onDeleteTask}
        onAcceptPlan={onAcceptPlan}
        onDismissPlan={onDismissPlan}
      />
      <Composer onSend={onSendMessage} disabled={hasError} />
      <DeleteConfirmDialog
        task={pendingDelete}
        open={pendingDeleteId != null}
        onConfirm={onConfirmDelete}
        onOpenChange={(open) => {
          if (!open) {
            onCancelDelete();
          }
        }}
      />
    </FactoryAgentSidebarShell>
  );
}

function PanelHeader({ onClose }: { onClose: () => void }) {
  return (
    <header className="flex shrink-0 items-center justify-between gap-2 px-3 py-2.5">
      <h2 className="min-w-0 truncate text-[13px] font-medium text-foreground">{BUILT_IN_AGENT_COPY.agentName}</h2>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className={factoryAgentIconButtonClassName}
        aria-label={BUILT_IN_AGENT_COPY.closePanel}
        onClick={onClose}
      >
        <X className="size-3.5" aria-hidden />
      </Button>
    </header>
  );
}

function ErrorBanner({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex shrink-0 items-start justify-between gap-2 px-3 py-2" role="alert">
      <p className={cn(factoryAgentMessageClassName, "text-destructive")}>{BUILT_IN_AGENT_COPY.errorMessage}</p>
      <Button type="button" variant="outline" size="xs" onClick={onRetry}>
        {BUILT_IN_AGENT_COPY.retry}
      </Button>
    </div>
  );
}

function TranscriptList({ messages }: { messages: BuiltInAgentMessage[] }) {
  return (
    <div className="min-h-0 min-w-0 flex-1 overflow-y-auto px-3" data-testid="built-in-agent-transcript">
      <div className="mx-auto w-full max-w-[800px] space-y-2 py-3">
        {messages.map((message) => (
          <p
            key={message.id}
            className={cn(
              factoryAgentMessageClassName,
              message.role === "user" ? "rounded-md bg-muted px-3 py-2 text-foreground" : "text-foreground",
            )}
          >
            {message.content}
          </p>
        ))}
      </div>
    </div>
  );
}

function TaskSection({
  tasks,
  pendingPlan,
  onCreateTask,
  onDeleteTask,
  onAcceptPlan,
  onDismissPlan,
}: {
  tasks: BuiltInAgentTask[];
  pendingPlan: BuiltInAgentProposedPlan | null;
  onCreateTask: (title: string) => void;
  onDeleteTask: (taskId: string) => void;
  onAcceptPlan: () => void;
  onDismissPlan: () => void;
}) {
  return (
    <section
      className="flex max-h-[45%] min-h-0 shrink-0 flex-col overflow-hidden px-3 py-2"
      aria-label={BUILT_IN_AGENT_COPY.tasksHeading}
    >
      <h3 className="mb-2 shrink-0 text-[13px] font-medium text-foreground">{BUILT_IN_AGENT_COPY.tasksHeading}</h3>
      <div className="min-h-0 flex-1 overflow-y-auto">
        {tasks.length === 0 ? (
          <EmptyState
            icon={ClipboardList}
            title={BUILT_IN_AGENT_COPY.emptyTitle}
            description={BUILT_IN_AGENT_COPY.emptyDescription}
            compact
            tone="neutral"
          />
        ) : (
          <ul className="flex flex-col gap-1">
            {tasks.map((task) => (
              <TaskRow key={task.id} task={task} onDelete={() => onDeleteTask(task.id)} />
            ))}
          </ul>
        )}
        <CreateTaskRow onCreateTask={onCreateTask} />
        {pendingPlan ? <PlanProposal tasks={tasks} plan={pendingPlan} /> : null}
      </div>
      {pendingPlan ? <PlanProposalActions onAccept={onAcceptPlan} onDismiss={onDismissPlan} /> : null}
    </section>
  );
}

function TaskRow({ task, onDelete }: { task: BuiltInAgentTask; onDelete: () => void }) {
  return (
    <li className="group flex items-center gap-2 rounded-md px-1.5 py-1 hover:bg-accent">
      <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{task.title}</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className={cn(
              factoryAgentIconButtonClassName,
              "opacity-0 group-hover:opacity-100 group-focus-within:opacity-100",
            )}
            aria-label={BUILT_IN_AGENT_COPY.deleteTask}
            onClick={onDelete}
          >
            <Trash2 className="size-3.5" aria-hidden />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{BUILT_IN_AGENT_COPY.deleteTask}</TooltipContent>
      </Tooltip>
    </li>
  );
}

function CreateTaskRow({ onCreateTask }: { onCreateTask: (title: string) => void }) {
  const [title, setTitle] = useState("");

  function submit() {
    const next = title.trim();
    if (!next) {
      return;
    }
    onCreateTask(next);
    setTitle("");
  }

  return (
    <form
      className="mt-2 flex items-center gap-2"
      onSubmit={(event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        submit();
      }}
    >
      <Label htmlFor="built-in-agent-create-title" className="sr-only">
        {BUILT_IN_AGENT_COPY.createTaskPlaceholder}
      </Label>
      <Input
        id="built-in-agent-create-title"
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        placeholder={BUILT_IN_AGENT_COPY.createTaskPlaceholder}
      />
      <Button type="submit" size="xs">
        {BUILT_IN_AGENT_COPY.createTask}
      </Button>
    </form>
  );
}

function PlanProposal({ tasks, plan }: { tasks: BuiltInAgentTask[]; plan: BuiltInAgentProposedPlan }) {
  const ordered = plan.taskIds
    .map((taskId) => tasks.find((task) => task.id === taskId))
    .filter((task): task is BuiltInAgentTask => Boolean(task));

  return (
    <div className="mt-2 rounded-md border border-border bg-card p-2.5" data-testid="built-in-agent-plan">
      <p className="text-[13px] font-medium text-foreground">{BUILT_IN_AGENT_COPY.planTitle}</p>
      <p className="mt-1 text-[12px] text-muted-foreground">{plan.reason}</p>
      <ol className="mt-2 list-decimal space-y-1 pl-4">
        {ordered.map((item) => (
          <li key={item.id} className="text-[13px] text-foreground">
            {item.title}
          </li>
        ))}
      </ol>
    </div>
  );
}

function PlanProposalActions({ onAccept, onDismiss }: { onAccept: () => void; onDismiss: () => void }) {
  return (
    <div className="mt-2 flex shrink-0 items-center gap-2">
      <Button type="button" size="xs" onClick={onAccept}>
        {BUILT_IN_AGENT_COPY.acceptPlan}
      </Button>
      <Button type="button" variant="outline" size="xs" onClick={onDismiss}>
        {BUILT_IN_AGENT_COPY.dismissPlan}
      </Button>
    </div>
  );
}

function Composer({ onSend, disabled }: { onSend: (content: string) => void; disabled: boolean }) {
  const [value, setValue] = useState("");

  const handleChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setValue(event.target.value);
  }, []);

  const handleSend = useCallback(() => {
    const content = value.trim();
    if (!content || disabled) {
      return;
    }
    onSend(content);
    setValue("");
  }, [disabled, onSend, value]);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  return (
    <footer className="px-3 pb-3 pt-2">
      <Label htmlFor="built-in-agent-composer" className="sr-only">
        {BUILT_IN_AGENT_COPY.composerPlaceholder}
      </Label>
      <div className="overflow-hidden rounded-lg bg-card shadow-sm outline outline-1 outline-border">
        <Textarea
          id="built-in-agent-composer"
          value={value}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={BUILT_IN_AGENT_COPY.composerPlaceholder}
          disabled={disabled}
          rows={3}
          className="min-h-16 border-0 shadow-none focus-visible:ring-0"
        />
        <div className="flex items-center justify-between gap-2 px-2 pb-2">
          <p className="min-w-0 truncate text-[11px] text-muted-foreground">{BUILT_IN_AGENT_COPY.composerHint}</p>
          <Button
            type="button"
            size="icon-xs"
            aria-label={BUILT_IN_AGENT_COPY.send}
            disabled={disabled || !value.trim()}
            onClick={handleSend}
          >
            <ArrowUp className="size-3.5" aria-hidden />
          </Button>
        </div>
      </div>
    </footer>
  );
}

function DeleteConfirmDialog({
  task,
  open,
  onConfirm,
  onOpenChange,
}: {
  task: BuiltInAgentTask | null;
  open: boolean;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{task ? deleteConfirmTitle(task.title) : BUILT_IN_AGENT_COPY.deleteTask}</AlertDialogTitle>
          <AlertDialogDescription>{BUILT_IN_AGENT_COPY.deleteConfirmBody}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{BUILT_IN_AGENT_COPY.keepTask}</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>{BUILT_IN_AGENT_COPY.deleteTask}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
