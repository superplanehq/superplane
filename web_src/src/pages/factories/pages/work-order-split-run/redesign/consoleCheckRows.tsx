import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

import { MERGE_CONFIDENCE_SCORE_NAME } from "../../../lib/mergeConfidenceScore";
import {
  workOrderCheckDisplayName,
  workOrderCheckLevelLabel,
  type WorkOrderCheckPresentation,
} from "../../../lib/workOrderChecks";
import { CheckStatusBars } from "../../../workOrders/ConfidenceMeter";
import { CheckStatusIcon } from "../../../workOrders/CheckStatusIcon";

/** Each header bar represents one check. Expanded rows show status and evidence. */
export function ConsoleCheckRows({
  checks,
  title = MERGE_CONFIDENCE_SCORE_NAME,
  testId = "redesign-console-checks",
  defaultOpen = false,
}: {
  checks: WorkOrderCheckPresentation[];
  title?: string;
  testId?: string;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  if (checks.length === 0) {
    return null;
  }
  return (
    <div data-testid={testId}>
      <button
        type="button"
        aria-expanded={open}
        aria-label={title}
        onClick={() => setOpen((current) => !current)}
        className="flex w-full items-center gap-2.5 text-left"
      >
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[14px] font-semibold leading-5 text-foreground">{title}</span>
          <span className="block text-[12px] leading-4 text-muted-foreground">{checksSummary(checks)}</span>
        </span>
        <CheckStatusBars checks={checks} testId={`${testId}-meter`} className="shrink-0" />
        <ChevronDown
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform", open ? "" : "-rotate-90")}
          aria-hidden
        />
      </button>
      {open ? (
        <ul className="mt-3 flex flex-col border-t border-border/70">
          {checks.map((check) => (
            <li key={check.id} className="border-b border-border/60 last:border-b-0">
              <ConsoleCheckRow check={check} />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function checksSummary(checks: WorkOrderCheckPresentation[]): string {
  const attention = checks.filter((check) => check.level === "critical" || check.level === "caution").length;
  if (attention > 0) {
    return attention === 1 ? "1 check needs review" : `${attention} checks need review`;
  }
  if (checks.some((check) => check.level === "neutral")) {
    return "Checks not fully assessed";
  }
  return "All checks passed";
}

function ConsoleCheckRow({ check }: { check: WorkOrderCheckPresentation }) {
  return (
    <div data-testid={`split-run-check-${check.id}`} className="flex min-w-0 gap-2.5 py-3">
      <CheckStatusIcon level={check.level} label={workOrderCheckLevelLabel(check.level)} className="mt-0.5" />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-medium text-foreground">{workOrderCheckDisplayName(check)}</p>
        {check.summary ? <p className="mt-0.5 text-[12px] leading-4 text-muted-foreground">{check.summary}</p> : null}
      </div>
    </div>
  );
}
