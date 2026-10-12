import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { overlayHeaderSpend } from "@/lib/overlayHeaderSpend";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { type ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { CONFIDENCE_CHECK_KEY, CONFIDENCE_CHECK_NAME, isScoreCheckName } from "../../../lib/confidenceScore";
import { consoleCheckList, isMergeConfidenceMetric } from "../../../lib/mergeConfidenceScore";
import { isPlanningReviewMetric } from "../../../lib/planningReviewScore";
import { workOrderCardPullRequestIsMergeable } from "../../../lib/workOrderCardPullRequest";
import { splitRunDecisionTone } from "../splitRunFooter";
import { attentionToneClassName } from "../splitRunNoteActionStyle";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { pullRequestLabel } from "../../../lib/workOrderPullRequest";
import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderMergeableChip, WorkOrderPullRequestChip } from "../../../workOrders/WorkOrderPullRequestChip";
import { OwnerSpendValue } from "../../work-order-popup-redesign/popupShared";
import { useLiveHeaderSpendOverlay } from "../liveHeaderSpendContext";
import { ConsoleCheckRows } from "./consoleCheckRows";
import type { SplitRunFixture } from "../splitRunMocks";
import { splitRunPanelArtifacts } from "../splitRunPopupModel";
import { isPullRequestReviewFooter, pullRequestReviewNote } from "../splitRunPullRequestReview";
import type { SplitRunSource } from "../splitRunSource";
import { WorkOrderSplitRunSource } from "../WorkOrderSplitRunSource";
import type { AutomationStage, outcomeSummary } from "./automationsViewModel";
import { StaticStatusGlyph } from "./redesignShared";

/**
 * Reads top to bottom by priority: the decision, then what the run
 * produced (pull request, checks, artifacts), then reference details.
 * The strip under the status never goes blank: a decision when there is
 * one, the live run while an automation works, a waiting note otherwise.
 */
