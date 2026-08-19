import { requestCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/canvasAgentSidebarOpenRequest";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { useCallback } from "react";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../lib/factoryPagePaths";
import { setSearchParamFlag } from "../lib/factoryAppSearchParamFlag";

type FactoryAppCanvasEditActionsInput = {
  organizationId: string;
  factoryKey: string;
  appId: string;
  from: string | null;
  lineId: string | null;
  orderNumber: string | null;
  setSearchParams: (updater: (current: URLSearchParams) => URLSearchParams, options?: { replace?: boolean }) => void;
  navigate: (to: string) => void;
};

export function useFactoryAppCanvasEditActions({
  organizationId,
  factoryKey,
  appId,
  from,
  lineId,
  orderNumber,
  setSearchParams,
  navigate,
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
      }),
    );
  }, [appId, factoryKey, from, lineId, navigate, orderNumber, organizationId]);

  const handleAskAgent = useCallback(() => {
    if (!appId) return;
    writeCanvasAgentSidebarOpen(appId, true);
    requestCanvasAgentSidebarOpen(appId);
  }, [appId]);

  const handleAgentPromptOpenChange = useCallback(
    (open: boolean) => {
      handleSearchParamFlag("agentPrompt", open);
    },
    [handleSearchParamFlag],
  );

  const handleYamlViewOpenChange = useCallback(
    (open: boolean) => {
      handleSearchParamFlag("yaml", open);
    },
    [handleSearchParamFlag],
  );

  return {
    handleOpenVisualEditor,
    handleAskAgent,
    handleAgentPromptOpenChange,
    handleYamlViewOpenChange,
  };
}
