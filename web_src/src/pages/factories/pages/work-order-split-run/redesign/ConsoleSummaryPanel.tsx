import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import { overlayHeaderSpend } from "@/lib/overlayHeaderSpend";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { type ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { workOrderCardPullRequestIsMergeable } from "../../../lib/workOrderCardPullRequest";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { pullRequestLabel } from "../../../lib/workOrderPullRequest";
import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderMergeableChip, WorkOrderPullRequestChip } from "../../../workOrders/WorkOrderPullRequestChip";
import { useLiveHeaderSpendOverlay } from "../liveHeaderSpendContext";
import { ConsoleCheckRows } from "./consoleCheckRows";
import type { SplitRunFixture } from "../splitRunMocks";
import { splitRunLinkedArtifacts } from "../splitRunPopupModel";
import { isPullRequestReviewFooter, pullRequestReviewNote } from "../splitRunPullRequestReview";
import type { SplitRunSource } from "../splitRunSource";
import { WorkOrderSplitRunSource } from "../WorkOrderSplitRunSource";
import type { AutomationStage, outcomeSummary } from "./automationsViewModel";
import { StaticStatusGlyph } from "./redesignShared";

/**
 * Reads top to bottom by priority: the decision, then what the run
 * produced (pull request, checks, artifacts), then reference details.
 */
export function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
  pullRequests,
  panelReview,
  source,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
  pullRequests?: FactoriesFactoryPullRequest[];
  panelReview?: ReactNode;
  source?: SplitRunSource;
}) {
  const liveSpend = useLiveHeaderSpendOverlay();
  const spend = overlayHeaderSpend(outcome.spend, outcome.tokens, liveSpend);
  const reviewedHref = panelReview ? reviewStripPullRequestHref(fixture) : undefined;
  const panelPullRequests = withoutPullRequest(
    pullRequests?.length ? pullRequests : outcome.pullRequests,
    reviewedHref,
  );
  const hasPullRequest = panelPullRequests.length > 0 || Boolean(reviewedHref);
  const artifacts = splitRunLinkedArtifacts([
    ...new Map(stages.flatMap((stage) => stage.outputs.artifacts).map((artifact) => [artifact.id, artifact])).values(),
  ]).filter((artifact) => !(hasPullRequest && isBranchArtifact(artifact)));
  const checks = stages.flatMap((stage) => stage.checks);
  // Placeholder values such as "Waiting" mirror the status; only a real time reads as a duration.
  const duration = /\d/.test(outcome.duration) ? outcome.duration : undefined;
  const spendRows = (fixture.usageByModel ?? []).map((row) => ({
    label: row.model?.split("/").at(-1) ?? row.provider ?? "",
    value: `$${(Number(row.costCents ?? 0) / 100).toFixed(2)}`,
  }));
  return (
    <aside className="lg:sticky lg:top-0 lg:self-start" data-testid="redesign-console-summary">
      <Frame variant="default" spacing="sm" stacked className="[--frame-radius:var(--radius-lg)]">
        <FrameHeader>
          <FrameTitle className="flex items-center gap-2">
            <StaticStatusGlyph status={outcome.status} />
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
        {checks.length > 0 ? (
          <FramePanel className="flex flex-col gap-2 py-3">
            <span className="text-[12px] font-medium text-muted-foreground">Checks</span>
            <ConsoleCheckRows checks={checks} />
          </FramePanel>
        ) : null}
        {artifacts.length > 0 ? (
          <FramePanel className="flex flex-col gap-2 py-3" data-testid="redesign-console-artifacts">
            <span className="text-[12px] font-medium text-muted-foreground">Artifacts</span>
            {artifacts.map((artifact) => (
              <WorkOrderArtifactInline
                key={artifact.id}
                artifact={{ id: artifact.id, type: artifact.type ?? "", data: toArtifactDataRecord(artifact.data) }}
              />
            ))}
          </FramePanel>
        ) : null}
        <FramePanel className="flex flex-col gap-2 py-3" data-testid="redesign-console-context">
          <SummaryRow label="Owner">
            <OrgUserReference display={outcome.owner} size="xs" nameClassName="text-[13px]" />
          </SummaryRow>
          <PanelSource source={source} owner={outcome.owner.id} />
          <SummaryRow label="Started">{outcome.startedLabel.replace(/^Started\s+/i, "")}</SummaryRow>
          {duration ? <SummaryRow label="Duration">{duration}</SummaryRow> : null}
          <SummaryRow label="Spend">
            {spend.costUsd} <span className="text-muted-foreground">· {spend.tokensLabel}</span>
          </SummaryRow>
          {spendRows.map((row) => (
            <SummaryRow key={row.label} label={row.label} muted>
              {row.value}
            </SummaryRow>
          ))}
        </FramePanel>
      </Frame>
    </aside>
  );
}

/** The pull request the decision strip already shows. The panel lists it only once. */
function reviewStripPullRequestHref(fixture: SplitRunFixture): string | undefined {
  if (!fixture.footer.note || !isPullRequestReviewFooter(fixture.footer)) {
    return undefined;
  }
  return pullRequestReviewNote(fixture.footer.note)?.href;
}

function withoutPullRequest(
  pullRequests: FactoriesFactoryPullRequest[],
  href: string | undefined,
): FactoriesFactoryPullRequest[] {
  if (!href) {
    return pullRequests;
  }
  const normalized = href.replace(/\/$/, "");
  return pullRequests.filter((pullRequest) => (pullRequest.url ?? "").replace(/\/$/, "") !== normalized);
}

/** The pull request supersedes its branch, so a listed branch would repeat it. */
function isBranchArtifact(artifact: FactoriesWorkOrderArtifact): boolean {
  return (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase() === "branch";
}

/**
 * The source of the task. A task the owner created by hand keeps one
 * "Created manually" row instead of repeating the owner's name.
 */
function PanelSource({ source, owner }: { source?: SplitRunSource; owner: string }) {
  if (!source) {
    return null;
  }
  if (source.kind === "manual" && source.person.id === owner) {
    return <SummaryRow label="Source">{source.detail}</SummaryRow>;
  }
  return (
    <div>
      <span className="text-[12px] font-medium text-muted-foreground">Source</span>
      <WorkOrderSplitRunSource source={source} />
    </div>
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
