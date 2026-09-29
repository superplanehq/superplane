import type { SuperplaneComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { RunsTabPanel } from "@/components/CanvasToolSidebar/RunsTabPanel";
import { RunsSidebarHrefProvider, type RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { useMemo } from "react";

interface SettingsAutomationRunsSidebarProps {
  canvasId: string;
  runHrefFor?: RunsSidebarHrefForRun;
  workflowNodes?: SuperplaneComponentsNode[];
  selectedRunId: string | null;
  onSelectRun: (runId: string | null) => void;
}

/** Canvas ListRuns list beside a factory settings automation preview. */
export function SettingsAutomationRunsSidebar({
  canvasId,
  runHrefFor,
  workflowNodes = [],
  selectedRunId,
  onSelectRun,
}: SettingsAutomationRunsSidebarProps) {
  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, Boolean(canvasId));
  const runs = useMemo(() => runsQuery.data?.pages.flatMap((page) => page?.runs ?? []) ?? [], [runsQuery.data]);

  return (
    <RunsSidebarHrefProvider hrefForRun={runHrefFor}>
      <CanvasRunsSidebar isOpen>
        <RunsTabPanel
          canvasId={canvasId}
          runs={runs}
          selectedRunId={selectedRunId}
          onSelectRun={onSelectRun}
          hasNextPage={runsQuery.hasNextPage}
          isFetchingNextPage={runsQuery.isFetchingNextPage}
          onLoadMore={() => void runsQuery.fetchNextPage()}
          isLoading={runsQuery.isPending}
          isError={runsQuery.isError}
          onRetry={() => void runsQuery.refetch()}
          workflowNodes={workflowNodes}
        />
      </CanvasRunsSidebar>
    </RunsSidebarHrefProvider>
  );
}
