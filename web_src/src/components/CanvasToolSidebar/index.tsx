import { useEffect } from "react";
import { AgentTabPanel } from "./AgentTabPanel";
import { CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, canvasToolSidebarTabFromEvent } from "./events";
import { SidebarShell } from "./SidebarShell";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

export interface CanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
}

export function CanvasToolSidebar({ toolSidebarState }: CanvasToolSidebarProps) {
  useCanvasToolSidebarTabEvents(toolSidebarState);

  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.canvasId || !toolSidebarState.isAgentEnabled) {
    return null;
  }

  if (!toolSidebarState.isToolSidebarOpen) {
    return null;
  }

  return (
    <SidebarShell>
      <div className="m-0 flex min-h-0 flex-1 flex-col overflow-hidden" role="tabpanel">
        <AgentTabPanel toolSidebarState={toolSidebarState} />
      </div>
    </SidebarShell>
  );
}

function useCanvasToolSidebarTabEvents(toolSidebarState: CanvasToolSidebarState) {
  useEffect(() => {
    const onSelectTab = (event: Event) => {
      const tab = canvasToolSidebarTabFromEvent(event);
      if (tab !== "agent") return;
      toolSidebarState.openToolSidebar();
    };

    window.addEventListener(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, onSelectTab);
    return () => window.removeEventListener(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, onSelectTab);
  }, [toolSidebarState]);
}
