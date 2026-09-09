import type { SuperplaneComponentsNode } from "@/api-client";
import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Link } from "@/components/Link/link";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
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
}

/** Square-corner edit link for factory settings headers. */
export function SettingsAutomationEditLink({
  href,
  children,
  testId = "split-run-canvas-edit",
}: {
  href: string;
  children: string;
  testId?: string;
}) {
  return (
    <Link href={href} className={cn(buttonVariants({ size: "sm" }), "rounded-md")} data-testid={testId}>
      {children}
    </Link>
  );
}

/** Tabs and the edit link on one row, under the popup title. */
export function SettingsAutomationHeaderRow({
  tabs,
  editHref,
  editLabel,
  editTestId,
}: {
  tabs?: ReactNode;
  editHref?: string;
  editLabel: string;
  editTestId?: string;
}) {
  if (!tabs && !editHref) {
    return null;
  }

  return (
    <div className="mt-3 flex items-center justify-between gap-3" data-testid="settings-automation-header-row">
      <div className="min-w-0">{tabs}</div>
      {editHref ? (
        <SettingsAutomationEditLink href={editHref} testId={editTestId}>
          {editLabel}
        </SettingsAutomationEditLink>
      ) : null}
    </div>
  );
}

/** Read-only automation canvas with the canvas ListRuns sidebar on the left. */
export function SettingsAutomationWorkspace({
  graph,
  testId,
  canvasId,
  runHrefFor,
  workflowNodes,
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
      className="flex min-h-0 min-w-0 flex-1 flex-col"
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
    </section>
  );
}
