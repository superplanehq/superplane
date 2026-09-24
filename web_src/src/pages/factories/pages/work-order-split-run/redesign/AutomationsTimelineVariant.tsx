import {
  Timeline,
  TimelineContent,
  TimelineDate,
  TimelineHeader,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from "@/components/reui/timeline";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight, GitPullRequest, ScrollText } from "lucide-react";
import { useState } from "react";

import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { SplitRunFixture } from "../splitRunMocks";
import { AgentStepList } from "./AgentStepList";
import {
  outcomeSummary,
  stagesFromFixture,
  type AutomationStage,
  type AutomationStageGroups,
  type PlumbingNode,
} from "./automationsViewModel";
import { OutcomeStrip } from "./OutcomeStrip";
import { RawLogSheet } from "./RawLogSheet";
import { META_TEXT_CLASSNAME, formatClock } from "./redesignFormat";
import { NodeIcon, StageStatusGlyph } from "./redesignShared";
import { StageOutputs } from "./StageOutputs";

/**
 * Variant A: stage timeline. A vertical ReUI Timeline is the spine, one
 * item per stage. Collapsed cards show outputs; open cards show the agent
 * transcript or the plumbing nodes. PR feedback runs nest under the PR.
 */
export function AutomationsTimelineVariant({
  fixture,
  initialOpenStageId = "implement",
}: {
  fixture: SplitRunFixture;
  initialOpenStageId?: string | null;
}) {
  const outcome = outcomeSummary(fixture);
  const groups = stagesFromFixture(fixture);
  const activeStep = activeStepIndex(groups);
  const [logStage, setLogStage] = useState<AutomationStage | null>(null);

  return (
    <div className="flex flex-col gap-5" data-testid="redesign-timeline-variant">
      <OutcomeStrip outcome={outcome} />
      <Timeline defaultValue={activeStep} className="pl-1">
        {groups.taskStages.map((stage, index) => (
          <StageTimelineItem
            key={stage.id}
            stage={stage}
            step={index + 1}
            defaultOpen={stage.id === initialOpenStageId}
            onViewLog={() => setLogStage(stage)}
          />
        ))}
      </Timeline>
      {groups.pullRequestGroups.map((group) => (
        <section key={group.id} className="flex flex-col gap-3" data-testid="redesign-timeline-pr-group">
          <header className="flex items-center gap-2 border-t border-border/70 pt-4">
            <GitPullRequest className="size-4 text-muted-foreground" aria-hidden />
            <span className="text-[13px] font-semibold text-foreground">Pull request activity</span>
            {group.pullRequest ? <WorkOrderPullRequestInline pullRequest={group.pullRequest} /> : null}
          </header>
          <Timeline defaultValue={group.stages.length} className="pl-1">
            {group.stages.map((stage, index) => (
              <StageTimelineItem
                key={stage.id}
                stage={stage}
                step={index + 1}
                defaultOpen={false}
                onViewLog={() => setLogStage(stage)}
              />
            ))}
          </Timeline>
        </section>
      ))}
      <RawLogSheet stage={logStage} open={logStage !== null} onOpenChange={(open) => !open && setLogStage(null)} />
    </div>
  );
}

function activeStepIndex(groups: AutomationStageGroups): number {
  const running = groups.taskStages.findIndex((stage) => stage.status !== "passed");
  return running === -1 ? groups.taskStages.length : running;
}

