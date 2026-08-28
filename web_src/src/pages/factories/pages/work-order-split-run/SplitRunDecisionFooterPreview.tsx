import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CheckCircle2, CircleX, ExternalLink, FileText, Hourglass, RotateCcw, XIcon } from "lucide-react";

/**
 * Storybook-only: close actions in a note-style footer. Header stays Close
 * only. Running has no footer. Production popup stays unchanged until one
 * look is chosen.
 */
export type DecisionFooterKind =
  | "draft"
  | "running"
  | "waiting"
  | "statusNote"
  | "failed"
  | "completed"
  | "rejected"
  | "closedFailed";

type DecisionAction = { id: string; label: string; emphasis: "primary" | "quiet" };

type DecisionCopy = {
  title: string;
  headline: string;
  text: string;
  tone: "draft" | "waiting" | "failed" | "done" | "rejected";
  actions: DecisionAction[];
  cta?: { label: string; href: string };
};

const COPY: Record<Exclude<DecisionFooterKind, "running">, DecisionCopy> = {
  draft: {
    title: "Add refund reconciliation test",
    headline: "This task is ready to start",
    text: "Review the details. Change anything you need. Then click Start to send it to the line.",
    tone: "draft",
    actions: [
      { id: "reject", label: "Reject", emphasis: "quiet" },
      { id: "start", label: "Start", emphasis: "primary" },
    ],
  },
  waiting: {
    title: "Add refund reconciliation test",
    headline: "This task waits on a person",
    text: "No automation is running. Click Approve if the result is good. Click Reject to close this task as rejected.",
    tone: "waiting",
    actions: [
      { id: "reject", label: "Reject", emphasis: "quiet" },
      { id: "approve", label: "Approve", emphasis: "primary" },
    ],
  },
  statusNote: {
    title: "Add refund reconciliation test",
    headline: "Waiting for user review",
    text: "This automation finished and opened this pull request. Tag @superplaneagent in a comment to request changes. This task closes when the pull request is closed or merged.",
    tone: "waiting",
    cta: { label: "Review PR", href: "https://github.com/superplanehq/superplane/pull/6812" },
    actions: [
      { id: "reject", label: "Reject", emphasis: "quiet" },
      { id: "approve", label: "Approve", emphasis: "primary" },
    ],
  },
  failed: {
    title: "Add refund reconciliation test",
    headline: "Implement did not pass",
    text: "This automation failed. Open the run to review the error. Fix the automation, then click Rerun. Or close this task.",
    tone: "failed",
    cta: { label: "Review the run", href: "/run/implement" },
    actions: [
      { id: "reject", label: "Reject", emphasis: "quiet" },
      { id: "rerun", label: "Rerun", emphasis: "primary" },
    ],
  },
  completed: {
    title: "Add refund reconciliation test",
    headline: "This task is completed",
    text: "Reopen this task if more work is needed.",
    tone: "done",
    actions: [{ id: "reopen", label: "Reopen", emphasis: "primary" }],
  },
  rejected: {
    title: "Add refund reconciliation test",
    headline: "This task is rejected",
    text: "Reopen this task if the work should continue.",
    tone: "rejected",
    actions: [{ id: "reopen", label: "Reopen", emphasis: "primary" }],
  },
  closedFailed: {
    title: "Add refund reconciliation test",
    headline: "This task is closed as failed",
    text: "Reopen this task to start the line again.",
    tone: "failed",
    actions: [{ id: "reopen", label: "Reopen", emphasis: "primary" }],
  },
};

const TONE = {
  draft: {
    strip: "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)]",
    iconWrap: "bg-[color:var(--status-draft-dot)]/15",
    icon: "text-[color:var(--status-draft-fg)]",
    Icon: FileText,
  },
  waiting: {
    strip: "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
    iconWrap: "bg-[color:var(--status-waiting-dot)]/15",
    icon: "text-[color:var(--status-waiting-fg)]",
    Icon: Hourglass,
  },
  failed: {
    strip: "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)]",
    iconWrap: "bg-[color:var(--status-failed-dot)]/15",
    icon: "text-[color:var(--status-failed-fg)]",
    Icon: CircleX,
  },
  done: {
    strip: "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)]",
    iconWrap: "bg-[color:var(--status-completed-dot)]/15",
    icon: "text-[color:var(--status-completed-fg)]",
    Icon: CheckCircle2,
  },
  rejected: {
    strip: "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)]",
    iconWrap: "bg-[color:var(--status-cancelled-dot)]/15",
    icon: "text-[color:var(--status-cancelled-fg)]",
    Icon: CircleX,
  },
} as const;

export function SplitRunDecisionFooterPreview({ kind }: { kind: DecisionFooterKind }) {
  const model = kind === "running" ? undefined : COPY[kind];
  return (
    <div
      className="flex min-h-[380px] w-full max-w-[560px] flex-col overflow-hidden rounded-lg border border-border bg-background shadow-sm"
      data-testid={`decision-footer-preview-${kind}`}
    >
      <header className="flex shrink-0 items-center gap-3 border-b border-border px-5 py-3">
        <h2 className="min-w-0 flex-1 truncate text-[16px] font-semibold tracking-[-0.02em]">
          {model?.title ?? "Add refund reconciliation test"}
        </h2>
        <button type="button" className="rounded-full p-1 hover:bg-muted" aria-label="Close">
          <XIcon className="size-4" />
        </button>
      </header>
      <div className="min-h-0 flex-1 px-5 py-4 text-[13px] text-muted-foreground">
        {kind === "running"
          ? "Implement is running. Stop lives on that automation. The header has no Reject or Approve."
          : kind === "statusNote"
            ? "A Set Work Order Status Note supplies the headline, body, and Review PR link. Reject and Approve stay on this strip."
            : "Automations log. Close actions stay in the footer note, not in the header."}
      </div>
      {model ? <DecisionNote copy={model} /> : null}
    </div>
  );
}

function DecisionNote({ copy }: { copy: DecisionCopy }) {
  const visual = TONE[copy.tone];
  const reopen = copy.actions.some((action) => action.id === "reopen");
  const Icon = reopen ? RotateCcw : visual.Icon;

  return (
    <div className={cn("shrink-0 border-t px-4 py-3", visual.strip)} data-testid="split-run-decision-note">
      <div className="flex items-start gap-3">
        <span
          className={cn("flex size-8 shrink-0 items-center justify-center rounded-full", visual.iconWrap)}
          aria-hidden
        >
          <Icon className={cn("size-4", visual.icon)} />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">{copy.headline}</h3>
          <p className="mt-1 text-[13px] text-foreground/80">{copy.text}</p>
          <div className="mt-3">
            <NoteActions copy={copy} />
          </div>
        </div>
      </div>
    </div>
  );
}

function NoteCta({ cta }: { cta: { label: string; href: string } }) {
  const external = cta.href.startsWith("http");
  return (
    <Button asChild type="button" size="sm" variant="outline">
      <a href={cta.href} {...(external ? { target: "_blank", rel: "noreferrer" } : {})}>
        {cta.label}
        {external ? <ExternalLink className="size-3.5" aria-hidden /> : null}
      </a>
    </Button>
  );
}

function NoteActions({ copy }: { copy: DecisionCopy }) {
  return (
    <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
      {copy.cta ? <NoteCta cta={copy.cta} /> : null}
      {copy.actions.map((action) => (
        <Button key={action.id} type="button" size="sm" variant={action.emphasis === "primary" ? "default" : "outline"}>
          {action.label}
        </Button>
      ))}
    </div>
  );
}
