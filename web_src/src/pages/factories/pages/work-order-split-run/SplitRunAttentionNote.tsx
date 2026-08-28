import { CheckCircle2, CircleX, ExternalLink, FileText, Hourglass, Loader2, RotateCcw } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

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
 * Sticky decision note. CTA and close actions sit under the copy, aligned
 * to the right. No Update manually, no source time.
 */
export function SplitRunAttentionNote({
  note,
  tone = "waiting",
  actions = [],
  runHref,
  actionBusy = false,
  startBusy = false,
  startDisabled = false,
  onAction,
}: {
  note: SplitRunFooterNote;
  tone?: SplitRunDecisionTone;
  actions?: SplitRunFooterAction[];
  runHref?: string | null;
  actionBusy?: boolean;
  startBusy?: boolean;
  startDisabled?: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const visual = TONE[tone];
  const Icon = actions.some((action) => action.kind === "reopen") ? RotateCcw : visual.Icon;

  return (
    <div className={cn("border-t px-4 py-3", visual.strip)} data-testid="split-run-attention-note">
      <div className="flex items-start gap-3">
        <span
          className={cn("flex size-8 shrink-0 items-center justify-center rounded-full", visual.iconWrap)}
          aria-hidden
        >
          <Icon className={cn("size-4", visual.icon)} />
        </span>

        <div className="min-w-0 flex-1">
          <h3 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">{note.headline}</h3>
          {note.text ? (
            <div className="mt-1 text-[13px] text-foreground/80">
              <MarkdownContent content={note.text} variant="workspace" />
            </div>
          ) : null}
          <NoteActionRow
            note={note}
            actions={actions}
            runHref={runHref}
            actionBusy={actionBusy}
            startBusy={startBusy}
            startDisabled={startDisabled}
            onAction={onAction}
          />
        </div>
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
  onAction,
}: {
  note: SplitRunFooterNote;
  actions: SplitRunFooterAction[];
  runHref?: string | null;
  actionBusy: boolean;
  startBusy: boolean;
  startDisabled: boolean;
  onAction?: (action: SplitRunFooterAction) => void;
}) {
  const href = note.cta?.href ?? runHref ?? undefined;
  const showCta = Boolean(note.cta && href);
  if (!showCta && actions.length === 0) {
    return null;
  }

  return (
    <div className="mt-3 flex flex-wrap items-center justify-end gap-2">
      {showCta && href && note.cta ? <NoteCta label={note.cta.label} href={href} /> : null}
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

function NoteCta({ label, href }: { label: string; href: string }) {
  const external = href.startsWith("http");
  return (
    <Button asChild size="sm" variant="outline">
      {external ? (
        <a href={href} target="_blank" rel="noreferrer">
          {label}
          <ExternalLink className="size-3.5" aria-hidden />
        </a>
      ) : (
        <Link href={href}>{label}</Link>
      )}
    </Button>
  );
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
      {busy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
      {action.label}
    </Button>
  );
}
