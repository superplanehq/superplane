import { Activity } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";
import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";
import { WorkOrderPullRequestInline } from "../../WorkOrderPullRequestInline";

import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { canvasNodesForRunnerModel, phaseWithRunnerModel } from "./draftStartModel";
import { attachArtifactsToStream, type StreamArtifactIndex } from "./attachStreamArtifacts";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import { groupSplitRunActivities, type PullRequestActivityGroup } from "./splitRunActivityGroups";
import { autoExpandedPhaseId, type SplitRunFixture, type SplitRunPhase, type SplitRunPhaseId } from "./splitRunMocks";
import { splitRunPhaseRunHref } from "./splitRunPopupModel";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import type { useFollowLogScroll } from "./useFollowLogScroll";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";

type WorkOrderSplitRunBodyProps = {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  lineId?: string;
  fixture: SplitRunFixture;
  canUpdate?: boolean;
  footerActions: SplitRunFooterActions;
  files?: FilesFile[];
};

export type WorkOrderSplitRunPopupProps = Omit<WorkOrderSplitRunBodyProps, "footerActions"> & {
  onClose?: () => void;
  fixed?: boolean;
  onDispatch?: (model?: string, thinkingLevel?: string) => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
  canUpdate?: boolean;
};

type SplitRunFollow = ReturnType<typeof useFollowLogScroll<HTMLOListElement>>;

interface SplitRunPhaseRenderOptions {
  markdownName?: boolean;
  inlineRunDetails?: boolean;
  collapsible?: boolean;
}

type SplitRunPhaseRenderer = (phase: SplitRunPhase, options?: SplitRunPhaseRenderOptions) => ReactNode;

/** Phase log for a work-order popup. The popup wraps this. */
export function WorkOrderSplitRunBody({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  fixture,
  canUpdate = true,
  footerActions,
  follow,
  onStreamTick,
  files,
}: WorkOrderSplitRunBodyProps & {
  follow: SplitRunFollow;
  onStreamTick: (tick: string) => void;
}) {
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
  const streamLengthsRef = useRef<Record<string, number>>({});
  const currentPhaseId = fixture.currentPhaseId;
  const expandedPhaseId = autoExpandedPhaseId(fixture);
  const { taskAutomationPhases, pullRequestActivityGroups } = useMemo(
    () => groupSplitRunActivities(fixture.phases),
    [fixture.phases],
  );
  useEffect(() => {
    setOpenPhaseId(expandedPhaseId);
  }, [currentPhaseId, expandedPhaseId]);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const demoArtifacts = !organizationId;
  const onStreamLength = useCallback(
    (phaseId: string, length: number) => {
      if (streamLengthsRef.current[phaseId] === length) {
        return;
      }
      streamLengthsRef.current = { ...streamLengthsRef.current, [phaseId]: length };
      onStreamTick(fixture.phases.map((entry) => streamLengthsRef.current[entry.id] ?? 0).join(":"));
    },
    [fixture.phases, onStreamTick],
  );
  const liveOrder = Boolean(organizationId && factoryId && orderId);
  const automationStop = (entry: SplitRunPhase) => {
    const appId = entry.appId;
    const runId = entry.runId;
    if (!canUpdate || !liveOrder || entry.status !== "running" || !appId || !runId) {
      return undefined;
    }
    return () => void footerActions.handleStopAutomation({ appId, runId });
  };
  const automationRerun = (entry: SplitRunPhase) => {
    if (!canUpdate || !liveOrder || entry.status !== "failed" || entry.stepIndex == null) {
      return undefined;
    }
    return () =>
      void footerActions.handleStop("rerun-step", {
        kind: "failed",
        lineName: fixture.lineName,
        stepIndex: entry.stepIndex,
      });
  };
  const renderPhase: SplitRunPhaseRenderer = (entry, options) => (
    <SplitRunPhaseLogItem
      entry={entry}
      headerLeading={<SplitRunPhaseTimestamp phase={entry} />}
      markdownName={options?.markdownName}
      inlineRunDetails={options?.inlineRunDetails}
      collapsible={options?.collapsible}
      organizationId={organizationId}
      factoryKey={factoryKey}
      orderNumber={orderNumber}
      lineId={lineId}
      expanded={entry.id === openPhaseId}
      selectedNodeId={nodeId}
      onSelectNode={setNodeId}
      demoArtifacts={demoArtifacts}
      artifactIndex={artifactIndex}
      onStop={automationStop(entry)}
      onRerun={automationRerun(entry)}
      actionBusy={footerActions.busy}
      onStreamLength={onStreamLength}
      files={files}
      onToggle={() => {
        setNodeId(null);
        setOpenPhaseId((current) => (current === entry.id ? null : entry.id));
      }}
    />
  );

  return (
    <div className="relative flex min-h-0 min-w-0 flex-1 flex-col" data-testid="split-run-log-pane">
      <ol
        ref={follow.scrollRef}
        onScroll={follow.onScroll}
        className="min-h-0 min-w-0 flex-1 list-none space-y-6 overflow-x-hidden overflow-y-auto px-3 pb-3"
        data-testid="split-run-log-scroll"
      >
        <SplitRunActivitySections
          taskAutomationPhases={taskAutomationPhases}
          pullRequestActivityGroups={pullRequestActivityGroups}
          renderPhase={renderPhase}
        />
      </ol>
      {follow.showJumpToLatest ? (
        <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-older" />
      ) : null}
    </div>
  );
}

