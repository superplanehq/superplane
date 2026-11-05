import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useId, useState } from "react";

import { MERGE_CONFIDENCE_SCORE_NAME } from "../../../lib/mergeConfidenceScore";
import {
  workOrderCheckDisplayName,
  workOrderCheckLevelLabel,
  type WorkOrderCheckPresentation,
} from "../../../lib/workOrderChecks";
import { CheckStatusBars } from "../../../workOrders/ConfidenceMeter";
import { CheckStatusIcon } from "../../../workOrders/CheckStatusIcon";
import { usePullRequestMergeChecks } from "./pullRequestChecksContext";

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
      {open ? <CheckList checks={checks} /> : null}
    </div>
  );
}

/**
 * Merge confidence of one pull request, under its title. It opens by itself
 * when a check needs review, so passing pull requests stay one line.
 */
export function PullRequestConfidence({
  repository,
  className,
}: {
  repository: string | null | undefined;
  className?: string;
}) {
  const checks = usePullRequestMergeChecks(repository);
  const [toggled, setToggled] = useState<boolean | null>(null);
  const summaryId = useId();
  if (checks.length === 0) {
    return null;
  }
  const attention = needsReview(checks);
  const open = toggled ?? attention;
  return (
    <div
      className={cn("min-w-0", className)}
      data-testid="redesign-console-pull-request-checks"
      data-repository={repository ?? ""}
    >
      <Button
        type="button"
        variant="ghost"
        aria-expanded={open}
        aria-label={MERGE_CONFIDENCE_SCORE_NAME}
        aria-describedby={summaryId}
        onClick={() => setToggled(!open)}
        className="h-auto w-full min-w-0 justify-start gap-2 rounded-md p-0 text-left font-normal hover:bg-transparent dark:hover:bg-transparent"
      >
        <CheckStatusBars checks={checks} className="shrink-0" />
        <span className="min-w-0 flex-1 truncate text-[12px] leading-4">
          <span className="font-medium text-foreground">{MERGE_CONFIDENCE_SCORE_NAME}</span>
          <span id={summaryId} className={attention ? "text-amber-700 dark:text-amber-400" : "text-muted-foreground"}>
            {" · "}
            {checksSummary(checks)}
          </span>
        </span>
        <ChevronDown
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform", open ? "" : "-rotate-90")}
          aria-hidden
        />
      </Button>
      {open ? <CheckRowList checks={checks} className="mt-1" /> : null}
    </div>
  );
}

/** Merge confidence scores each pull request repository on its own. Several repositories get one labeled list each. */
function CheckList({ checks }: { checks: WorkOrderCheckPresentation[] }) {
  const groups = checksByRepository(checks);
  if (groups.length <= 1) {
    return <CheckRowList checks={checks} className="mt-3 border-t border-border/70" />;
  }
  return (
    <div className="mt-3 flex flex-col gap-3 border-t border-border/70 pt-3">
      {groups.map((group) => (
        <section key={group.repository} aria-label={group.repository || "Task"}>
          <p className="truncate text-[12px] leading-4 text-muted-foreground">{group.repository || "Task"}</p>
          <CheckRowList checks={group.checks} />
        </section>
      ))}
    </div>
  );
}

function CheckRowList({ checks, className }: { checks: WorkOrderCheckPresentation[]; className?: string }) {
  return (
    <ul className={cn("flex flex-col", className)}>
      {checks.map((check) => (
        <li key={check.id} className="border-b border-border/60 last:border-b-0">
          <ConsoleCheckRow check={check} />
        </li>
      ))}
    </ul>
  );
}

function checksByRepository(
  checks: WorkOrderCheckPresentation[],
): Array<{ repository: string; checks: WorkOrderCheckPresentation[] }> {
  const groups = new Map<string, WorkOrderCheckPresentation[]>();
  for (const check of checks) {
    const repository = check.repository ?? "";
    groups.set(repository, [...(groups.get(repository) ?? []), check]);
  }
  return [...groups].map(([repository, grouped]) => ({ repository, checks: grouped }));
}

function needsReview(checks: WorkOrderCheckPresentation[]): boolean {
  return checks.some(checkNeedsReview);
}

function checkNeedsReview(check: WorkOrderCheckPresentation): boolean {
  return check.level === "critical" || check.level === "caution";
}

function checksSummary(checks: WorkOrderCheckPresentation[]): string {
  const attention = checks.filter(checkNeedsReview).length;
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
