import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { cn } from "@/lib/utils";
import { Link } from "@/components/Link/link";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";
import { Pencil } from "lucide-react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { SettingsAutomationCanvas } from "./SettingsAutomationCanvas";
import { FactoryAutomationRunsSidebar } from "./factoryAutomationRunsSidebar/FactoryAutomationRunsSidebar";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { useSettingsAutomationRunCanvas } from "./useSettingsAutomationRunCanvas";

interface SettingsAutomationWorkspaceProps {
  graph: IntakeAutomationGraph;
  testId: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  workflowNodes?: SuperplaneComponentsNode[];
  editHref?: string;
  editLabel?: string;
  editTestId?: string;
  onNodeSelect?: (nodeId: string) => void;
  showRuns?: boolean;
  /** Select the newest listed run once, when this workspace opens. */
  selectLatestRun?: boolean;
  showStatusControls?: boolean;
  showFindControls?: boolean;
  focusNodeId?: string | null;
  focusNonce?: number;
  layoutFitNonce?: number | null;
  /** Keep the chosen run when this workspace unmounts. */
  selectedRunId?: string | null;
  onSelectedRunIdChange?: (runId: string | null) => void;
  className?: string;
}

const DEFAULT_EDIT_LABEL = "Edit automation";
const DEFAULT_EDIT_TEST_ID = "settings-automation-edit";

/** Tabs under the popup title. Edit lives on the canvas, not here. */
export function SettingsAutomationHeaderRow({ tabs }: { tabs?: ReactNode }) {
  if (!tabs) {
    return null;
  }

  return (
    <div className="mt-3 flex items-center gap-3" data-testid="settings-automation-header-row">
      <div className="min-w-0">{tabs}</div>
    </div>
  );
}

/** Small pencil on the dotted canvas. Use on empty and loaded automation panes. */
export function SettingsAutomationCanvasEdit({ href, label, testId }: { href: string; label: string; testId: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Link
          href={href}
          aria-label={label}
          data-testid={testId}
          className="absolute top-2 right-2 z-20 flex size-6 items-center justify-center rounded-md text-muted-foreground/80 transition-colors hover:bg-background/80 hover:text-foreground"
        >
          <Pencil className="size-3.5" aria-hidden />
        </Link>
      </TooltipTrigger>
      <TooltipContent side="left">{label}</TooltipContent>
    </Tooltip>
  );
}

/** Read-only automation canvas with the factory task-card runs sidebar. */
export function SettingsAutomationWorkspace({
  graph,
  testId,
  canvasId,
  runHrefFor,
  editHref,
  editLabel = DEFAULT_EDIT_LABEL,
  editTestId = DEFAULT_EDIT_TEST_ID,
  onNodeSelect,
  showRuns = true,
  selectLatestRun = false,
  showStatusControls = true,
  showFindControls = true,
  focusNodeId = null,
  focusNonce = 0,
  layoutFitNonce = null,
  selectedRunId: controlledRunId,
  onSelectedRunIdChange,
  className,
}: SettingsAutomationWorkspaceProps) {
  const runs = useAutomationRuns({
    graph,
    canvasId,
    selectLatestRun,
    showRuns,
    controlledRunId,
    onSelectedRunIdChange,
  });

  return (
    <AutomationWorkspaceLayout
      graph={graph}
      testId={testId}
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      editHref={editHref}
      editLabel={editLabel}
      editTestId={editTestId}
      onNodeSelect={onNodeSelect}
      showStatusControls={showStatusControls}
      showFindControls={showFindControls}
      focusNodeId={focusNodeId}
      focusNonce={focusNonce}
      layoutFitNonce={layoutFitNonce}
      runs={runs}
      className={className}
    />
  );
}

function useAutomationRuns({
  graph,
  canvasId,
  selectLatestRun,
  showRuns,
  controlledRunId,
  onSelectedRunIdChange,
}: {
  graph: IntakeAutomationGraph;
  canvasId?: string;
  selectLatestRun: boolean;
  showRuns: boolean;
  controlledRunId?: string | null;
  onSelectedRunIdChange?: (runId: string | null) => void;
}) {
  const resolvedCanvasId = canvasId ?? "";
  const organizationId = graph.organizationId ?? "";
  useCanvasRuntimeWebsocket(resolvedCanvasId, organizationId, Boolean(resolvedCanvasId && organizationId));
  const runsQuery = useInfiniteCanvasRuns(resolvedCanvasId, {}, Boolean(resolvedCanvasId));
  const listedRuns = useMemo(() => runsQuery.data?.pages.flatMap((page) => page?.runs ?? []) ?? [], [runsQuery.data]);
  const { selectedRunId, selectRun } = useSelectedCanvasRun(
    listedRuns,
    selectLatestRun,
    controlledRunId,
    onSelectedRunIdChange,
  );
  const selectedRunFromList = useMemo(
    () => listedRuns.find((run) => run.id === selectedRunId) ?? null,
    [listedRuns, selectedRunId],
  );
  const runCanvas = useSettingsAutomationRunCanvas({
    organizationId: graph.organizationId,
    canvasId,
    selectedRunId,
    selectedRunFromList,
    liveGraph: graph,
  });

  return { selectedRunId, selectRun, showRunList: Boolean(canvasId) && showRuns, runCanvas };
}

