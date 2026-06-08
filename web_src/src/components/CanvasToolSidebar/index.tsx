import { History, Rabbit, Sparkles } from "lucide-react";
import { useCallback, useEffect, useRef, useState, type MutableRefObject, type ReactNode } from "react";
import { AgentTabPanel } from "./AgentTabPanel";
import { EmptyToolTab } from "./EmptyToolTab";
import { CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, canvasToolSidebarTabFromEvent } from "./events";
import type { CanvasToolSidebarTab } from "./events";
import { SidebarShell } from "./SidebarShell";
import { ToolTabsHeader } from "./ToolTabsHeader";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const TAB_AGENT = "agent",
  TAB_RUNS = "runs",
  TAB_VERSIONS = "versions";

type CanvasToolSidebarMode = "default" | "version-live" | "version-edit" | "runs" | "console" | "memory" | "files";

export interface CanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
  mode?: CanvasToolSidebarMode;
  onSelectRuns?: () => void;
  onExitRunsMode?: () => void;
  runsContent?: ReactNode;
  isVersionControlOpen?: boolean;
  onOpenVersionControl?: () => void;
  hasAutoOpenedVersionControl?: boolean;
  onVersionControlAutoOpened?: () => void;
  onCloseVersionControl?: () => void;
  versionsContent?: ReactNode;
}

export function CanvasToolSidebar({
  toolSidebarState,
  mode = "default",
  onSelectRuns,
  onExitRunsMode,
  runsContent,
  isVersionControlOpen,
  onOpenVersionControl,
  hasAutoOpenedVersionControl,
  onVersionControlAutoOpened,
  onCloseVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.canvasId) {
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
      onOpenVersionControl={onOpenVersionControl}
      hasAutoOpenedVersionControl={hasAutoOpenedVersionControl}
      onVersionControlAutoOpened={onVersionControlAutoOpened}
      onCloseVersionControl={onCloseVersionControl}
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
  onOpenVersionControl,
  hasAutoOpenedVersionControl,
  onVersionControlAutoOpened,
  onCloseVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  const hasAgentTab = toolSidebarState.isAgentEnabled;
  const isToolSidebarOpen = toolSidebarState.isToolSidebarOpen;
  const hasAutoOpenedVersionControlInMountRef = useRef(false);
  const [activeTab, setActiveTab] = useState<CanvasToolSidebarTab>(() =>
    defaultToolTab(mode, Boolean(isVersionControlOpen), hasAgentTab),
  );
  const handleTabSelect = useCallback(
    (nextTab: typeof TAB_AGENT | typeof TAB_RUNS | typeof TAB_VERSIONS) => {
      if (nextTab === TAB_AGENT && !hasAgentTab) return;

      setActiveTab(nextTab);

      if (nextTab === TAB_RUNS) {
        if (isVersionControlOpen) onCloseVersionControl?.();
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
          onOpenVersionControl?.();
        }
        return;
      }

      toolSidebarState.openToolSidebar();
      if (isVersionControlOpen) onCloseVersionControl?.();
    },
    [
      hasAgentTab,
      isVersionControlOpen,
      mode,
      onCloseVersionControl,
      onExitRunsMode,
      onOpenVersionControl,
      onSelectRuns,
      toolSidebarState,
    ],
  );

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
      if (currentTab === TAB_AGENT && !hasAgentTab) return TAB_VERSIONS;
      if ((currentTab === TAB_RUNS || currentTab === TAB_VERSIONS) && hasAgentTab) return TAB_AGENT;
      return currentTab;
    });
  }, [hasAgentTab, isVersionControlOpen, mode]);

  useAutoOpenVersionControl({
    activeTab,
    hasAgentTab,
    hasAutoOpenedVersionControl,
    hasAutoOpenedVersionControlInMountRef,
    isToolSidebarOpen,
    isVersionControlOpen,
    mode,
    onOpenVersionControl,
    onVersionControlAutoOpened,
    toolSidebarState,
  });

  useCanvasToolSidebarTabEvents(handleTabSelect);

  const tabs = [
    ...(hasAgentTab ? [{ value: TAB_AGENT, label: "Agent", icon: Sparkles }] : []),
    { value: TAB_RUNS, label: "Runs", icon: Rabbit },
    { value: TAB_VERSIONS, label: "Versions", icon: History },
  ] as const;

  if (!isToolSidebarOpen) return null;

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

function defaultToolTab(
  mode: CanvasToolSidebarMode,
  isVersionControlOpen: boolean,
  hasAgentTab: boolean,
): CanvasToolSidebarTab {
  if (mode === "runs") return TAB_RUNS;
  if (isVersionControlOpen) return TAB_VERSIONS;
  if (hasAgentTab) return TAB_AGENT;
  return TAB_VERSIONS;
}

function useAutoOpenVersionControl({
  activeTab,
  hasAgentTab,
  hasAutoOpenedVersionControl,
  hasAutoOpenedVersionControlInMountRef,
  isToolSidebarOpen,
  isVersionControlOpen,
  mode,
  onOpenVersionControl,
  onVersionControlAutoOpened,
  toolSidebarState,
}: {
  activeTab: CanvasToolSidebarTab;
  hasAgentTab: boolean;
  hasAutoOpenedVersionControl?: boolean;
  hasAutoOpenedVersionControlInMountRef: MutableRefObject<boolean>;
  isToolSidebarOpen: boolean;
  isVersionControlOpen?: boolean;
  mode: CanvasToolSidebarMode;
  onOpenVersionControl?: () => void;
  onVersionControlAutoOpened?: () => void;
  toolSidebarState: CanvasToolSidebarState;
}) {
  useEffect(() => {
    if (
      hasAgentTab ||
      !isToolSidebarOpen ||
      isVersionControlOpen ||
      mode === "runs" ||
      activeTab !== TAB_VERSIONS ||
      !onOpenVersionControl ||
      hasAutoOpenedVersionControl ||
      hasAutoOpenedVersionControlInMountRef.current
    ) {
      return;
    }

    hasAutoOpenedVersionControlInMountRef.current = true;
    onVersionControlAutoOpened?.();
    toolSidebarState.openToolSidebar();
    onOpenVersionControl();
  }, [
    activeTab,
    hasAgentTab,
    hasAutoOpenedVersionControl,
    hasAutoOpenedVersionControlInMountRef,
    isToolSidebarOpen,
    isVersionControlOpen,
    mode,
    onOpenVersionControl,
    onVersionControlAutoOpened,
    toolSidebarState,
  ]);
}

function useCanvasToolSidebarTabEvents(handleTabSelect: (tab: CanvasToolSidebarTab) => void) {
  useEffect(() => {
    const onSelectTab = (event: Event) => {
      const tab = canvasToolSidebarTabFromEvent(event);
      if (!tab) return;
      handleTabSelect(tab);
    };

    window.addEventListener(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, onSelectTab);
    return () => window.removeEventListener(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, onSelectTab);
  }, [handleTabSelect]);
}
