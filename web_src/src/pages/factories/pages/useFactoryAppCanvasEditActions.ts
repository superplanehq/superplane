import {
  requestCanvasAgentSidebarClose,
  requestCanvasAgentSidebarOpen,
  subscribeCanvasAgentSidebarChanged,
} from "@/components/CanvasToolSidebar/canvasAgentSidebarOpenRequest";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import {
  requestBuildingBlocksSidebar,
  subscribeBuildingBlocksSidebarChanged,
} from "@/ui/CanvasPage/buildingBlocksSidebarRequest";
import { useCallback, useEffect } from "react";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../lib/factoryPagePaths";
import { setSearchParamFlag } from "../lib/factoryAppSearchParamFlag";

type FactoryAppCanvasEditActionsInput = {
  organizationId: string;
  factoryKey: string;
  appId: string;
  from: string | null;
  lineId: string | null;
  orderNumber: string | null;
  runId: string | null;
  isConfigure: boolean;
  agentOpen: boolean;
  componentsOpen: boolean;
  setSearchParams: (updater: (current: URLSearchParams) => URLSearchParams, options?: { replace?: boolean }) => void;
  navigate: (to: string) => void;
  /** Storybook-only. Live factory canvas ignores agent/components URL flags. */
  enabled?: boolean;
};

export function useFactoryAppCanvasEditActions({
  organizationId,
  factoryKey,
  appId,
  from,
  lineId,
  orderNumber,
  runId,
  isConfigure,
  agentOpen,
  componentsOpen,
  setSearchParams,
  navigate,
  enabled = false,
}: FactoryAppCanvasEditActionsInput) {
  const handleSearchParamFlag = useCallback(
    (key: string, open: boolean) => {
      setSearchParams((current) => setSearchParamFlag(current, key, open), { replace: true });
    },
    [setSearchParams],
  );

  const handleOpenVisualEditor = useCallback(() => {
    navigate(
      factoryAppConfigurePath(organizationId, factoryKey, appId, {
        from: parseFactoryAppNavFrom(from),
        lineId: lineId ?? undefined,
        orderNumber: orderNumber ?? undefined,
        runId: runId ?? undefined,
      }),
    );
  }, [appId, factoryKey, from, lineId, navigate, orderNumber, organizationId, runId]);

  const handleAgentPromptOpenChange = useCallback(
    (open: boolean) => {
      setSearchParams(
        (current) => {
          const next = setSearchParamFlag(current, "agentPrompt", open);
          if (!open) {
            return next;
          }
          return setSearchParamFlag(next, "yaml", false);
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const handleYamlViewOpenChange = useCallback(
    (open: boolean) => {
      setSearchParams(
        (current) => {
          const next = setSearchParamFlag(current, "yaml", open);
          if (!open) {
            return next;
          }
          return setSearchParamFlag(next, "agentPrompt", false);
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const handleViewYaml = useCallback(() => {
    handleYamlViewOpenChange(true);
  }, [handleYamlViewOpenChange]);

  const handleEditWithLocalAgent = useCallback(() => {
    handleAgentPromptOpenChange(true);
  }, [handleAgentPromptOpenChange]);

  const handleAgentOpenChange = useCallback(
    (open: boolean) => {
      handleSearchParamFlag("agent", open);
    },
    [handleSearchParamFlag],
  );

  const handleComponentsOpenChange = useCallback(
    (open: boolean) => {
      handleSearchParamFlag("blocks", open);
    },
    [handleSearchParamFlag],
  );

  useFactoryAppCanvasWorkspaceSync({
    appId,
    isConfigure,
    agentOpen,
    componentsOpen,
    onAgentOpenChange: handleAgentOpenChange,
    onComponentsOpenChange: handleComponentsOpenChange,
    enabled,
  });

  return {
    handleOpenVisualEditor,
    handleAgentPromptOpenChange,
    handleYamlViewOpenChange,
    handleViewYaml,
    handleEditWithLocalAgent,
    handleAgentOpenChange,
    handleComponentsOpenChange,
  };
}

function useFactoryAppCanvasWorkspaceSync({
  appId,
  isConfigure,
  agentOpen,
  componentsOpen,
  onAgentOpenChange,
  onComponentsOpenChange,
  enabled,
}: {
  appId: string;
  isConfigure: boolean;
  agentOpen: boolean;
  componentsOpen: boolean;
  onAgentOpenChange: (open: boolean) => void;
  onComponentsOpenChange: (open: boolean) => void;
  enabled: boolean;
}) {
  useEffect(() => {
    if (!enabled || !appId) return;
    if (!isConfigure) {
      writeCanvasAgentSidebarOpen(appId, false);
      requestCanvasAgentSidebarClose(appId);
      return;
    }
    writeCanvasAgentSidebarOpen(appId, agentOpen);
    if (agentOpen) {
      requestCanvasAgentSidebarOpen(appId);
      return;
    }
    requestCanvasAgentSidebarClose(appId);
  }, [appId, agentOpen, enabled, isConfigure]);

  useEffect(() => {
    if (!enabled || !isConfigure || !appId) return;
    requestBuildingBlocksSidebar(appId, componentsOpen);
  }, [appId, componentsOpen, enabled, isConfigure]);

  useEffect(() => {
    if (!enabled || !isConfigure || !appId) return;
    return subscribeCanvasAgentSidebarChanged((canvasId, open) => {
      if (canvasId !== appId) return;
      onAgentOpenChange(open);
    });
  }, [appId, enabled, isConfigure, onAgentOpenChange]);

  useEffect(() => {
    if (!enabled || !isConfigure || !appId) return;
    return subscribeBuildingBlocksSidebarChanged((canvasId, open) => {
      if (canvasId !== appId) return;
      onComponentsOpenChange(open);
    });
  }, [appId, enabled, isConfigure, onComponentsOpenChange]);
}
