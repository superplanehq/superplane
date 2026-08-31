import type { CanvasesCanvas } from "@/api-client";
import {
  requestCanvasAgentSidebarClose,
  requestCanvasAgentSidebarOpen,
  subscribeCanvasAgentSidebarChanged,
} from "@/components/CanvasToolSidebar/canvasAgentSidebarOpenRequest";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { showErrorToast } from "@/lib/toast";
import type { FactoryConfigureActions } from "@/pages/app";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";
import { materializeFactoryCanvas } from "@/pages/home/factories";
import {
  requestBuildingBlocksSidebar,
  subscribeBuildingBlocksSidebarChanged,
} from "@/ui/CanvasPage/buildingBlocksSidebarRequest";
import { useCallback, useEffect, useMemo, useState, type MutableRefObject } from "react";
import { deriveFactoryAppResetWiring, resolveFactoryAppTemplate } from "../lib/factoryAppTemplate";
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
  canvas?: CanvasesCanvas | null;
  canUpdateCanvas: boolean;
  configureActionsRef: MutableRefObject<FactoryConfigureActions | null>;
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

  const resetTemplate = useMemo(() => resolveFactoryAppTemplate(canvas), [canvas]);
  const resetAvailable = isConfigure && canUpdateCanvas && Boolean(resetTemplate);
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false);

  const handleOpenResetConfirm = useCallback(() => {
    if (!resetAvailable) return;
    setResetConfirmOpen(true);
  }, [resetAvailable]);

  const handleResetConfirmOpenChange = useCallback((open: boolean) => {
    setResetConfirmOpen(open);
  }, []);

  // Rematerializes the app's bundled template with its own real repo/agent
  // wiring and loads it as an unsaved configure draft. The user still needs
  // to click Save — this never persists or calls a gRPC action.
  const handleResetToFactoryDefaults = useCallback(() => {
    setResetConfirmOpen(false);
    if (!resetAvailable || !resetTemplate || !canvas) return;

    const wiring = deriveFactoryAppResetWiring(canvas);
    const yamlText = materializeFactoryCanvas({
      definition: resetTemplate,
      canvasName: canvas.metadata?.name || resetTemplate.title,
      canvasId: appId,
      installParams: wiring.installParams,
      integrations: wiring.integrations,
      agentRewrite: wiring.agentRewrite,
    });

    const parsed = parseCanvasYamlForImport(yamlText);
    if (!parsed.ok) {
      showErrorToast(parsed.error);
      return;
    }

    configureActionsRef.current?.applyDraftSpec(parsed.spec);
  }, [appId, canvas, configureActionsRef, resetAvailable, resetTemplate]);

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