export function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
  pullRequests,
  artifacts,
  panelReview,
  source,
  actionBusy = false,
  onStopLiveRun,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
  pullRequests?: FactoriesFactoryPullRequest[];
  /** Every artifact on this task. Merged with what the runs produced. */
  artifacts?: FactoriesWorkOrderArtifact[];
  panelReview?: ReactNode;
  source?: SplitRunSource;
  actionBusy?: boolean;
  /** Cancels the live run. The live note shows Stop only when set. */
  onStopLiveRun?: () => void;
}) {
  const liveSpend = useLiveHeaderSpendOverlay();
  const spend = overlayHeaderSpend(outcome.spend, outcome.tokens, liveSpend);
  const panel = consolePanelFacts({ fixture, outcome, stages, pullRequests, artifacts, source, panelReview });
  return (
    <aside className="lg:sticky lg:top-0 lg:self-start" data-testid="redesign-console-summary">
      <Frame variant="default" spacing="sm" stacked className="[--frame-radius:var(--radius-lg)]">
        <FrameHeader>
          <FrameTitle className="flex items-center gap-2">
            <StaticStatusGlyph status={outcome.status} />
            {outcome.statusLabel}
          </FrameTitle>
          {panel.showsStrip ? null : <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>}
        </FrameHeader>
        <SummaryDecisionStrip
          hasDecision={panel.hasDecision}
          panelReview={panelReview}
          decisionClassName={panel.decisionClassName}
          isLive={panel.isLive}
          liveStage={panel.liveStage}
          footer={fixture.footer}
          actionBusy={actionBusy}
          onStop={onStopLiveRun}
        />
        {panel.panelPullRequests.length > 0 ? (
          <FramePanel className="flex flex-col gap-2.5 py-3" data-testid="redesign-console-pull-requests">
            <span className="text-[12px] font-medium text-muted-foreground">
              {panel.panelPullRequests.length === 1 ? "Pull request" : "Pull requests"}
            </span>
            {panel.panelPullRequests.map((pullRequest, index) => (
              <PanelPullRequest key={pullRequest.id ?? pullRequest.url ?? index} pullRequest={pullRequest} />
            ))}
          </FramePanel>
        ) : null}
        {panel.mergeChecks.length > 0 ? (
          <FramePanel className="py-3">
            <ConsoleCheckRows checks={panel.mergeChecks} />
          </FramePanel>
        ) : null}
        {panel.otherChecks.length > 0 ? (
          <FramePanel className="py-3">
            <ConsoleCheckRows
              checks={panel.otherChecks}
              title="Checks"
              testId="redesign-console-other-checks"
              defaultOpen
            />
          </FramePanel>
        ) : null}
        {panel.panelArtifacts.length > 0 ? (
          <FramePanel className="flex flex-col gap-2 py-3" data-testid="redesign-console-artifacts">
            <span className="text-[12px] font-medium text-muted-foreground">Artifacts</span>
            {panel.panelArtifacts.map((artifact) => (
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
          {panel.duration ? <SummaryRow label="Duration">{panel.duration}</SummaryRow> : null}
          <SummaryRow label="Spend">
            <OwnerSpendValue
              costUsd={spend.costUsd}
              usageByModel={fixture.usageByModel}
              usageByMachineType={fixture.usageByMachineType}
            />{" "}
            <span className="text-muted-foreground">· {spend.tokensLabel}</span>
          </SummaryRow>
        </FramePanel>
      </Frame>
    </aside>
  );
}

function consolePanelFacts({
  fixture,
  outcome,
  stages,
  pullRequests,
  artifacts,
  source,
  panelReview,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
  pullRequests?: FactoriesFactoryPullRequest[];
  artifacts?: FactoriesWorkOrderArtifact[];
  source?: SplitRunSource;
  panelReview?: ReactNode;
}) {
  const hasDecision = Boolean(fixture.footer.attentionCard && fixture.footer.note);
  const liveStage = stages.find((stage) => stage.status === "running");
  const isLive = fixture.footer.kind === "running" || Boolean(liveStage);
  const reviewedHref = panelReview ? reviewStripPullRequestHref(fixture) : undefined;
  const panelPullRequests = withoutPullRequest(
    pullRequests?.length ? pullRequests : outcome.pullRequests,
    reviewedHref,
  );
  const hasPullRequest = panelPullRequests.length > 0 || Boolean(reviewedHref);
  const taskArtifacts = artifacts ?? [];
  const taskArtifactIds = new Set(taskArtifacts.map((artifact) => artifact.id).filter(Boolean));
  const stageArtifacts = stages
    .flatMap((stage) => stage.outputs.artifacts)
    .filter((artifact) => !artifact.id || !taskArtifactIds.has(artifact.id));
  const panelArtifacts = splitRunPanelArtifacts([...taskArtifacts, ...stageArtifacts], source).filter(
    (artifact) => !(hasPullRequest && isBranchArtifact(artifact)),
  );
  return {
    hasDecision,
    liveStage,
    isLive,
    showsStrip: hasDecision ? Boolean(panelReview) : true,
    decisionClassName: hasDecision
      ? attentionToneClassName(
          isPullRequestReviewFooter(fixture.footer) ? "done" : splitRunDecisionTone(fixture.footer),
        )
      : undefined,
    panelPullRequests,
    panelArtifacts,
    // Merge confidence stays on its own list. Other non-planning checks stay
    // reachable here when no verify card copied them.
    mergeChecks: consoleCheckList(fixture.checks)?.metrics ?? [],
    otherChecks: fixture.checks.filter(
      (check) =>
        !isMergeConfidenceMetric(check) &&
        !isPlanningReviewMetric(check) &&
        !isScoreCheckName(check.name) &&
        check.key !== CONFIDENCE_CHECK_KEY &&
        check.name !== CONFIDENCE_CHECK_NAME,
    ),
    duration: /\d/.test(outcome.duration) ? outcome.duration : undefined,
  };
}

function SummaryDecisionStrip({
  hasDecision,
  panelReview,
  decisionClassName,
  isLive,
  liveStage,
  footer,
  actionBusy,
  onStop,
}: {
  hasDecision: boolean;
  panelReview?: ReactNode;
  decisionClassName?: string;
  isLive: boolean;
  liveStage?: AutomationStage;
  footer: SplitRunFixture["footer"];
  actionBusy: boolean;
  onStop?: () => void;
}) {
  if (hasDecision) {
    return panelReview ? <FramePanel className={cn("py-3", decisionClassName)}>{panelReview}</FramePanel> : null;
  }
  if (isLive) {
    return <ConsoleLiveNote stage={liveStage} footer={footer} actionBusy={actionBusy} onStop={onStop} />;
  }
  return <ConsoleWaitingNote />;
}

/** The strip fills its panel section: the tint sits on the panel, the alert stays flat. */
const FLAT_ALERT_CLASSNAME = "rounded-none border-0 bg-transparent p-0";

/**
 * Strip while an automation works: which automation, on what, and Stop.
 * The card glyph already spins, so this note stays static. Blue, as the
 * Running status everywhere else.
 */
function ConsoleLiveNote({
  stage,
  footer,
  actionBusy,
  onStop,
}: {
  stage?: AutomationStage;
  footer: SplitRunFixture["footer"];
  actionBusy: boolean;
  onStop?: () => void;
}) {
  // A footer note describes the live run only while the footer itself is
  // running; on a follow-up run (kind "waiting") it can be stale.
  const note = footer.kind === "running" ? footer.note : undefined;
  // The automation name, as on the card. A stage name can be an activity title.
  const stageName = stage ? stage.componentName.trim() || stage.name : undefined;
  const headline = note?.headline ?? (stageName ? `${stageName} is running` : "An automation is running");
  const text = note?.text ?? "The log shows live progress.";
  return (
    <FramePanel
      className="border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] py-3"
      data-testid="redesign-console-live-note"
    >
      <Alert className={FLAT_ALERT_CLASSNAME}>
        <AlertTitle>{headline}</AlertTitle>
        <AlertDescription>{text}</AlertDescription>
        {onStop && !actionBusy ? (
          <AlertAction>
            <Button type="button" variant="outline" size="sm" onClick={onStop} data-testid="redesign-console-stop-run">
              Stop
            </Button>
          </AlertAction>
        ) : null}
      </Alert>
    </FramePanel>
  );
}

/** Strip for an open task with no decision and no live run. */
function ConsoleWaitingNote() {
  return (
    <FramePanel className="py-3" data-testid="redesign-console-waiting-note">
      <Alert className={FLAT_ALERT_CLASSNAME}>
        <AlertTitle>This task is waiting</AlertTitle>
        <AlertDescription>No automation is running. Review the run output to decide the next step.</AlertDescription>
      </Alert>
    </FramePanel>
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

function SummaryRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3 text-[13px]">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right text-foreground tabular-nums">{children}</span>
    </div>
  );
}
