import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CircleAlert, CircleCheck, ExternalLink, Loader2, Minus, TriangleAlert } from "lucide-react";
import { useState } from "react";

import {
  formatCheckScore,
  LEVEL_LABEL,
  type WorkOrderCheckLevel,
  type WorkOrderCheckPresentation,
} from "../../lib/workOrderChecks";
import { WorkOrderCheckDialog } from "../../WorkOrderCheckDialog";
import type { SplitRunFooter, SplitRunFooterAction, SplitRunFooterNote } from "./splitRunFooter";

const PILL_TONE: Record<WorkOrderCheckLevel, string> = {
  positive: "border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300",
  neutral: "border-slate-400/40 bg-slate-500/10 text-slate-700 dark:text-slate-300",
  caution: "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300",
  critical: "border-red-900/40 bg-red-700 text-white hover:bg-red-700/90",
};

const PILL_ICON = {
  positive: CircleCheck,
  neutral: Minus,
  caution: TriangleAlert,
  critical: CircleAlert,
} as const;

/**
 * Quiet two-row footer under the Log. A run note sits above an always-on
 * state bar. Check pills sit in the header next to owner and cost.
 */
export function SplitRunReview({
  footer,
  className,
  onStart,
  startBusy = false,
  startDisabled = false,
}: {
  footer: SplitRunFooter;
  className?: string;
  onStart?: () => void | Promise<void>;
  startBusy?: boolean;
  startDisabled?: boolean;
}) {
  const showBar = footer.sentence !== "" || footer.actions.length > 0;

  return (
    <aside
      className={cn("shrink-0 border-t border-border bg-background px-4 py-3 text-foreground", className)}
      role="complementary"
      aria-label="Work order status"
      data-testid="split-run-review"
    >
      {footer.note ? <FooterNote note={footer.note} /> : null}
      {showBar ? (
        <div className={cn("flex flex-wrap items-center justify-between gap-3", footer.note && "mt-3")}>
          <p className="min-w-0 text-[13px] leading-5 text-muted-foreground">{footer.sentence}</p>
          {footer.actions.length > 0 ? (
            <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
              {footer.actions.map((action) => (
                <FooterAction
                  key={action.id}
                  action={action}
                  onStart={onStart}
                  startBusy={startBusy}
                  startDisabled={startDisabled}
                />
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </aside>
  );
}

function FooterNote({ note }: { note: SplitRunFooterNote }) {
  return (
    <div className="min-w-0">
      <h3 className="text-[15px] font-semibold tracking-[-0.01em]">{note.headline}</h3>
      {note.text ? (
        <div className="mt-1 text-[13px] leading-5 text-muted-foreground [&_a]:underline">
          <MarkdownContent content={note.text} variant="workspace" />
        </div>
      ) : null}
      {note.sourceName ? <p className="mt-1.5 truncate text-[12px] text-muted-foreground">{note.sourceName}</p> : null}
    </div>
  );
}

function FooterAction({
  action,
  onStart,
  startBusy,
  startDisabled,
}: {
  action: SplitRunFooterAction;
  onStart?: () => void | Promise<void>;
  startBusy: boolean;
  startDisabled: boolean;
}) {
  const primary = action.emphasis === "primary";
  const variant = primary ? "default" : "outline";
  const testId = primary ? "split-run-review-cta" : `split-run-footer-${action.id}`;

  if (action.kind === "start") {
    return (
      <Button
        type="button"
        size="sm"
        variant={variant}
        disabled={startDisabled || startBusy}
        onClick={() => void onStart?.()}
        data-testid={testId}
      >
        {startBusy ? <Loader2 className="size-3.5 animate-spin" aria-hidden /> : null}
        {action.label}
      </Button>
    );
  }

  if (action.href) {
    const inApp = action.href.startsWith("/");
    return (
      <Button asChild size="sm" variant={variant} data-testid={testId}>
        {inApp ? (
          <Link href={action.href}>{action.label}</Link>
        ) : (
          <a href={action.href} target="_blank" rel="noreferrer">
            {action.label}
            <ExternalLink className="size-3.5" aria-hidden />
          </a>
        )}
      </Button>
    );
  }

  return (
    <Button type="button" size="sm" variant={variant} data-testid={testId}>
      {action.label}
    </Button>
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
