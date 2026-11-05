import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import { Button } from "@/components/ui/button";
import { overlayHeaderSpend } from "@/lib/overlayHeaderSpend";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { type ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { CONFIDENCE_CHECK_KEY, CONFIDENCE_CHECK_NAME, isScoreCheckName } from "../../../lib/confidenceScore";
import {
  checksForRepository,
  isMergeConfidenceMetric,
  splitMergeChecksByPullRequest,
} from "../../../lib/mergeConfidenceScore";
import { isPlanningReviewMetric } from "../../../lib/planningReviewScore";
import { workOrderCardPullRequestShowsMergeable } from "../../../lib/workOrderCardPullRequest";
import { splitRunDecisionTone } from "../splitRunFooter";
import { attentionToneClassName } from "../splitRunNoteActionStyle";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import {
  pullRequestLabel,
  pullRequestRepositoryLabel,
  pullRequestsSpanRepositories,
} from "../../../lib/workOrderPullRequest";
import { OrgUserReference, EmptyOwnerMark } from "../../../OrgUserReference";
import { OwnerAssignTrigger } from "../../../WorkOrderAssigneesPopover";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderMergeableChip, WorkOrderPullRequestChip } from "../../../workOrders/WorkOrderPullRequestChip";
import { OwnerSpendValue } from "../../work-order-popup-redesign/popupShared";
import { useLiveHeaderSpendOverlay } from "../liveHeaderSpendContext";
import { ConsoleCheckRows, PullRequestConfidence } from "./consoleCheckRows";
import { PullRequestChecksContext } from "./pullRequestChecksContext";
import type { SplitRunFixture } from "../splitRunMocks";
import { splitRunPanelArtifacts } from "../splitRunPopupModel";
import { isPullRequestReviewFooter, openReviewPullRequests, pullRequestReviewNote } from "../splitRunPullRequestReview";
import type { SplitRunSource } from "../splitRunSource";
import type { AutomationStage, outcomeSummary } from "./automationsViewModel";
import { StaticStatusGlyph } from "./redesignShared";

/**
 * Reads top to bottom by priority: the decision, then what the run
 * produced (pull request, checks, artifacts), then reference details.
 * The strip under the status never goes blank: a decision when there is
 * one, the live run while an automation works, a waiting note otherwise.
 */
type ConsoleSummaryPanelProps = {
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
  organizationId?: string;
  assigneeIds?: string[];
  owner?: OrgUserDisplay;
  canEditOwner?: boolean;
  ownerBusy?: boolean;
  onOwnerSave?: (assigneeIds: string[]) => Promise<void>;
};

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
  organizationId,
  assigneeIds,
  owner,
  canEditOwner = false,
  ownerBusy = false,
  onOwnerSave,
}: ConsoleSummaryPanelProps) {
  const liveSpend = useLiveHeaderSpendOverlay();
  const spend = overlayHeaderSpend(outcome.spend, outcome.tokens, liveSpend);
  const panel = consolePanelFacts({ fixture, outcome, stages, pullRequests, artifacts, source, panelReview });
  const shownOwner = owner ?? outcome.owner;
  const shownAssigneeIds = assigneeIds ?? fixture.assigneeIds;
  return (
    <aside className="lg:sticky lg:top-0 lg:self-start" data-testid="redesign-console-summary">
      <PullRequestChecksContext.Provider value={panel.pullRequestChecks}>
        <Frame variant="default" spacing="sm" stacked className="[--frame-radius:var(--radius-lg)]">
          <FrameHeader>
            <FrameTitle className="flex items-center gap-2">
              <StaticStatusGlyph status={outcome.status} />
              {outcome.statusLabel}
            </FrameTitle>
            {panel.showsStrip ? null : (
              <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>
            )}
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
                <PanelPullRequest
                  key={pullRequest.id ?? pullRequest.url ?? index}
                  pullRequest={pullRequest}
                  checks={fixture.checks}
                  showRepository={panel.showsPullRequestRepository}
                />
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
            <SummaryRow label="Owner" truncate={false}>
              <OwnerSummaryValue
                organizationId={organizationId}
                assigneeIds={shownAssigneeIds}
                owner={shownOwner}
                canEditOwner={canEditOwner}
                ownerBusy={ownerBusy}
                onOwnerSave={onOwnerSave}
              />
            </SummaryRow>
            <StartSummaryRow fixture={fixture} />
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
      </PullRequestChecksContext.Provider>
    </aside>
  );
}

