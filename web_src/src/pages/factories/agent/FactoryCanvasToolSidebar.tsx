import { useEffect } from "react";
import {
  CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT,
  canvasToolSidebarTabFromEvent,
} from "@/components/CanvasToolSidebar/events";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { FactoryAgentTabPanel } from "./FactoryAgentTabPanel";
import { FactoryAgentSidebarShell } from "./FactoryAgentSidebarShell";

export interface FactoryCanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
}

export function FactoryCanvasToolSidebar({ toolSidebarState }: FactoryCanvasToolSidebarProps) {
  useFactoryAgentSidebarTabEvents(toolSidebarState);

  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.canvasId || !toolSidebarState.isAgentEnabled) {
    return null;
  }

  if (!toolSidebarState.isToolSidebarOpen) {
    return null;
  }

  return (
    <FactoryAgentSidebarShell>
      <div className="m-0 flex min-h-0 flex-1 flex-col overflow-hidden" role="tabpanel">
        <FactoryAgentTabPanel toolSidebarState={toolSidebarState} />
      </div>
    </FactoryAgentSidebarShell>
  );
}

function useFactoryAgentSidebarTabEvents(toolSidebarState: CanvasToolSidebarState) {
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
