import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";
import { ChevronDown, ExternalLink } from "lucide-react";

import { formatCheckScore, workOrderCheckStatus, type WorkOrderCheckPresentation } from "./lib/workOrderChecks";
import { WorkOrderCheckAnalysis } from "./WorkOrderCheckDialog";

/**
 * One check as a details/summary row: score, label, and name, then a
 * single-line summary. Analysis opens below.
 */
export function WorkOrderCheckComment({
  check,
  runHref = null,
  defaultOpen = false,
}: {
  check: WorkOrderCheckPresentation;
  runHref?: string | null;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);

  return (
    <details
      className="border-t border-border py-4 open:pb-5"
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
      data-testid={`split-run-check-comment-${check.id}`}
    >
      <summary
        className="flex cursor-pointer list-none items-start justify-between gap-3 [&::-webkit-details-marker]:hidden"
        data-testid={`split-run-check-comment-toggle-${check.id}`}
      >
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span className="flex items-baseline gap-0.5 tabular-nums">
              <span className="text-[15px] font-semibold tracking-tight text-foreground">{value}</span>
              <span className="text-[12px] text-muted-foreground">{scale}</span>
            </span>
            <Badge variant="outline" className={cn("border", status.badgeClassName)}>
              {status.label}
            </Badge>
            <span className="text-[13px] text-muted-foreground">{check.name}</span>
          </div>
          {check.summary ? (
            <p className="mt-1 truncate text-[13px] leading-5 text-foreground">{check.summary}</p>
          ) : null}
        </div>
        <ChevronDown
          className={cn("mt-1 size-3.5 shrink-0 text-muted-foreground transition-transform", open && "rotate-180")}
          aria-hidden
        />
      </summary>

      <div className="mt-1">
        <WorkOrderCheckAnalysis check={check} />
        {runHref ? (
          <p className="mt-3">
            <Link
              href={runHref}
              className="inline-flex items-center gap-1 text-[12px] font-medium text-foreground/80 hover:text-foreground hover:underline"
            >
              View run
              <ExternalLink className="size-3 shrink-0" aria-hidden />
            </Link>
          </p>
        ) : null}
      </div>
    </details>
  );
}
