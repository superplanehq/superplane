import { useCallback, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { usePermissions } from "@/contexts/usePermissions";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { canvasKeys, useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { setAgentSuggestions } from "@/lib/agentSuggestionsContext";
import { appPath } from "@/lib/appPaths";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";

import {
  ensureFactoryCanvas,
  invokeFactoryRun,
  materializeAndCommitFactoryTemplate,
  type FactoryCanvasHandle,
} from "./installFactoryCanvas";
import { getFactoryDefinition, type FactoryDefinition } from "./factories";
import type { IntegrationSelections } from "./homeIntegrationStatus";
import type { CanvasFolderData } from "./types";

export interface InstallFactoryInput {
  /** Bundled template id. Defaults to the Software Factory template. */
  factoryId?: string;
  /** When set, create the canvas owned by this workspace factory. */
  workspaceFactoryId?: string;
  /** Resume materialization onto this canvas instead of creating a new one. */
  existingCanvasId?: string;
  /** Called as soon as the canvas exists, before template materialization starts. */
  onCanvasReady?: (canvas: FactoryCanvasHandle) => void | Promise<void>;
  integrations: IntegrationSelections;
  installParams: Record<string, string>;
  startingTaskPrompt: string;
  /**
   * When false, skip navigation and canvas sidebar preference writes.
   * Defaults to true for legacy home install.
   */
  navigateOnComplete?: boolean;
  /**
   * When false, never invoke the starting-task run.
   * Defaults to true when `startingTaskPrompt` is non-empty.
   */
  startInitialRun?: boolean;
}

export type InstallFactoryResult = FactoryCanvasHandle;

interface UseInstallFactoryOptions {
  folder?: CanvasFolderData;
}

async function prepareFactoryCanvas(
  canvas: FactoryCanvasHandle,
  onCanvasReady?: (canvas: FactoryCanvasHandle) => void | Promise<void>,
): Promise<FactoryCanvasHandle> {
  await onCanvasReady?.(canvas);
  return canvas;
}

async function finishFactoryInstall(args: {
  organizationId: string;
  canvasId: string;
  definition: FactoryDefinition;
  startingTaskPrompt: string;
  startInitialRun: boolean;
  navigateOnComplete: boolean;
  queryClient: ReturnType<typeof useQueryClient>;
  navigate: (path: string) => void;
}) {
  const shouldTriggerRun = args.startInitialRun && args.startingTaskPrompt.length > 0;
  if (shouldTriggerRun) {
    await invokeFactoryRun(args.canvasId, args.definition, args.startingTaskPrompt);
    args.queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteRuns(args.canvasId) });
  }

  if (args.definition.agentSuggestions?.length) {
    setAgentSuggestions(args.canvasId, args.definition.agentSuggestions);
  }

  args.queryClient.invalidateQueries({ queryKey: canvasKeys.list(args.organizationId) });

  if (!args.navigateOnComplete) {
    return;
  }

  writeCanvasAgentSidebarOpen(args.canvasId, false);
  writeCanvasRunsSidebarOpen(args.canvasId, shouldTriggerRun);
  localStorage.setItem("canvasSidebarOpen", "false");
  args.navigate(appPath(args.organizationId, args.canvasId, shouldTriggerRun ? "?view=console" : ""));
}

export function useInstallFactory({ folder }: UseInstallFactoryOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { canAct } = usePermissions();
  const createCanvasMutation = useCreateCanvas(organizationId || "");
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId || "");
  const { mutateAsync: createCanvas } = createCanvasMutation;
  const { mutateAsync: updateCanvasFolderMembership } = updateCanvasFolderMembershipMutation;
  const [isInstalling, setIsInstalling] = useState(false);
  const isInstallingRef = useRef(false);
  // Reuse a canvas created on a failed attempt so retry does not spawn duplicates.
  const pendingCanvasRef = useRef<FactoryCanvasHandle | null>(null);

  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");

  const installFactory = useCallback(
    async (input: InstallFactoryInput): Promise<InstallFactoryResult | undefined> => {
      if (!organizationId || isInstallingRef.current) return;
      const reusingCanvas = Boolean(input.existingCanvasId);
      if (!reusingCanvas && !canCreateCanvases) {
        showErrorToast("You don't have permission to create canvases.");
        return;
      }
      if ((folder || reusingCanvas) && !canUpdateCanvases) {
        showErrorToast("You don't have permission to update canvases.");
        return;
      }

      const definition = getFactoryDefinition(input.factoryId);
      const navigateOnComplete = input.navigateOnComplete !== false;
      const startInitialRun = input.startInitialRun !== false;
      isInstallingRef.current = true;
      setIsInstalling(true);

      try {
        const { canvasId, canvasName } = await ensureFactoryCanvas({
          pending: pendingCanvasRef.current,
          organizationId,
          queryClient,
          definition,
          folder,
          workspaceFactoryId: input.workspaceFactoryId,
          existingCanvasId: input.existingCanvasId,
          createCanvas,
          updateCanvasFolderMembership,
        });
        pendingCanvasRef.current = await prepareFactoryCanvas({ canvasId, canvasName }, input.onCanvasReady);

        await materializeAndCommitFactoryTemplate({
          canvasId,
          canvasName,
          definition,
          installParams: input.installParams,
          integrations: input.integrations,
        });

        // Drop the empty canvas cached by create — page load must see the committed template.
        queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });

        await finishFactoryInstall({
          organizationId,
          canvasId,
          definition,
          startingTaskPrompt: input.startingTaskPrompt.trim(),
          startInitialRun,
          navigateOnComplete,
          queryClient,
          navigate,
        });
        if (input.workspaceFactoryId) {
          await queryClient.invalidateQueries({
            queryKey: factoryAppsKey(organizationId, input.workspaceFactoryId),
          });
        }
        pendingCanvasRef.current = null;
        return { canvasId, canvasName };
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to install factory"));
        throw error;
      } finally {
        isInstallingRef.current = false;
        setIsInstalling(false);
      }
    },
    [
      canCreateCanvases,
      canUpdateCanvases,
      createCanvas,
      folder,
      navigate,
      organizationId,
      queryClient,
      updateCanvasFolderMembership,
    ],
  );

  return {
    installFactory,
    isInstalling,
  };
}
