import { useEventExecutions } from "@/hooks/useCanvasData";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import { useLayoutEffect, useRef, useState } from "react";

import { getWorkOrderRunHref } from "../../../lib/workOrderExecutions";
import type { SplitRunPhase, SplitRunPhaseStatus } from "../splitRunMocks";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import type { AutomationStage } from "./automationsViewModel";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { formatClock, META_TEXT_CLASSNAME } from "./redesignFormat";
import { StaticStatusGlyph } from "./redesignShared";
import {
  waitForPullRequestChecksNode,
  watchedPullRequestChecksFromExecutions,
  type WatchedCheckStatus,
  type WatchedPullRequestCheck,
} from "./watchedPullRequestChecks";

const RUN_TITLE_MARKDOWN = "min-w-0 truncate text-left text-[12px] font-medium text-foreground [&_p]:m-0 [&_p]:inline";
const RUN_LINK = "font-medium text-current !underline !decoration-current underline-offset-2";

/**
 * Every run of this automation, oldest first. One flat list: runs are
 * divider rows, not nested boxes — the card border is the only frame.
 * A lone run skips the row chrome; its title stays when it says more
 * than the card title (each Address PR feedback run has its own title).
 */
export function AgentRunsPage({
  runs,
  phases,
  automationName,
  organizationId,
  factoryKey,
  orderNumber,
  usagePhaseId,
}: {
  runs: AutomationStage[];
  phases: SplitRunPhase[];
  automationName: string;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  /** Only this run reports into the card usage chart. */
  usagePhaseId?: string;
}) {
  const single = runs.length === 1 ? runs[0] : undefined;
  if (single) {
    return (
      <SingleRun
        run={single}
        phase={phases.find((phase) => phase.id === single.id)}
        automationName={automationName}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        reportUsage={!usagePhaseId || single.id === usagePhaseId}
      />
    );
  }
  return (
    <div className="flex flex-col divide-y divide-border">
      {runs.map((run, index) => (
        <AgentRunRow
          key={run.id}
          run={run}
          phase={phases.find((phase) => phase.id === run.id)}
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={orderNumber}
          defaultOpen={index === runs.length - 1}
          reportUsage={!usagePhaseId || run.id === usagePhaseId}
        />
      ))}
    </div>
  );
}

/** The only run: no row to open, just the title (when it adds one), then the log. */
function SingleRun({
  run,
  phase,
  automationName,
  organizationId,
  factoryKey,
  orderNumber,
  reportUsage,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  automationName: string;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  reportUsage: boolean;
}) {
  const title = plainRunTitle(run.name);
  return (
    <div className="space-y-2">
      {title && title !== automationName ? (
        <MarkdownContent
          content={run.name}
          variant="workspace"
          openLinksInNewTab
          linkClassName={RUN_LINK}
          className={RUN_TITLE_MARKDOWN}
        />
      ) : null}
      <RunDetails
        run={run}
        phase={phase}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        reportUsage={reportUsage}
      />
    </div>
  );
}

function AgentRunRow({
  run,
  phase,
  organizationId,
  factoryKey,
  orderNumber,
  defaultOpen,
  reportUsage,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  defaultOpen: boolean;
  reportUsage: boolean;
}) {
  const title = plainRunTitle(run.name);
  const clock = formatClock(run.startedAt);
  return (
    <Collapsible defaultOpen={defaultOpen} className="group py-1.5 first:pt-0 last:pb-0">
      <div className="relative flex items-center gap-1.5 py-1">
        <CollapsibleTrigger className="absolute inset-0 z-0 cursor-pointer" aria-label={`Toggle ${title}`} />
        <div className="pointer-events-none relative z-10 flex min-w-0 flex-1 items-center gap-1.5">
          <ChevronRight
            className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
          {clock ? <span className={cn(META_TEXT_CLASSNAME, "shrink-0 tabular-nums")}>{clock}</span> : null}
          <StaticStatusGlyph status={run.status} />
          <MarkdownContent
            content={run.name}
            variant="workspace"
            openLinksInNewTab
            linkClassName={RUN_LINK}
            className={cn(RUN_TITLE_MARKDOWN, "flex-1 [&_a]:pointer-events-auto")}
          />
        </div>
      </div>
      <CollapsibleContent className="space-y-3 py-2 pl-5">
        <RunDetails
          run={run}
          phase={phase}
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={orderNumber}
          reportUsage={reportUsage}
        />
      </CollapsibleContent>
    </Collapsible>
  );
}

function RunDetails({
  run,
  phase,
  organizationId,
  factoryKey,
  orderNumber,
  reportUsage,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  reportUsage: boolean;
}) {
  const checks = useWatchedPullRequestChecks(phase, organizationId);
  return (
    <>
      <WatchedPullRequestCheckList checks={checks} />
      {checksDescriptionIsRedundant(run.description, checks) ? null : <RunDescription description={run.description} />}
      <LiveAgentSteps
        stage={run}
        phase={phase}
        organizationId={organizationId}
        emptyNote="No steps for this run."
        reportUsage={reportUsage}
        runHref={runHrefFor(organizationId, factoryKey, phase, orderNumber)}
      />
    </>
  );
}