function StageTimelineItem({
  stage,
  step,
  defaultOpen,
  onViewLog,
}: {
  stage: AutomationStage;
  step: number;
  defaultOpen: boolean;
  onViewLog: () => void;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const hasDetail = stage.agentSteps.length > 0 || stage.plumbing.length > 0;
  return (
    <TimelineItem step={step} className="group-data-[orientation=vertical]/timeline:not-last:pb-5">
      <TimelineHeader>
        <TimelineSeparator className="bg-border group-data-completed/timeline-item:bg-border" />
        <TimelineIndicator className="flex size-5 items-center justify-center border-0 bg-background">
          <StageStatusGlyph status={stage.status} className="size-4" />
        </TimelineIndicator>
        <TimelineDate className="mb-0.5">{formatClock(stage.startedAt)}</TimelineDate>
        <TimelineTitle className="flex min-w-0 flex-wrap items-baseline gap-x-2 text-[14px]">
          <span>{stage.name}</span>
          <span className={cn(META_TEXT_CLASSNAME, "font-normal")}>
            {[stage.componentName, stage.duration, stage.model, stage.cost].filter(Boolean).join(" · ")}
          </span>
        </TimelineTitle>
      </TimelineHeader>
      <TimelineContent className="mt-2">
        <Collapsible open={open} onOpenChange={setOpen}>
          <div className="rounded-lg border border-border/80 bg-card" data-testid={`redesign-stage-card-${stage.id}`}>
            <div className="flex min-w-0 items-start gap-2 px-3 py-2.5">
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                {stage.description ? <MarkdownContent content={stage.description} variant="workspace" /> : null}
                <StageOutputs
                  pullRequests={stage.outputs.pullRequests}
                  artifacts={stage.outputs.artifacts}
                  checks={stage.checks}
                  testId={`redesign-stage-outputs-${stage.id}`}
                />
                {!stage.description &&
                stage.outputs.pullRequests.length === 0 &&
                stage.outputs.artifacts.length === 0 &&
                stage.checks.length === 0 ? (
                  <p className="text-[13px] text-muted-foreground">{stage.statusLabel}. No outputs.</p>
                ) : null}
              </div>
              {hasDetail ? (
                <CollapsibleTrigger asChild>
                  <button
                    type="button"
                    className="inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-1 text-[12px] font-medium text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                    aria-expanded={open}
                    data-testid={`redesign-stage-toggle-${stage.id}`}
                  >
                    {stage.agentSteps.length > 0
                      ? countLabel(stage.agentSteps.length, "step")
                      : countLabel(stage.plumbing.length, "node")}
                    <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-90")} aria-hidden />
                  </button>
                </CollapsibleTrigger>
              ) : null}
            </div>
            <CollapsibleContent>
              <div className="flex flex-col gap-3 border-t border-border/70 px-3 py-3">
                <PlumbingList nodes={stage.plumbing} />
                {stage.agentSteps.length > 0 ? (
                  <>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-[12px] font-medium text-muted-foreground">Agent transcript</span>
                      <button
                        type="button"
                        onClick={onViewLog}
                        className="inline-flex h-6 items-center gap-1 rounded px-1.5 text-[12px] text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                        data-testid={`redesign-stage-view-log-${stage.id}`}
                      >
                        <ScrollText className="size-3.5" aria-hidden />
                        View raw log
                      </button>
                    </div>
                    <AgentStepList stage={stage} view="detailed" showToggle={false} />
                  </>
                ) : null}
              </div>
            </CollapsibleContent>
          </div>
        </Collapsible>
      </TimelineContent>
    </TimelineItem>
  );
}

function countLabel(count: number, noun: string): string {
  return count === 1 ? `1 ${noun}` : `${count} ${noun}s`;
}

function PlumbingList({ nodes }: { nodes: PlumbingNode[] }) {
  if (nodes.length === 0) {
    return null;
  }
  return (
    <ol className="flex flex-wrap items-center gap-1.5" aria-label="Automation nodes">
      {nodes.map((node, index) => (
        <li key={node.id} className="flex items-center gap-1.5">
          {index > 0 ? <span className="text-muted-foreground/60">›</span> : null}
          <span className="inline-flex h-6 items-center gap-1.5 rounded-md border border-border/70 bg-muted/30 px-2 text-[12px] text-foreground/90">
            <NodeIcon iconSlug={node.iconSlug} className="size-3" />
            {node.name}
            {node.duration ? <span className={META_TEXT_CLASSNAME}>{node.duration}</span> : null}
          </span>
        </li>
      ))}
    </ol>
  );
}
