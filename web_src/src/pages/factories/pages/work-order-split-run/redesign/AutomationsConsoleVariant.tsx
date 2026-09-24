import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CircleX, ExternalLink, RotateCw, ScrollText } from "lucide-react";
import { useState, type ReactNode } from "react";

import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { SplitRunCheckPills } from "../SplitRunReview";
import type { SplitRunFixture } from "../splitRunMocks";
import { AgentStepList } from "./AgentStepList";
import { RawLogSheet } from "./RawLogSheet";
import { allStages, outcomeSummary, stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import { META_TEXT_CLASSNAME, formatClock } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

/**
 * Variant B: run console. Left: the step trace, stage by stage, with the
 * same detailed transcript as the stage timeline. Right: a sticky Frame
 * that holds status, spend, outputs, checks, and the action row.
 */
export function AutomationsConsoleVariant({ fixture }: { fixture: SplitRunFixture }) {
  const outcome = outcomeSummary(fixture);
  const groups = stagesFromFixture(fixture);
  const stages = allStages(groups);
  const [logStage, setLogStage] = useState<AutomationStage | null>(null);

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_300px]" data-testid="redesign-console-variant">
      <section className="flex min-w-0 flex-col gap-4">
        <h3 className="text-[13px] font-semibold text-foreground">Step trace</h3>
        <ol className="flex flex-col gap-3">
          {groups.taskStages.map((stage) => (
            <ConsoleStageSection key={stage.id} stage={stage} onViewLog={() => setLogStage(stage)} />
          ))}
          {groups.pullRequestGroups.flatMap((group) =>
            group.stages.map((stage) => (
              <ConsoleStageSection key={stage.id} stage={stage} onViewLog={() => setLogStage(stage)} pullRequestRun />
            )),
          )}
        </ol>
      </section>
      <ConsoleSummaryPanel fixture={fixture} outcome={outcome} stages={stages} />
      <RawLogSheet stage={logStage} open={logStage !== null} onOpenChange={(open) => !open && setLogStage(null)} />
    </div>
  );
}

function ConsoleStageSection({
  stage,
  onViewLog,
  pullRequestRun = false,
}: {
  stage: AutomationStage;
  onViewLog: () => void;
  pullRequestRun?: boolean;
}) {
  const failed = stage.status === "failed" || stage.status === "cancelled";
  return (
    <li className="rounded-lg border border-border/80 bg-card" data-testid={`redesign-console-stage-${stage.id}`}>
      <header className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 border-b border-border/70 px-3 py-2">
        <StageStatusGlyph status={stage.status} />
        <span className="text-[13px] font-semibold text-foreground">{stage.name}</span>
        {pullRequestRun && stage.pullRequestActivity?.pullRequest ? (
          <WorkOrderPullRequestInline pullRequest={stage.pullRequestActivity.pullRequest} className="text-[12px]" />
        ) : null}
        <span className={cn(META_TEXT_CLASSNAME, "ml-auto")}>
          {[formatClock(stage.startedAt), stage.duration, stage.model].filter(Boolean).join(" · ")}
        </span>
      </header>
      <div className="flex flex-col gap-2 px-3 py-2.5">
        {stage.description ? (
          <div className="text-[12.5px] text-muted-foreground">
            <MarkdownContent content={stage.description} variant="workspace" />
          </div>
        ) : null}
        {stage.agentSteps.length > 0 ? (
          <>
            <div className="flex items-center justify-between gap-3">
              <span className="text-[12px] font-medium text-muted-foreground">Agent transcript</span>
              <button
                type="button"
                onClick={onViewLog}
                className="inline-flex h-6 items-center gap-1 rounded px-1.5 text-[12px] text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                data-testid={`redesign-console-view-log-${stage.id}`}
              >
                <ScrollText className="size-3.5" aria-hidden />
                View raw log
              </button>
            </div>
            <AgentStepList stage={stage} view="detailed" showToggle={false} />
          </>
        ) : null}
        {failed ? (
          <Alert variant="destructive">
            <CircleX />
            <AlertTitle>{stage.name} did not finish</AlertTitle>
            <AlertDescription>Fix the error, then run this step again.</AlertDescription>
            <AlertAction>
              <Button size="sm" variant="outline">
                <RotateCw className="size-3.5" aria-hidden />
                Retry
              </Button>
            </AlertAction>
          </Alert>
        ) : null}
      </div>
    </li>
  );
}

