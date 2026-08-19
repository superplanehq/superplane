import { ExternalLink } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import {
  formatCheckScore,
  isBooleanCheck,
  LEVEL_LABEL,
  type ScoreCheckPresentation,
  type WorkOrderCheckPresentation,
} from "./lib/workOrderChecks";
import { WorkOrderCheckAttribution } from "./WorkOrderCheckAttribution";

/** Expanded view of one check: score (or pass/fail badge), summary, and the full markdown analysis. */
export function WorkOrderCheckDialog({
  open,
  onClose,
  check,
  runHref,
}: {
  open: boolean;
  onClose: () => void;
  check: WorkOrderCheckPresentation;
  runHref?: string | null;
}) {
  const level = LEVEL_LABEL[check.level];

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        size="large"
        className="max-h-[calc(100vh-2rem)] w-[calc(100%-2rem)] max-w-2xl overflow-y-auto text-left"
      >
        <DialogTitle>{check.name}</DialogTitle>

        {isBooleanCheck(check) ? (
          <div className="mt-3 flex items-center gap-3">
            <Badge variant="outline" className={cn("border text-[13px]", level.badgeClassName)}>
              {check.passed ? "Pass" : "Fail"}
            </Badge>
          </div>
        ) : (
          <ScoreDialogHeader check={check} badgeClassName={level.badgeClassName} label={level.label} />
        )}

        {check.summary ? <p className="mt-3 text-sm text-foreground">{check.summary}</p> : null}

        {check.analysis?.trim() ? (
          <div className="mt-4 border-t border-border pt-4">
            <MarkdownContent content={check.analysis} variant="workspace" />
          </div>
        ) : null}

        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-[12px] text-muted-foreground">
          <span>
            <WorkOrderCheckAttribution check={check} />
          </span>
          {runHref ? (
            <Link
              href={runHref}
              className="inline-flex items-center gap-1 font-medium text-foreground/80 hover:text-foreground hover:underline"
            >
              View run
              <ExternalLink className="size-3 shrink-0" aria-hidden />
            </Link>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/** Score, level badge, and trend line — scored checks only. */
function ScoreDialogHeader({
  check,
  badgeClassName,
  label,
}: {
  check: ScoreCheckPresentation;
  badgeClassName: string;
  label: string;
}) {
  const { value, scale } = formatCheckScore(check);
  return (
    <>
      <div className="mt-3 flex items-center gap-3">
        <span className="flex items-baseline gap-0.5">
          <span className="text-3xl font-semibold tabular-nums tracking-tight text-foreground">{value}</span>
          <span className="text-sm text-muted-foreground">{scale}</span>
        </span>
        <Badge variant="outline" className={cn("border", badgeClassName)}>
          {label}
        </Badge>
      </div>
      <CheckTrendLine check={check} />
    </>
  );
}

/** Trend versus the previous report, spelled out in text. */
function CheckTrendLine({ check }: { check: ScoreCheckPresentation }) {
  if (check.previousScore === undefined || check.previousScore === check.score) {
    return null;
  }

  const direction = check.score > check.previousScore ? "Up" : "Down";
  return (
    <p className="mt-2 text-[12px] text-muted-foreground">
      {direction} from <span className="tabular-nums">{check.previousScore}</span> on the previous run.
    </p>
  );
}
