import { factoriesMaterializeFactoryAppDefaults, type CanvasesCanvas } from "@/api-client";
import {
  requestCanvasAgentSidebarClose,
  requestCanvasAgentSidebarOpen,
  subscribeCanvasAgentSidebarChanged,
} from "@/components/CanvasToolSidebar/canvasAgentSidebarOpenRequest";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { FactoryConfigureActions } from "@/pages/app";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";
import {
  requestBuildingBlocksSidebar,
  subscribeBuildingBlocksSidebarChanged,
} from "@/ui/CanvasPage/buildingBlocksSidebarRequest";
import { useCallback, useEffect, useState, type MutableRefObject } from "react";
import { hasFactoryAppDefaults } from "../lib/factoryAppTemplate";
import { factoryAppConfigurePath, parseFactoryAppNavFrom } from "../lib/factoryPagePaths";
import { setSearchParamFlag } from "../lib/factoryAppSearchParamFlag";

type FactoryAppCanvasEditActionsInput = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  appId: string;
  from: string | null;
  lineId: string | null;
  orderNumber: string | null;
  runId: string | null;
  isConfigure: boolean;
  agentOpen: boolean;
  componentsOpen: boolean;
  canvas?: CanvasesCanvas | null;
  canUpdateCanvas: boolean;
  configureActionsRef: MutableRefObject<FactoryConfigureActions | null>;
  setSearchParams: (updater: (current: URLSearchParams) => URLSearchParams, options?: { replace?: boolean }) => void;
  navigate: (to: string) => void;
};

export function useFactoryAppCanvasEditActions({
  organizationId,
  factoryId,
  factoryKey,
  appId,
  from,
  lineId,
  orderNumber,
  runId,
  isConfigure,
  agentOpen,
  componentsOpen,
  canvas,
  canUpdateCanvas,
  configureActionsRef,
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

  const resetAvailable = isConfigure && canUpdateCanvas && hasFactoryAppDefaults(canvas);
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false);

  const handleOpenResetConfirm = useCallback(() => {
    if (!resetAvailable) return;
    setResetConfirmOpen(true);
  }, [resetAvailable]);

  const handleResetConfirmOpenChange = useCallback((open: boolean) => {
    setResetConfirmOpen(open);
  }, []);

  // The backend owns template generation and preserves the app's live wiring.
  // This only loads the result as a draft. The user still needs to click Save.
  const handleResetToFactoryDefaults = useCallback(async () => {
    setResetConfirmOpen(false);
    if (!resetAvailable) return;

    try {
      const response = await factoriesMaterializeFactoryAppDefaults(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, appId },
          body: {},
        }),
      );
      const parsed = parseCanvasYamlForImport(response.data?.canvasYaml ?? "");
      if (!parsed.ok) {
        showErrorToast(parsed.error);
        return;
      }
      configureActionsRef.current?.applyDraftSpec(parsed.spec);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to reset app"));
    }
  }, [appId, configureActionsRef, factoryId, organizationId, resetAvailable]);

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
  });

  return {
    handleOpenVisualEditor,
    handleAgentPromptOpenChange,
    handleYamlViewOpenChange,
    handleViewYaml,
    handleEditWithLocalAgent,
    handleAgentOpenChange,
    handleComponentsOpenChange,
    resetAvailable,
    resetConfirmOpen,
    handleOpenResetConfirm,
    handleResetConfirmOpenChange,
    handleResetToFactoryDefaults,
  };
}

function useFactoryAppCanvasWorkspaceSync({
  appId,
  isConfigure,
  agentOpen,
  componentsOpen,
  onAgentOpenChange,
  onComponentsOpenChange,
}: {
  appId: string;
  isConfigure: boolean;
  agentOpen: boolean;
  componentsOpen: boolean;
  onAgentOpenChange: (open: boolean) => void;
  onComponentsOpenChange: (open: boolean) => void;
}) {
  useEffect(() => {
    if (!appId) return;
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
  }, [appId, agentOpen, isConfigure]);

  useEffect(() => {
    if (!isConfigure || !appId) return;
    requestBuildingBlocksSidebar(appId, componentsOpen);
  }, [appId, componentsOpen, isConfigure]);

  useEffect(() => {
    if (!isConfigure || !appId) return;
    return subscribeCanvasAgentSidebarChanged((canvasId, open) => {
      if (canvasId !== appId) return;
      onAgentOpenChange(open);
    });
  }, [appId, isConfigure, onAgentOpenChange]);

  useEffect(() => {
    if (!isConfigure || !appId) return;
    return subscribeBuildingBlocksSidebarChanged((canvasId, open) => {
      if (canvasId !== appId) return;
      onComponentsOpenChange(open);
    });
  }, [appId, isConfigure, onComponentsOpenChange]);
}