function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
}) {
  const artifacts = [
    ...new Map(stages.flatMap((stage) => stage.outputs.artifacts).map((artifact) => [artifact.id, artifact])).values(),
  ];
  const checks = stages.flatMap((stage) => stage.checks);
  const spendRows = (fixture.usageByModel ?? []).map((row) => ({
    label: row.model?.split("/").at(-1) ?? row.provider ?? "",
    value: `$${(Number(row.costCents ?? 0) / 100).toFixed(2)}`,
  }));
  return (
    <aside className="lg:sticky lg:top-0 lg:self-start" data-testid="redesign-console-summary">
      <Frame variant="default" spacing="sm" stacked className="[--frame-radius:var(--radius-lg)]">
        <FrameHeader>
          <FrameTitle className="flex items-center gap-2">
            <StageStatusGlyph status={outcome.status} />
            {outcome.statusLabel}
          </FrameTitle>
          <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>
        </FrameHeader>
        <FramePanel className="flex flex-col gap-2 py-3">
          <SummaryRow label="Owner">
            <OrgUserReference display={outcome.owner} size="xs" nameClassName="text-[13px]" />
          </SummaryRow>
          <SummaryRow label="Started">{outcome.startedLabel.replace(/^Started\s+/i, "")}</SummaryRow>
          <SummaryRow label="Duration">{outcome.duration}</SummaryRow>
          <SummaryRow label="Spend">
            {outcome.spend} <span className="text-muted-foreground">· {outcome.tokens}</span>
          </SummaryRow>
          {spendRows.map((row) => (
            <SummaryRow key={row.label} label={row.label} muted>
              {row.value}
            </SummaryRow>
          ))}
        </FramePanel>
        <FramePanel className="flex flex-col gap-2 py-3">
          <span className="text-[12px] font-medium text-muted-foreground">Outputs</span>
          {outcome.pullRequests.map((pullRequest) => (
            <WorkOrderPullRequestInline key={pullRequest.id} pullRequest={pullRequest} showTitle />
          ))}
          {artifacts.map((artifact) => (
            <WorkOrderArtifactInline
              key={artifact.id}
              artifact={{ id: artifact.id, type: artifact.type ?? "", data: toArtifactDataRecord(artifact.data) }}
            />
          ))}
          {checks.length > 0 ? (
            <>
              <span className="mt-1 text-[12px] font-medium text-muted-foreground">Checks</span>
              <SplitRunCheckPills checks={checks} testId="redesign-console-checks" />
            </>
          ) : null}
        </FramePanel>
        <FramePanel className="flex flex-wrap gap-2 py-3">
          <Button size="sm" variant="outline">
            <ExternalLink className="size-3.5" aria-hidden />
            View run
          </Button>
          {fixture.footer.actions.map((action) => (
            <Button key={action.id} size="sm" variant={action.emphasis === "primary" ? "default" : "outline"}>
              {action.label}
            </Button>
          ))}
        </FramePanel>
      </Frame>
    </aside>
  );
}

function SummaryRow({ label, children, muted = false }: { label: string; children: ReactNode; muted?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3 text-[13px]">
      <span className={cn("shrink-0 text-muted-foreground", muted && "pl-3 text-[12px]")}>{label}</span>
      <span className={cn("min-w-0 truncate text-right text-foreground tabular-nums", muted && "text-[12px]")}>
        {children}
      </span>
    </div>
  );
}