function SplitRunActivitySections({
  taskAutomationPhases,
  pullRequestActivityGroups,
  renderPhase,
}: {
  taskAutomationPhases: SplitRunPhase[];
  pullRequestActivityGroups: PullRequestActivityGroup[];
  renderPhase: SplitRunPhaseRenderer;
}) {
  return (
    <>
      {taskAutomationPhases.length > 0 ? (
        <li className="min-w-0 first:mt-3">
          <section aria-label="Task automations" data-testid="split-run-task-automations">
            <ol className="list-none space-y-2">
              {taskAutomationPhases.map((phase) => (
                <li key={phase.id} className="min-w-0">
                  {renderPhase(phase)}
                </li>
              ))}
            </ol>
          </section>
        </li>
      ) : null}
      {pullRequestActivityGroups.length > 0 ? (
        <li className="min-w-0 first:mt-3">
          <section aria-label="Pull request activity" data-testid="split-run-pull-request-activity">
            <div className="space-y-5">
              {pullRequestActivityGroups.map((group, index) => (
                <PullRequestActivityTimeline key={group.id} group={group} index={index} renderPhase={renderPhase} />
              ))}
            </div>
          </section>
        </li>
      ) : null}
    </>
  );
}

function PullRequestActivityTimeline({
  group,
  index,
  renderPhase,
}: {
  group: PullRequestActivityGroup;
  index: number;
  renderPhase: SplitRunPhaseRenderer;
}) {
  const headingId = `split-run-pull-request-heading-${index}`;
  return (
    <section aria-labelledby={headingId} data-testid={`split-run-pull-request-${index}`}>
      <h2 id={headingId} className="mb-3 flex min-w-0 items-center gap-1.5 px-1">
        <Activity className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        {group.pullRequest ? (
          <WorkOrderPullRequestInline pullRequest={group.pullRequest} showTitle showStateIcon={false} />
        ) : (
          <span className="text-[13px] font-medium text-foreground">Pull request</span>
        )}
      </h2>
      <ol className="list-none space-y-2" data-testid={`split-run-pull-request-timeline-${index}`}>
        {group.phases.map((phase, activityIndex) => (
          <PullRequestActivityTimelineItem
            key={phase.id}
            phase={phase}
            index={`${index}-${activityIndex}`}
            renderPhase={renderPhase}
          />
        ))}
      </ol>
    </section>
  );
}

function PullRequestActivityTimelineItem({
  phase,
  index,
  renderPhase,
}: {
  phase: SplitRunPhase;
  index: string;
  renderPhase: SplitRunPhaseRenderer;
}) {
  return (
    <li className="min-w-0" data-testid={`split-run-pull-request-activity-item-${index}`}>
      {renderPhase(phase, { markdownName: true, inlineRunDetails: false, collapsible: false })}
    </li>
  );
}

function SplitRunPhaseTimestamp({ phase }: { phase: SplitRunPhase }) {
  const timestamp = phase.startedAt ?? phase.pullRequestActivity?.startedAt ?? "";
  const formatted = formatWorkOrderDateTime(new Date(timestamp));
  const fallback = phase.stream[0]?.at?.trim();
  return (
    <time
      dateTime={timestamp || undefined}
      className="shrink-0 font-mono text-[11px] font-normal text-muted-foreground"
      data-testid={`split-run-phase-time-${phase.id}`}
    >
      {formatted || fallback || "Time unavailable"}
    </time>
  );
}

function SplitRunPhaseLogItem({
  entry,
  headerLeading,
  markdownName,
  inlineRunDetails = true,
  collapsible = true,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  expanded,
  selectedNodeId,
  onSelectNode,
  demoArtifacts,
  artifactIndex,
  onStop,
  onRerun,
  actionBusy,
  onStreamLength,
  onToggle,
  files,
}: {
  entry: SplitRunPhase;
  headerLeading?: ReactNode;
  markdownName?: boolean;
  inlineRunDetails?: boolean;
  collapsible?: boolean;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  expanded: boolean;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  demoArtifacts: boolean;
  artifactIndex: StreamArtifactIndex;
  onStop?: () => void;
  onRerun?: () => void;
  actionBusy: boolean;
  onStreamLength: (phaseId: string, length: number) => void;
  onToggle: () => void;
  files?: FilesFile[];
}) {
  const [usageOpen, setUsageOpen] = useState(false);
  const loadLive = usageOpen || (inlineRunDetails && expanded);
  const live = useSplitRunLiveCanvas(organizationId, loadLive ? entry : undefined);
  const visual = useMemo(() => resolveSplitRunVisual(entry, live, { demoArtifacts }), [demoArtifacts, entry, live]);
  const stream = useMemo(
    () => attachArtifactsToStream(visual.stream, artifactIndex, entry.runId),
    [artifactIndex, entry.runId, visual.stream],
  );
  useEffect(() => {
    onStreamLength(entry.id, stream?.length ?? 0);
  }, [entry.id, onStreamLength, stream?.length]);

  return (
    <PhaseLogCard
      phase={phaseWithRunnerModel(entry, canvasNodesForRunnerModel(live.canvas?.nodes, live.canvas?.statuses))}
      expanded={expanded}
      headerLeading={headerLeading}
      markdownName={markdownName}
      showMarkdownDescription={markdownName}
      showExpandedStream={inlineRunDetails}
      collapsible={collapsible}
      stream={loadLive ? (stream ?? entry.stream) : []}
      streamLoading={loadLive && live.isLoading}
      selectedNodeId={selectedNodeId}
      onSelectNode={onSelectNode}
      organizationId={organizationId}
      canvasId={entry.appId}
      onStop={onStop}
      onRerun={onRerun}
      runHref={splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase: entry })}
      actionBusy={actionBusy}
      onToggle={onToggle}
      onUsageOpenChange={setUsageOpen}
      files={files}
    />
  );
}
