import { CircleCheck, CircleX, ExternalLink } from "lucide-react";

import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";

import {
  booleanCheckVerdict,
  formatCheckScore,
  LEVEL_LABEL,
  workOrderCheckStatus,
  type WorkOrderCheckPresentation,
} from "./lib/workOrderChecks";
import { WorkOrderCheckAttribution } from "./WorkOrderCheckAttribution";

/** Expanded view of one check: score, summary, and the full markdown analysis. */
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
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        size="large"
        className="max-h-[calc(100vh-2rem)] w-[calc(100%-2rem)] max-w-2xl overflow-y-auto text-left"
      >
        <DialogTitle>{check.name}</DialogTitle>
        <WorkOrderCheckBody check={check} />
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-[12px] text-muted-foreground">
          <span>
            <WorkOrderCheckAttribution check={check} />
          </span>
          {runHref ? <CheckRunLink href={runHref} /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

/** Score, summary, and analysis. Shared by the dialog and the description thread. */
export function WorkOrderCheckBody({ check }: { check: WorkOrderCheckPresentation }) {
  return (
    <>
      <WorkOrderCheckScoreRow check={check} />
      <WorkOrderCheckWriteup check={check} />
    </>
  );
}

export function WorkOrderCheckScoreRow({ check }: { check: WorkOrderCheckPresentation }) {
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);

  return (
    <div className="mt-3 flex items-center gap-3">
      <span className="flex items-center gap-2">
        <CheckVerdictIcon check={check} />
        <span className="flex items-baseline gap-0.5">
          <span className="text-3xl font-semibold tabular-nums tracking-tight text-foreground">{value}</span>
          <span className="text-sm text-muted-foreground">{scale}</span>
        </span>
      </span>
      <Badge variant="outline" className={cn("border", status.badgeClassName)}>
        {status.label}
      </Badge>
    </div>
  );
}

export function WorkOrderCheckWriteup({ check }: { check: WorkOrderCheckPresentation }) {
  return (
    <>
      <CheckTrendLine check={check} />
      {check.summary ? <p className="mt-3 text-sm text-foreground">{check.summary}</p> : null}
      <WorkOrderCheckAnalysis check={check} />
    </>
  );
}

export function WorkOrderCheckAnalysis({ check }: { check: WorkOrderCheckPresentation }) {
  if (!check.analysis?.trim()) {
    return null;
  }
  return (
    <div className="mt-4 border-t border-border pt-4">
      <MarkdownContent content={check.analysis} variant="workspace" />
    </div>
  );
}

function CheckRunLink({ href }: { href: string }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-1 font-medium text-foreground/80 hover:text-foreground hover:underline"
    >
      View run
      <ExternalLink className="size-3 shrink-0" aria-hidden />
    </Link>
  );
}

/** Level-colored pass/fail icon shown next to a boolean check's verdict. */
function CheckVerdictIcon({ check }: { check: WorkOrderCheckPresentation }) {
  if (check.format !== "boolean") {
    return null;
  }

  const Icon = check.score > 0 ? CircleCheck : CircleX;
  return <Icon className={cn("size-7 shrink-0", LEVEL_LABEL[check.level].className)} aria-hidden />;
}

/** Trend versus the previous report, spelled out in text. */
function CheckTrendLine({ check }: { check: WorkOrderCheckPresentation }) {
  if (check.previousScore === undefined || check.previousScore === check.score) {
    return null;
  }

  if (check.format === "boolean") {
    return (
      <p className="mt-2 text-[12px] text-muted-foreground">
        Was {booleanCheckVerdict(check.previousScore)} on the previous run.
      </p>
    );
  }

  const direction = check.score > check.previousScore ? "Up" : "Down";
  return (
    <p className="mt-2 text-[12px] text-muted-foreground">
      {direction} from <span className="tabular-nums">{check.previousScore}</span> on the previous run.
    </p>
  );
}
