import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Link } from "@/components/Link/link";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { Pencil } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";

import { SettingsAutomationCanvas } from "./SettingsAutomationCanvas";
import { SettingsAutomationRunsSidebar } from "./SettingsAutomationRunsSidebar";
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

/** Read-only automation canvas with the canvas ListRuns sidebar on the left. */
export function SettingsAutomationWorkspace({
  graph,
  testId,
  canvasId,
  runHrefFor,
  workflowNodes,
  editHref,
  editLabel = DEFAULT_EDIT_LABEL,
  editTestId = DEFAULT_EDIT_TEST_ID,
}: SettingsAutomationWorkspaceProps) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const runsQuery = useInfiniteCanvasRuns(canvasId ?? "", {}, Boolean(canvasId));
  const selectedRunFromList = useMemo(
    () => runsQuery.data?.pages.flatMap((page) => page?.runs ?? []).find((run) => run.id === selectedRunId) ?? null,
    [runsQuery.data, selectedRunId],
  );
  const runCanvas = useSettingsAutomationRunCanvas({
    organizationId: graph.organizationId,
    canvasId,
    selectedRunId,
    selectedRunFromList,
    liveGraph: graph,
  });

  return (
    <section
      className="relative flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Automation"
      data-testid={testId}
      data-selected-run-id={selectedRunId ?? undefined}
    >
      <div className="flex min-h-[18rem] min-w-0 flex-1 overflow-hidden">
        {canvasId ? (
          <SettingsAutomationRunsSidebar
            canvasId={canvasId}
            runHrefFor={runHrefFor}
            workflowNodes={workflowNodes}
            selectedRunId={selectedRunId}
            onSelectRun={setSelectedRunId}
          />
        ) : null}
        <div className="min-h-0 min-w-0 flex-1">
          <SettingsAutomationCanvas
            graph={runCanvas.graph}
            isRunInspectionMode={runCanvas.isRunInspectionMode}
            runCanvasLoading={runCanvas.runCanvasLoading}
            selectedRun={runCanvas.selectedRun}
            runParticipantNodeIds={runCanvas.runParticipantNodeIds}
            fitAllRequest={runCanvas.fitAllRequest}
            fitAllFocusNodeIds={runCanvas.fitAllFocusNodeIds}
          />
        </div>
      </div>
      {editHref ? <SettingsAutomationCanvasEdit href={editHref} label={editLabel} testId={editTestId} /> : null}
    </section>
  );
}
