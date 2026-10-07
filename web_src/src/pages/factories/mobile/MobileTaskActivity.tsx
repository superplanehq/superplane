import type { FilesFile } from "@/api-client";
import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useMemo, useState } from "react";

import type { PhaseGlyphKind } from "../lib/linePhaseRuns";
import { PhaseGlyph } from "../pages/linePhaseGlyph";
import { attachArtifactsToStream, type StreamArtifactIndex } from "../pages/work-order-split-run/attachStreamArtifacts";
import { canvasNodesForRunnerModel, phaseWithRunnerModel } from "../pages/work-order-split-run/draftStartModel";
import { PhaseLogCard } from "../pages/work-order-split-run/PhaseLogCard";
import { SpecificModelIdsProvider } from "../pages/work-order-split-run/specificModelIds";
import { resolveSplitRunVisual } from "../pages/work-order-split-run/splitRunLiveCanvas";
import {
  splitRunStatusLabel,
  type SplitRunPhase,
  type SplitRunPhaseStatus,
} from "../pages/work-order-split-run/splitRunMocks";
import { SplitRunCheckPills } from "../pages/work-order-split-run/SplitRunReview";
import { useSplitRunLiveCanvas } from "../pages/work-order-split-run/useSplitRunLiveCanvas";
import { useSplitRunStreamArtifacts } from "../pages/work-order-split-run/useSplitRunStreamArtifacts";
import { MOBILE_TASK_COPY } from "./mobileCopy";

const PHASE_GLYPH: Record<SplitRunPhaseStatus, PhaseGlyphKind> = {
  passed: "passed",
  running: "running",
  pending: "pending",
  waiting: "waiting",
  failed: "failed",
  cancelled: "cancelled",
};

type MobileTaskActivityProps = {
  organizationId: string;
  factoryId: string;
  orderId: string;
  phases: SplitRunPhase[];
  /** Phase that opens on load. Undefined keeps every phase closed. */
  expandedPhaseId?: string;
  files?: FilesFile[];
};

/**
 * Automation phases on the task. Each row opens the same phase log as the
 * desktop task popup, so the phone does not need the full run page.
 */
export function MobileTaskActivity({
  organizationId,
  factoryId,
  orderId,
  phases,
  expandedPhaseId,
  files,
}: MobileTaskActivityProps) {
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);

  if (phases.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.noActivity}</p>;
  }
  return (
    <SpecificModelIdsProvider organizationId={organizationId}>
      <ol className="flex flex-col divide-y divide-border rounded-lg border border-border bg-card">
        {phases.map((phase) => (
          <li key={phase.id} className="min-w-0">
            <ActivityRow
              phase={phase}
              expandedByDefault={phase.id === expandedPhaseId}
              organizationId={organizationId}
              artifactIndex={artifactIndex}
              files={files}
            />
          </li>
        ))}
      </ol>
    </SpecificModelIdsProvider>
  );
}

/** A phase has a log when it already carries lines or points at a run. */
function phaseHasLog(phase: SplitRunPhase): boolean {
  return phase.stream.length > 0 || Boolean(phase.appId && phase.runId);
}

/** One automation on the line. Tap to read its log. */
function ActivityRow({
  phase,
  expandedByDefault,
  organizationId,
  artifactIndex,
  files,
}: {
  phase: SplitRunPhase;
  expandedByDefault: boolean;
  organizationId: string;
  artifactIndex: StreamArtifactIndex;
  files?: FilesFile[];
}) {
  const canExpand = phaseHasLog(phase);
  const [expanded, setExpanded] = useState(expandedByDefault && canExpand);

  return (
    <div className="flex min-w-0 flex-col" data-testid={`mobile-task-phase-${phase.id}`}>
      <button
        type="button"
        onClick={() => canExpand && setExpanded((current) => !current)}
        aria-expanded={canExpand ? expanded : undefined}
        aria-label={canExpand ? (expanded ? MOBILE_TASK_COPY.hideLog : MOBILE_TASK_COPY.showLog) : undefined}
        className="flex w-full items-start gap-3 px-3 py-2.5 text-left"
      >
        <PhaseGlyph kind={PHASE_GLYPH[phase.status]} className="mt-1" />
        <span className="min-w-0 flex-1">
          <span className="block text-[14px] font-medium text-foreground break-words">{phase.name}</span>
          <span className="block text-[12px] text-muted-foreground">
            {splitRunStatusLabel(phase.status)}
            {phase.componentName ? ` · ${phase.componentName}` : ""}
            {phase.duration ? ` · ${phase.duration}` : ""}
          </span>
          {phase.checks && phase.checks.length > 0 ? (
            <span className="mt-1.5 block">
              <SplitRunCheckPills checks={phase.checks} testId={`mobile-task-phase-checks-${phase.id}`} />
            </span>
          ) : null}
        </span>
        {canExpand ? (
          <ChevronDown
            className={cn("mt-1 size-4 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-180")}
            aria-hidden
          />
        ) : null}
      </button>
      {expanded && canExpand ? (
        <div className="min-w-0 border-t border-border px-2 py-2" data-testid={`mobile-task-phase-log-${phase.id}`}>
          <PhaseLog phase={phase} organizationId={organizationId} artifactIndex={artifactIndex} files={files} />
        </div>
      ) : null}
    </div>
  );
}

/**
 * Live phase log, built the way the desktop popup builds it. The row above
 * already names the phase, so the card hides its own header.
 */
function PhaseLog({
  phase,
  organizationId,
  artifactIndex,
  files,
}: {
  phase: SplitRunPhase;
  organizationId: string;
  artifactIndex: StreamArtifactIndex;
  files?: FilesFile[];
}) {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const visual = useMemo(() => resolveSplitRunVisual(phase, live, { demoArtifacts: false }), [live, phase]);
  const stream = useMemo(
    () => attachArtifactsToStream(visual.stream, artifactIndex, phase.runId),
    [artifactIndex, phase.runId, visual.stream],
  );

  return (
    <PhaseLogCard
      phase={phaseWithRunnerModel(phase, canvasNodesForRunnerModel(live.canvas?.nodes, live.canvas?.statuses))}
      expanded
      collapsible={false}
      showHeader={false}
      stream={stream ?? phase.stream}
      streamLoading={live.isLoading}
      organizationId={organizationId}
      canvasId={phase.appId}
      files={files}
    />
  );
}