function StartSummaryRow({ fixture }: { fixture: SplitRunFixture }) {
  return (
    <SummaryRow label={fixture.autoStarted ? "Auto-started" : "Started"} truncate={false}>
      <span className="inline-flex items-center gap-1.5 tabular-nums">
        {fixture.startedBy && !fixture.autoStarted ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                tabIndex={0}
                role="img"
                aria-label={`Started by ${fixture.startedBy.name}`}
                className="inline-flex shrink-0"
              >
                <OrgUserReference display={fixture.startedBy} size="xs" showName={false} />
              </span>
            </TooltipTrigger>
            <TooltipContent>Started by {fixture.startedBy.name}</TooltipContent>
          </Tooltip>
        ) : null}
        {fixture.startedLabel.replace(/^Started\s+/i, "")}
      </span>
    </SummaryRow>
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
  const taskPullRequests = pullRequests?.length ? pullRequests : outcome.pullRequests;
  const reviewedHrefs = panelReview ? reviewStripPullRequestHrefs(fixture, taskPullRequests) : [];
  const panelPullRequests = withoutPullRequests(taskPullRequests, reviewedHrefs);
  const hasPullRequest = panelPullRequests.length > 0 || reviewedHrefs.length > 0;
  const mergeChecks = splitMergeChecksByPullRequest(
    fixture.checks,
    taskPullRequests.map((pullRequest) => pullRequest.repository),
  );
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
    showsPullRequestRepository: pullRequestsSpanRepositories(taskPullRequests),
    panelArtifacts,
    // Each pull request row shows its own merge checks. The list keeps the
    // rest. Other non-planning checks stay reachable here when no verify
    // card copied them.
    pullRequestChecks: mergeChecks.byRepository,
    mergeChecks: mergeChecks.unattached,
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
  const headline = note?.headline ?? (stageName ? `${stageName} in progress` : "Automation in progress");
  const text = note?.text ?? "View the log for live progress.";
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

/** The pull requests the decision strip already shows. The panel lists each only once. */
function reviewStripPullRequestHrefs(fixture: SplitRunFixture, pullRequests: FactoriesFactoryPullRequest[]): string[] {
  if (!fixture.footer.note || !isPullRequestReviewFooter(fixture.footer)) {
    return [];
  }
  const open = openReviewPullRequests(pullRequests);
  if (open.length > 1) {
    return open.map((pullRequest) => pullRequest.url ?? "");
  }
  const href = pullRequestReviewNote(fixture.footer.note)?.href;
  return href ? [href] : [];
}

function withoutPullRequests(
  pullRequests: FactoriesFactoryPullRequest[],
  hrefs: string[],
): FactoriesFactoryPullRequest[] {
  if (hrefs.length === 0) {
    return pullRequests;
  }
  const shown = new Set(hrefs.map(normalizedPullRequestUrl));
  return pullRequests.filter((pullRequest) => !shown.has(normalizedPullRequestUrl(pullRequest.url ?? "")));
}

function normalizedPullRequestUrl(url: string): string {
  return url.replace(/\/$/, "");
}

/** The pull request supersedes its branch, so a listed branch would repeat it. */
function isBranchArtifact(artifact: FactoriesWorkOrderArtifact): boolean {
  return (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase() === "branch";
}

/**
 * One pull request, first-class: the title links to the provider, the
 * chips below carry state and mergeability, as on the board card.
 */
function PanelPullRequest({
  pullRequest,
  checks,
  showRepository,
}: {
  pullRequest: FactoriesFactoryPullRequest;
  checks: SplitRunFixture["checks"];
  showRepository: boolean;
}) {
  const href = safeExternalUrl(pullRequest.url);
  const title = pullRequest.title?.trim() || pullRequestLabel(pullRequest);
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      {showRepository ? (
        <span className="truncate text-[12px] leading-4 text-muted-foreground">
          {pullRequestRepositoryLabel(pullRequest)}
        </span>
      ) : null}
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
        {workOrderCardPullRequestShowsMergeable(pullRequest, checksForRepository(checks, pullRequest.repository)) ? (
          <WorkOrderMergeableChip />
        ) : null}
      </div>
      <PullRequestConfidence repository={pullRequest.repository} />
    </div>
  );
}

function OwnerSummaryValue({
  organizationId,
  assigneeIds,
  owner,
  canEditOwner,
  ownerBusy,
  onOwnerSave,
}: {
  organizationId?: string;
  assigneeIds: string[] | undefined;
  owner: OrgUserDisplay;
  canEditOwner: boolean;
  ownerBusy: boolean;
  onOwnerSave?: (assigneeIds: string[]) => Promise<void>;
}) {
  const mark =
    assigneeIds === undefined || assigneeIds.length > 0 ? (
      <OrgUserReference display={owner} size="xs" nameClassName="text-[13px]" />
    ) : (
      <EmptyOwnerMark />
    );

  if (!canEditOwner || !organizationId || !onOwnerSave) {
    return mark;
  }

  return (
    <OwnerAssignTrigger
      organizationId={organizationId}
      selectedIds={assigneeIds ?? []}
      canAssign
      isSaving={ownerBusy}
      onSave={onOwnerSave}
      align="end"
      label={assigneeIds && assigneeIds.length > 0 ? `Owner: ${owner.name}` : "Assign owner"}
      testId="task-edit-owner"
    >
      {mark}
    </OwnerAssignTrigger>
  );
}

function SummaryRow({ label, children, truncate = true }: { label: string; children: ReactNode; truncate?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3 text-[13px]">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span
        className={cn(
          "min-w-0 max-w-full text-right text-foreground",
          truncate ? "truncate tabular-nums" : "flex justify-end overflow-x-hidden",
        )}
      >
        {children}
      </span>
    </div>
  );
}
