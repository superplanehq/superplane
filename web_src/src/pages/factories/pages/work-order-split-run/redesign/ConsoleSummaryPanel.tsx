import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { type ReactNode } from "react";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import { workOrderCardPullRequestIsMergeable } from "../../../lib/workOrderCardPullRequest";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { pullRequestLabel } from "../../../lib/workOrderPullRequest";
import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderMergeableChip, WorkOrderPullRequestChip } from "../../../workOrders/WorkOrderPullRequestChip";
import { SplitRunCheckPills } from "../SplitRunReview";
import type { SplitRunFixture } from "../splitRunMocks";
import { splitRunLinkedArtifacts } from "../splitRunPopupModel";
import type { AutomationStage, outcomeSummary } from "./automationsViewModel";
import { StageStatusGlyph } from "./redesignShared";

export function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
  pullRequests,
  panelReview,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
  pullRequests?: FactoriesFactoryPullRequest[];
  panelReview?: ReactNode;
}) {
  const artifacts = splitRunLinkedArtifacts([
    ...new Map(stages.flatMap((stage) => stage.outputs.artifacts).map((artifact) => [artifact.id, artifact])).values(),
  ]);
  const checks = stages.flatMap((stage) => stage.checks);
  const panelPullRequests = pullRequests?.length ? pullRequests : outcome.pullRequests;
  const hasOutputs = artifacts.length > 0 || checks.length > 0;
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
          {panelReview ? null : <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>}
        </FrameHeader>
        {panelReview ? <FramePanel className="py-3">{panelReview}</FramePanel> : null}
        {panelPullRequests.length > 0 ? (
          <FramePanel className="flex flex-col gap-2.5 py-3" data-testid="redesign-console-pull-requests">
            <span className="text-[12px] font-medium text-muted-foreground">
              {panelPullRequests.length === 1 ? "Pull request" : "Pull requests"}
            </span>
            {panelPullRequests.map((pullRequest, index) => (
              <PanelPullRequest key={pullRequest.id ?? pullRequest.url ?? index} pullRequest={pullRequest} />
            ))}
          </FramePanel>
        ) : null}
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
        {hasOutputs ? (
          <FramePanel className="flex flex-col gap-2 py-3">
            <span className="text-[12px] font-medium text-muted-foreground">Outputs</span>
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
        ) : null}
      </Frame>
    </aside>
  );
}

/**
 * One pull request, first-class: the title links to the provider, the
 * chips below carry state and mergeability, as on the board card.
 */
function PanelPullRequest({ pullRequest }: { pullRequest: FactoriesFactoryPullRequest }) {
  const href = safeExternalUrl(pullRequest.url);
  const title = pullRequest.title?.trim() || pullRequestLabel(pullRequest);
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      {href ? (
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="line-clamp-2 min-w-0 text-[13px] font-medium leading-5 text-foreground hover:underline"
        >
          {title}
        </a>
      ) : (
        <span className="line-clamp-2 min-w-0 text-[13px] font-medium leading-5 text-foreground">{title}</span>
      )}
      <div className="flex flex-wrap items-center gap-1.5">
        <WorkOrderPullRequestChip pullRequest={pullRequest} />
        {workOrderCardPullRequestIsMergeable(pullRequest) ? <WorkOrderMergeableChip /> : null}
      </div>
    </div>
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
