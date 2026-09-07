import type { ReactNode } from "react";
import { Bug, CheckCircle2, CircleX, ExternalLink, FileText, Hourglass, Loader2, RotateCcw, Undo2 } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { WorkOrderPersonMention } from "@/pages/app/markdownMentions";

import type { SplitRunDecisionTone, SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";

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

/**
 * Sticky decision note. Actions sit beside the copy. No Update manually,
 * no source time.
 */
function StoppedHeadline({ note }: { note: SplitRunFooterNote }) {
  if (!note.actor) {
    return note.headline;
  }
  return (
    <span className="inline-flex flex-wrap items-center gap-x-1">
      <WorkOrderPersonMention person={note.actor} />
      <span>{note.headline}</span>
    </span>
  );
}

export function SplitRunAttentionNote({
  note,
  tone = "waiting",
  actions = [],
  runHref,
  actionBusy = false,
  startBusy = false,
  startDisabled = false,
  modelSelect,
  onAction,
}: {
  note: SplitRunFooterNote;
  tone?: SplitRunDecisionTone;
  actions?: SplitRunFooterAction[];
  runHref?: string | null;
  actionBusy?: boolean;
  startBusy?: boolean;
  startDisabled?: boolean;
  modelSelect?: ReactNode;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const visual = TONE[tone];
  const Icon = actions.some((action) => action.kind === "reopen") ? RotateCcw : visual.Icon;

  return (
    <div className={cn("border-t px-5 py-4", visual.strip)} data-testid="split-run-attention-note">
      <div className="flex items-center gap-3.5">
        <span
          className={cn("flex size-10 shrink-0 items-center justify-center rounded-full", visual.iconWrap)}
          aria-hidden
        >
          <Icon className={cn("size-5", visual.icon)} />
        </span>

        <div className="min-w-0 flex-1">
          <h3 className="workspace-section-title">
            <StoppedHeadline note={note} />
          </h3>
          {note.text ? (
            <div className="mt-1.5">
              <MarkdownContent
                content={note.text}
                variant="workspace"
                className="max-w-none text-[14px] leading-relaxed text-foreground/80"
              />
            </div>
          ) : null}
        </div>
        <NoteActionRow
          note={note}
          actions={actions}
          runHref={runHref}
          actionBusy={actionBusy}
          startBusy={startBusy}
          startDisabled={startDisabled}
          modelSelect={modelSelect}
          onAction={onAction}
        />
      </div>
    </div>
  );
}

function NoteActionRow({
  note,
  actions,
  runHref,
  actionBusy,
  startBusy,
  startDisabled,
  modelSelect,
  onAction,
}: {
  note: SplitRunFooterNote;
  actions: SplitRunFooterAction[];
  runHref?: string | null;
  actionBusy: boolean;
  startBusy: boolean;
  startDisabled: boolean;
  modelSelect?: ReactNode;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const href = note.cta?.href ?? runHref ?? undefined;
  const showCta = Boolean(note.cta && href);
  if (!showCta && actions.length === 0 && !modelSelect) {
    return null;
  }

  return (
    <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
      {modelSelect}
      {showCta && href && note.cta ? <NoteCta label={note.cta.label} href={href} icon={note.cta.icon} /> : null}
      {actions.map((action) => (
        <NoteAction
          key={action.id}
          action={action}
          actionBusy={actionBusy}
          startBusy={startBusy}
          startDisabled={startDisabled}
          onClick={() => onAction?.(action)}
        />
      ))}
    </div>
  );
}

function NoteCta({ label, href, icon }: { label: string; href: string; icon?: "bug" }) {
  const external = href.startsWith("http");
  const mark = icon === "bug" ? <Bug className="size-3.5" aria-hidden /> : null;
  return (
    <Button asChild size="sm" variant="outline">
      {external ? (
        <a href={href} target="_blank" rel="noreferrer">
          {mark}
          {label}
          <ExternalLink className="size-3.5" aria-hidden />
        </a>
      ) : (
        <Link href={href}>
          {mark}
          {label}
        </Link>
      )}
    </Button>
  );
}

function ActionIcon({ icon }: { icon?: SplitRunFooterAction["icon"] }) {
  if (icon === "undo-2") {
    return <Undo2 className="size-3.5" aria-hidden />;
  }
  return null;
}

function NoteAction({
  action,
  actionBusy,
  startBusy,
  startDisabled,
  onClick,
}: {
  action: SplitRunFooterAction;
  actionBusy: boolean;
  startBusy: boolean;
  startDisabled: boolean;
  onClick: () => void;
}) {
  const primary = action.emphasis === "primary";
  const busy = action.kind === "start" ? startBusy : actionBusy;
  const disabled = action.kind === "start" ? startDisabled || startBusy : actionBusy;

  return (
    <Button
      type="button"
      size="sm"
      variant={primary ? "default" : "outline"}
      disabled={disabled}
      onClick={onClick}
      data-testid={primary ? "split-run-review-cta" : `split-run-footer-${action.id}`}
    >
      {busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : <ActionIcon icon={action.icon} />}
      {action.label}
    </Button>
  );
}