function AutomationWorkspaceLayout({
  graph,
  testId,
  canvasId,
  runHrefFor,
  editHref,
  editLabel,
  editTestId,
  onNodeSelect,
  showStatusControls,
  showFindControls,
  focusNodeId,
  focusNonce,
  layoutFitNonce,
  runs,
  className,
}: {
  graph: IntakeAutomationGraph;
  testId: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editHref?: string;
  editLabel: string;
  editTestId: string;
  onNodeSelect?: (nodeId: string) => void;
  showStatusControls: boolean;
  showFindControls: boolean;
  focusNodeId: string | null;
  focusNonce: number;
  layoutFitNonce: number | null;
  runs: ReturnType<typeof useAutomationRuns>;
  className?: string;
}) {
  return (
    <section
      className={cn("relative flex min-h-0 min-w-0 flex-1 flex-col", className)}
      aria-label="Automation"
      data-testid={testId}
      data-selected-run-id={runs.selectedRunId ?? undefined}
    >
      <div className="flex min-h-[18rem] min-w-0 flex-1 overflow-hidden">
        {runs.showRunList && canvasId ? (
          <FactoryAutomationRunsSidebar
            canvasId={canvasId}
            organizationId={graph.organizationId}
            factoryId={graph.factoryId}
            runHrefFor={runHrefFor}
            selectedRunId={runs.selectedRunId}
            onSelectRun={runs.selectRun}
          />
        ) : null}
        <div className="min-h-0 min-w-0 flex-1">
          <SettingsAutomationCanvas
            graph={runs.runCanvas.graph}
            isRunInspectionMode={runs.runCanvas.isRunInspectionMode}
            runCanvasLoading={runs.runCanvas.runCanvasLoading}
            selectedRun={runs.runCanvas.selectedRun}
            runParticipantNodeIds={runs.runCanvas.runParticipantNodeIds}
            fitAllRequest={runs.runCanvas.fitAllRequest}
            fitAllFocusNodeIds={runs.runCanvas.fitAllFocusNodeIds}
            onNodeSelect={onNodeSelect}
            showStatusControls={showStatusControls}
            showFindControls={showFindControls}
            focusNodeId={focusNodeId}
            focusNonce={focusNonce}
            layoutFitNonce={layoutFitNonce}
          />
        </div>
      </div>
      {editHref ? <SettingsAutomationCanvasEdit href={editHref} label={editLabel} testId={editTestId} /> : null}
    </section>
  );
}

function useSelectedCanvasRun(
  listedRuns: { id?: string }[],
  selectLatestRun: boolean,
  controlledRunId: string | null | undefined,
  onSelectedRunIdChange?: (runId: string | null) => void,
) {
  const [uncontrolledRunId, setUncontrolledRunId] = useState<string | null>(null);
  const latestRunSelected = useRef(false);
  const isControlled = onSelectedRunIdChange !== undefined;
  const selectedRunId = isControlled ? (controlledRunId ?? null) : uncontrolledRunId;

  function selectRun(runId: string | null) {
    if (!isControlled) {
      setUncontrolledRunId(runId);
    }
    onSelectedRunIdChange?.(runId);
  }

  useEffect(() => {
    if (!selectLatestRun || latestRunSelected.current || selectedRunId) {
      return;
    }
    const latestRunId = listedRuns.find((run) => run.id)?.id;
    if (!latestRunId) {
      return;
    }
    latestRunSelected.current = true;
    if (!isControlled) {
      setUncontrolledRunId(latestRunId);
    }
    onSelectedRunIdChange?.(latestRunId);
  }, [isControlled, listedRuns, onSelectedRunIdChange, selectLatestRun, selectedRunId]);

  return { selectedRunId, selectRun };
}
