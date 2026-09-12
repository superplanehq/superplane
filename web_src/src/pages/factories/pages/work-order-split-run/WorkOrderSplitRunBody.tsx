import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { phaseWithRunnerModel } from "./draftStartModel";
import { attachArtifactsToStream, type StreamArtifactIndex } from "./attachStreamArtifacts";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
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
};

export type WorkOrderSplitRunPopupProps = Omit<WorkOrderSplitRunBodyProps, "footerActions"> & {
  onClose?: () => void;
  fixed?: boolean;
  onDispatch?: (model?: string) => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
  canUpdate?: boolean;
};

type SplitRunFollow = ReturnType<typeof useFollowLogScroll<HTMLOListElement>>;

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
}: WorkOrderSplitRunBodyProps & {
  follow: SplitRunFollow;
  onStreamTick: (tick: string) => void;
}) {
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
  const streamLengthsRef = useRef<Record<string, number>>({});
  const currentPhaseId = fixture.currentPhaseId;
  const expandedPhaseId = autoExpandedPhaseId(fixture);
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

  return (
    <div className="relative flex min-h-0 min-w-0 flex-1 flex-col" data-testid="split-run-log-pane">
      <ol
        ref={follow.scrollRef}
        onScroll={follow.onScroll}
        className="min-h-0 min-w-0 flex-1 list-none space-y-2 overflow-x-hidden overflow-y-auto px-3 pb-3"
        data-testid="split-run-log-scroll"
      >
        {fixture.phases.map((entry) => (
          <li key={entry.id} className="min-w-0 first:mt-3">
            <SplitRunPhaseLogItem
              entry={entry}
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
              onToggle={() => {
                setNodeId(null);
                setOpenPhaseId((current) => (current === entry.id ? null : entry.id));
              }}
            />
          </li>
        ))}
      </ol>
      {follow.following ? null : (
        <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-older" />
      )}
    </div>
  );
}

function SplitRunPhaseLogItem({
  entry,
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
}: {
  entry: SplitRunPhase;
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
}) {
  const [usageOpen, setUsageOpen] = useState(false);
  const live = useSplitRunLiveCanvas(organizationId, expanded || usageOpen ? entry : undefined);
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
      phase={phaseWithRunnerModel(entry, live.canvas?.nodes)}
      expanded={expanded}
      stream={stream ?? entry.stream}
      streamLoading={live.isLoading}
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
    />
  );
}
