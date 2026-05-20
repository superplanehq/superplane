import { useCallback, useEffect, useState, type ReactNode } from "react";
import { AgentTabPanel } from "./AgentTabPanel";
import { EmptyToolTab } from "./EmptyToolTab";
import { SidebarShell } from "./SidebarShell";
import { ToolTabsHeader } from "./ToolTabsHeader";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const TAB_AGENT = "agent",
  TAB_RUNS = "runs",
  TAB_VERSIONS = "versions";

type CanvasToolSidebarMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard";

export interface CanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
  mode?: CanvasToolSidebarMode;
  onSelectRuns?: () => void;
  onExitRunsMode?: () => void;
  runsContent?: ReactNode;
  isVersionControlOpen?: boolean;
  onToggleVersionControl?: () => void;
  versionsContent?: ReactNode;
}

export function CanvasToolSidebar({
  toolSidebarState,
  mode = "default",
  onSelectRuns,
  onExitRunsMode,
  runsContent,
  isVersionControlOpen,
  onToggleVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.isToolSidebarOpen || !toolSidebarState.canvasId) {
    return null;
  }

  return (
    <OpenCanvasToolSidebar
      toolSidebarState={toolSidebarState}
      mode={mode}
      onSelectRuns={onSelectRuns}
      onExitRunsMode={onExitRunsMode}
      runsContent={runsContent}
      isVersionControlOpen={isVersionControlOpen}
      onToggleVersionControl={onToggleVersionControl}
      versionsContent={versionsContent}
    />
  );
}

function OpenCanvasToolSidebar({
  toolSidebarState,
  mode = "default",
  onSelectRuns,
  onExitRunsMode,
  runsContent,
  isVersionControlOpen,
  onToggleVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  const showRunsTab = Boolean(onSelectRuns || mode === "runs" || runsContent);
  const showVersionsTab = Boolean(onToggleVersionControl || isVersionControlOpen || versionsContent);
  const [activeTab, setActiveTab] = useState(() => {
    if (mode === "runs" && showRunsTab) return TAB_RUNS;
    if (isVersionControlOpen && showVersionsTab) return TAB_VERSIONS;
    return TAB_AGENT;
  });

  useEffect(() => {
    if (mode === "runs" && showRunsTab) {
      setActiveTab(TAB_RUNS);
      return;
    }
    if (isVersionControlOpen && showVersionsTab) {
      setActiveTab(TAB_VERSIONS);
      return;
    }
    setActiveTab((currentTab) => (currentTab === TAB_RUNS || currentTab === TAB_VERSIONS ? TAB_AGENT : currentTab));
  }, [isVersionControlOpen, mode, showRunsTab, showVersionsTab]);

  const tabs = [
    { value: TAB_AGENT, label: "Agent" },
    ...(showRunsTab ? ([{ value: TAB_RUNS, label: "Runs" }] as const) : []),
    ...(showVersionsTab ? ([{ value: TAB_VERSIONS, label: "Versions" }] as const) : []),
  ] as const;

  const handleClose = useCallback(() => {
    if (activeTab === TAB_RUNS) {
      onExitRunsMode?.();
    }

    if (activeTab === TAB_VERSIONS && isVersionControlOpen) {
      onToggleVersionControl?.();
    }

    toolSidebarState.closeToolSidebar();
  }, [activeTab, isVersionControlOpen, onExitRunsMode, onToggleVersionControl, toolSidebarState]);

  const handleTabSelect = useCallback(
    (nextTab: typeof TAB_AGENT | typeof TAB_RUNS | typeof TAB_VERSIONS) => {
      setActiveTab(nextTab);

      if (nextTab === TAB_RUNS) {
        if (isVersionControlOpen) onToggleVersionControl?.();
        if (mode !== "runs") {
          toolSidebarState.openToolSidebar();
          onSelectRuns?.();
        }
        return;
      }

      if (mode === "runs") onExitRunsMode?.();

      if (nextTab === TAB_VERSIONS) {
        if (!isVersionControlOpen) {
          toolSidebarState.openToolSidebar();
          onToggleVersionControl?.();
        }
        return;
      }

      if (isVersionControlOpen) onToggleVersionControl?.();
    },
    [isVersionControlOpen, mode, onExitRunsMode, onSelectRuns, onToggleVersionControl, toolSidebarState],
  );

  return (
    <SidebarShell>
      <div className="flex min-h-0 flex-1 flex-col gap-0">
        <ToolTabsHeader
          tabs={tabs}
          activeTab={activeTab}
          onSelectTab={(value) => handleTabSelect(value as typeof TAB_AGENT | typeof TAB_RUNS | typeof TAB_VERSIONS)}
          onClose={handleClose}
        />

        {activeTab === TAB_AGENT ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col overflow-hidden" role="tabpanel">
            <AgentTabPanel toolSidebarState={toolSidebarState} />
          </div>
        ) : null}
        {activeTab === TAB_RUNS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            {runsContent ?? <EmptyToolTab />}
          </div>
        ) : null}
        {activeTab === TAB_VERSIONS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            {versionsContent ?? <EmptyToolTab />}
          </div>
        ) : null}
      </div>
    </SidebarShell>
  );
}
