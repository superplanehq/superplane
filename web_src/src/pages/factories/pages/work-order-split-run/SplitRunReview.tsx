import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CircleAlert, CircleCheck, ExternalLink, Hourglass, Minus, Play, TriangleAlert } from "lucide-react";
import { useState } from "react";

import {
  formatCheckScore,
  LEVEL_LABEL,
  type WorkOrderCheckLevel,
  type WorkOrderCheckPresentation,
} from "../../lib/workOrderChecks";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
import { WorkOrderCheckDialog } from "../../WorkOrderCheckDialog";
import type { SplitRunFooterTone } from "./splitRunMocks";

const PILL_TONE: Record<WorkOrderCheckLevel, string> = {
  positive: "border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300",
  neutral: "border-slate-400/40 bg-slate-500/10 text-slate-700 dark:text-slate-300",
  caution: "border-red-800/30 bg-red-700 text-white hover:bg-red-700/90",
  critical: "border-red-900/40 bg-red-700 text-white hover:bg-red-700/90",
};

const PILL_ICON = {
  positive: CircleCheck,
  neutral: Minus,
  caution: TriangleAlert,
  critical: CircleAlert,
} as const;

const FOOTER_TONE: Record<
  SplitRunFooterTone,
  { panel: string; label: string; iconWrap: string; icon: string; Icon: typeof Hourglass }
> = {
  waiting: {
    panel: "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
    label: "text-[color:var(--status-waiting-fg)]",
    iconWrap: "bg-[color:var(--status-waiting-dot)]/15",
    icon: "text-[color:var(--status-waiting-fg)]",
    Icon: Hourglass,
  },
  draft: {
    panel: "border-[color:var(--status-waiting-border)] bg-[color:var(--status-waiting-bg)]",
    label: "text-[color:var(--status-waiting-fg)]",
    iconWrap: "bg-[color:var(--status-waiting-dot)]/15",
    icon: "text-[color:var(--status-waiting-fg)]",
    Icon: Play,
  },
  failed: {
    panel: "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)]",
    label: "text-[color:var(--status-failed-fg)]",
    iconWrap: "bg-[color:var(--status-failed-dot)]/15",
    icon: "text-[color:var(--status-failed-fg)]",
    Icon: CircleAlert,
  },
};

/**
 * Sticky next-step footer at the bottom of the Log. Check pills sit in
 * the header next to owner and cost.
 */
export function SplitRunReview({
  notes,
  tone = "waiting",
  className,
}: {
  notes: WorkOrderStatusNotePresentation[];
  tone?: SplitRunFooterTone;
  className?: string;
}) {
  const note = notes[0];

  if (!note) {
    return null;
  }

  const chrome = FOOTER_TONE[tone];
  const Icon = chrome.Icon;

  return (
    <aside
      className={cn(
        "flex h-[11.5rem] shrink-0 flex-col justify-between border-t px-4 py-3 text-foreground",
        chrome.panel,
        className,
      )}
      role="complementary"
      aria-label="Next step"
      data-testid="split-run-review"
    >
      <div className="min-h-0">
        <p className={cn("text-[11px] font-semibold tracking-[0.06em] uppercase", chrome.label)}>Next step</p>
        <div className="mt-1.5 flex items-start gap-2.5">
          <span
            className={cn("flex size-8 shrink-0 items-center justify-center rounded-full", chrome.iconWrap)}
            aria-hidden
          >
            <Icon className={cn("size-4", chrome.icon)} />
          </span>
          <div className="min-w-0 flex-1">
            <h3 className="text-[15px] font-semibold tracking-[-0.01em]">{note.headline}</h3>
            {note.text ? (
              <div className="mt-1 line-clamp-3 text-[13px] leading-5 [&_a]:underline">
                <MarkdownContent content={note.text} variant="workspace" />
              </div>
            ) : null}
          </div>
        </div>
      </div>

      <div className="mt-3 flex items-center justify-between gap-3">
        {note.source?.name ? (
          <p className="truncate text-[12px] text-muted-foreground">{note.source.name}</p>
        ) : (
          <span />
        )}
        {note.cta ? (
          <Button asChild className={cn("shrink-0", tone === "failed" && "bg-red-700 text-white hover:bg-red-700/90")}>
            <a href={note.cta.href} target="_blank" rel="noreferrer">
              {note.cta.label}
              <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </Button>
        ) : null}
      </div>
    </aside>
  );
}

export function SplitRunCheckPills({
  checks,
  label = "Checks",
  testId = "split-run-checks",
}: {
  checks: WorkOrderCheckPresentation[];
  label?: string;
  testId?: string;
}) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <section className="inline-flex max-w-full" aria-label={label} data-testid={testId}>
      <ul className="flex flex-wrap gap-1.5">
        {checks.map((check) => (
          <li key={check.id}>
            <SplitRunCheckPill check={check} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function SplitRunCheckPill({ check }: { check: WorkOrderCheckPresentation }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { value, scale } = formatCheckScore(check);
  const level = LEVEL_LABEL[check.level];
  const Icon = PILL_ICON[check.level];
  const score = `${value}${scale}`;

  return (
    <>
      <Badge asChild variant="outline" className={cn("rounded-full px-2 py-0.5", PILL_TONE[check.level])}>
        <button
          type="button"
          onClick={() => setDialogOpen(true)}
          aria-label={`${check.name} ${score}. ${level.label}`}
          data-testid={`split-run-check-${check.id}`}
        >
          <Icon aria-hidden />
          <span>{check.name}</span>
          <span className="tabular-nums">{score}</span>
        </button>
      </Badge>
      <WorkOrderCheckDialog open={dialogOpen} onClose={() => setDialogOpen(false)} check={check} />
    </>
  );
}
