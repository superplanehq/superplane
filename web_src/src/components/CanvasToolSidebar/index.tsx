import { useCallback, useEffect, useState, type ReactNode } from "react";
import { AgentTabPanel } from "./AgentTabPanel";
import { EmptyToolTab } from "./EmptyToolTab";
import { SidebarShell } from "./SidebarShell";
import { ToolTabsHeader } from "./ToolTabsHeader";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const TAB_AGENT = "agent",
  TAB_RUNS = "runs",
  TAB_VERSIONS = "versions";

type CanvasToolSidebarMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard" | "memory";

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
  const hasAgentTab = toolSidebarState.isAgentEnabled;
  const [activeTab, setActiveTab] = useState(() => defaultToolTab(mode, Boolean(isVersionControlOpen), hasAgentTab));

  useEffect(() => {
    if (mode === "runs") {
      setActiveTab(TAB_RUNS);
      return;
    }
    if (isVersionControlOpen) {
      setActiveTab(TAB_VERSIONS);
      return;
    }
    setActiveTab((currentTab) => {
      if ((currentTab === TAB_AGENT || currentTab === TAB_VERSIONS) && !hasAgentTab) return TAB_RUNS;
      if ((currentTab === TAB_RUNS || currentTab === TAB_VERSIONS) && hasAgentTab) return TAB_AGENT;
      return currentTab;
    });
  }, [hasAgentTab, isVersionControlOpen, mode]);

  const tabs = [
    ...(hasAgentTab ? [{ value: TAB_AGENT, label: "Agent" }] : []),
    { value: TAB_RUNS, label: "Runs" },
    { value: TAB_VERSIONS, label: "Versions" },
  ] as const;

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
        />

        {activeTab === TAB_AGENT && hasAgentTab ? (
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

function defaultToolTab(mode: CanvasToolSidebarMode, isVersionControlOpen: boolean, hasAgentTab: boolean) {
  if (mode === "runs") return TAB_RUNS;
  if (isVersionControlOpen) return TAB_VERSIONS;
  if (hasAgentTab) return TAB_AGENT;
  return TAB_RUNS;
}