function useWatchedPullRequestChecks(
  phase: SplitRunPhase | undefined,
  organizationId?: string,
): WatchedPullRequestCheck[] {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const node = waitForPullRequestChecksNode(live.canvas?.nodes);
  const executions = useEventExecutions(phase?.appId ?? "", node ? (live.rootEventId ?? null) : null);
  if (!node || (executions.isError && !executions.data)) {
    return [];
  }
  return watchedPullRequestChecksFromExecutions(executions.data?.executions, node.id, node.checkNames);
}

const CHECK_GLYPH: Record<WatchedCheckStatus, SplitRunPhaseStatus> = {
  Pending: "waiting",
  Failed: "failed",
  Passed: "passed",
};

const CHECK_STATUS_CLASS: Record<WatchedCheckStatus, string> = {
  Pending: "text-[color:var(--status-waiting-fg)]",
  Failed: "text-[color:var(--status-failed-fg)]",
  Passed: "text-[color:var(--status-completed-fg)]",
};

function WatchedPullRequestCheckList({ checks }: { checks: WatchedPullRequestCheck[] }) {
  if (checks.length === 0) {
    return null;
  }
  return (
    <ul aria-label="Pull request checks" className="flex flex-col" data-testid="pull-request-checks">
      {checks.map((check, index) => (
        <li
          key={`${check.name}-${index}`}
          className={cn("flex min-w-0 gap-2 py-1", check.summary ? "items-start" : "items-center")}
        >
          <span className={cn("flex shrink-0", check.summary && "mt-0.5")}>
            <StaticStatusGlyph status={CHECK_GLYPH[check.status]} />
          </span>
          <div className="min-w-0 flex-1">
            <CheckName check={check} />
            {check.summary ? (
              <p className="mt-0.5 text-[12px] leading-5 text-muted-foreground">{check.summary}</p>
            ) : null}
          </div>
          <span
            className={cn(
              "shrink-0 text-[12px] font-medium",
              check.summary && "mt-0.5",
              CHECK_STATUS_CLASS[check.status],
            )}
          >
            {check.status}
          </span>
        </li>
      ))}
    </ul>
  );
}

function checksDescriptionIsRedundant(description: string | undefined, checks: WatchedPullRequestCheck[]): boolean {
  if (!isChecksMarkdownDescription(description) || checks.length === 0) {
    return false;
  }
  const rendered = new Set(checks.map(watchedCheckMarkdownLine));
  return checksMarkdownLines(description).every((line) => rendered.has(line));
}

function isChecksMarkdownDescription(description: string | undefined): boolean {
  const body = checksMarkdownLines(description);
  return body.length > 0 && body.every((line) => line.startsWith("· "));
}

function checksMarkdownLines(description: string | undefined): string[] {
  const lines = description
    ?.split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  if (!lines || lines.length === 0) {
    return [];
  }
  return lines[0] === "Failed checks" ? lines.slice(1) : lines;
}

function CheckName({ check }: { check: WatchedPullRequestCheck }) {
  const href = safeExternalUrl(check.detailsUrl);
  const label = watchedCheckLabel(check);
  const className = "block min-w-0 break-words text-[12.5px] font-medium text-foreground";
  if (!href) {
    return <span className={className}>{label}</span>;
  }
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={cn(className, RUN_LINK)}>
      {label}
    </a>
  );
}

function watchedCheckLabel(check: WatchedPullRequestCheck): string {
  return check.description ? `${check.name}: ${check.description}` : check.name;
}

function watchedCheckMarkdownLine(check: WatchedPullRequestCheck): string {
  const label = watchedCheckLabel(check);
  const item = check.detailsUrl ? `[${label}](${check.detailsUrl})` : label;
  return check.summary ? `· ${item}: ${check.summary}` : `· ${item}`;
}

/** Review comments stay at five lines until the person asks for the rest. */
function RunDescription({ description }: { description?: string }) {
  const [expanded, setExpanded] = useState(false);
  const [overflows, setOverflows] = useState(false);
  const textRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    if (expanded) {
      return;
    }
    const el = textRef.current;
    if (!el) {
      return;
    }
    const measure = () => setOverflows(el.scrollHeight - el.clientHeight > 1);
    measure();
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, [description, expanded]);

  if (!description?.trim()) {
    return null;
  }
  return (
    <div data-testid="redesign-run-description">
      <div
        ref={textRef}
        data-testid="redesign-run-description-body"
        className={cn("text-[12.5px] leading-5 text-muted-foreground", !expanded && "line-clamp-5")}
      >
        <MarkdownContent content={description} variant="workspace" />
      </div>
      {overflows ? (
        <button
          type="button"
          className="mt-1 text-[12px] font-medium text-muted-foreground hover:text-foreground"
          aria-expanded={expanded}
          onClick={(event) => {
            event.stopPropagation();
            setExpanded((open) => !open);
          }}
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      ) : null}
    </div>
  );
}

/** Markdown links reduced to their text, for aria labels. */
function plainRunTitle(name: string): string {
  return name.replace(/\[([^\]]*)\]\([^)]*\)/g, "$1").trim() || "run";
}

function runHrefFor(
  organizationId: string | undefined,
  factoryKey: string | undefined,
  phase: SplitRunPhase | undefined,
  orderNumber: string | undefined,
): string | null {
  if (!organizationId || !factoryKey) {
    return null;
  }
  return getWorkOrderRunHref(organizationId, factoryKey, phase?.appId, phase?.runId, { orderNumber });
}
