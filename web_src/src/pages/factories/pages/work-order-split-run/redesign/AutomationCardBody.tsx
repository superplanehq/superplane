import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CircleStop, History, Maximize2, RotateCw } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { SplitRunPhase } from "../splitRunMocks";
import type { ConsoleAutomation } from "./automationsViewModel";
import { runFooterLine, showDescriptionInBody } from "./consoleCardText";
import { ArtifactChip, CheckBadgeRow } from "./consoleOutputChips";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

export function AutomationCardBody({
  automation,
  phase,
  organizationId,
  taskDescription,
  runHref,
  onStop,
  onRetry,
  actionBusy,
  onOpen,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  taskDescription?: string;
  runHref?: string;
  onStop?: () => void;
  onRetry?: () => void;
  actionBusy: boolean;
  onOpen: () => void;
}) {
  const { latest, runs } = automation;
  const pullRequest = latest.outputs.pullRequests[0];
  const hasOutputs = Boolean(pullRequest) || latest.outputs.artifacts.length > 0;
  const showDescription = showDescriptionInBody(latest);
  const creationDescription = latest.id === "backlog" ? taskDescription?.trim() : undefined;
  return (
    <div className="space-y-3">
      {showDescription ? (
        <div className="text-[12.5px] leading-5 text-muted-foreground">
          <MarkdownContent content={latest.description ?? ""} variant="workspace" />
        </div>
      ) : null}
      {creationDescription ? <ClampedMarkdown content={creationDescription} /> : null}
      <LiveAgentSteps stage={latest} phase={phase} organizationId={organizationId} />
      {hasOutputs || latest.checks.length > 0 ? (
        <div className="flex flex-col gap-2" data-testid={`redesign-console-outputs-${automation.id}`}>
          {pullRequest ? (
            <WorkOrderPullRequestInline pullRequest={pullRequest} showTitle className="text-[12px]" />
          ) : null}
          {latest.outputs.artifacts.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              {latest.outputs.artifacts.map((artifact) => (
                <ArtifactChip key={artifact.id} artifact={artifact} />
              ))}
            </div>
          ) : null}
          {latest.checks.length > 0 ? (
            <div className="flex flex-wrap items-center gap-1.5">
              {latest.checks.map((check) => (
                <CheckBadgeRow key={check.id} check={check} />
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3">
        <span className={cn(META_TEXT_CLASSNAME, "min-w-0")}>{runFooterLine(latest)}</span>
        <div className="ms-auto flex shrink-0 items-center gap-1.5">
          {runs.length > 1 ? (
            <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" onClick={onOpen}>
              <History className="size-3.5" aria-hidden />
              View {runs.length} runs
            </Button>
          ) : runHref ? (
            <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" asChild>
              <Link href={runHref}>
                <Maximize2 className="size-3.5" aria-hidden />
                View run
              </Link>
            </Button>
          ) : latest.appId ? (
            <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" onClick={onOpen}>
              Open run
            </Button>
          ) : null}
          {onRetry ? (
            <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onRetry}>
              <RotateCw className="size-3.5" aria-hidden />
              Retry
            </Button>
          ) : null}
          {onStop ? (
            <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onStop}>
              <CircleStop className="size-3.5" aria-hidden />
              Stop
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

const DESCRIPTION_CLAMP_PX = 320;

/** The task description on the creation card. Long text clamps with Show more. */
function ClampedMarkdown({ content }: { content: string }) {
  const [expanded, setExpanded] = useState(false);
  const [clamped, setClamped] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    setClamped((bodyRef.current?.scrollHeight ?? 0) > DESCRIPTION_CLAMP_PX + 40);
  }, [content]);
  return (
    <div data-testid="redesign-console-task-description">
      <div
        ref={bodyRef}
        className={cn("relative overflow-hidden text-[13px] leading-6", !expanded && clamped && "max-h-80")}
      >
        <MarkdownContent content={content} variant="workspace" />
        {!expanded && clamped ? (
          <div
            className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-background to-transparent"
            aria-hidden
          />
        ) : null}
      </div>
      {clamped ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="mt-1 h-7 px-2 text-[12px] text-muted-foreground"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Show less" : "Show more"}
        </Button>
      ) : null}
    </div>
  );
}
