import { useCallback, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { canvasesCommitCanvasStaging, canvasesInvokeNodeTriggerHook, canvasesPutCanvasStaging } from "@/api-client";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { usePermissions } from "@/contexts/usePermissions";
import { canvasKeys, useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { setAgentSuggestions } from "@/lib/agentSuggestionsContext";
import { appPath } from "@/lib/appPaths";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";

import { appendCanvasToFolderMembership } from "./canvasFolderMembership";
import { createCanvasWithUniqueName, listExistingCanvasNames } from "./createCanvasWithUniqueName";
import {
  buildFactoryRunParameters,
  getFactoryDefinition,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  type FactoryDefinition,
} from "./factories";
import type { IntegrationSelections } from "./homeIntegrationStatus";
import type { CanvasFolderData } from "./types";

export interface InstallFactoryInput {
  factoryId?: string;
  integrations: IntegrationSelections;
  installParams: Record<string, string>;
  startingTaskPrompt: string;
}

interface UseInstallFactoryOptions {
  folder?: CanvasFolderData;
}

async function stageAndCommitFactorySpecs(canvasId: string, canvasYaml: string, consoleYaml: string) {
  await canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: {
        operations: [
          { path: CANVAS_YAML_PATH, content: encodeRepositoryFileContent(canvasYaml) },
          { path: CONSOLE_YAML_PATH, content: encodeRepositoryFileContent(consoleYaml) },
        ],
      },
    }),
  );
  await canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: { commitMessage: "Install factory template" },
    }),
  );
}

async function invokeFactoryRun(canvasId: string, definition: FactoryDefinition, startingTaskPrompt: string) {
  await canvasesInvokeNodeTriggerHook(
    withOrganizationHeader({
      path: {
        canvasId,
        nodeId: definition.run.nodeId,
        hookName: definition.run.hookName,
      },
      body: {
        parameters: buildFactoryRunParameters(definition, startingTaskPrompt),
      },
    }),
  );
}

async function materializeAndCommitFactoryTemplate(args: {
  canvasId: string;
  canvasName: string;
  definition: FactoryDefinition;
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
}) {
  const canvasYaml = materializeFactoryCanvas({
    definition: args.definition,
    canvasName: args.canvasName,
    canvasId: args.canvasId,
    installParams: args.installParams,
    integrations: args.integrations,
  });
  const consoleYaml = materializeFactoryConsole(args.definition, args.canvasName, args.canvasId);
  await stageAndCommitFactorySpecs(args.canvasId, canvasYaml, consoleYaml);
}

async function ensureFactoryCanvas(args: {
  pending: { canvasId: string; canvasName: string } | null;
  organizationId: string;
  queryClient: QueryClient;
  definition: FactoryDefinition;
  folder?: CanvasFolderData;
  createCanvas: (input: { name: string; description?: string; method: "ui" }) => Promise<{
    data?: { canvas?: { metadata?: { id?: string } } };
  }>;
  updateCanvasFolderMembership: (membership: ReturnType<typeof appendCanvasToFolderMembership>) => Promise<unknown>;
}): Promise<{ canvasId: string; canvasName: string }> {
  if (args.pending) return args.pending;

  const existingNames = new Set(await listExistingCanvasNames(args.organizationId, args.queryClient));
  const created = await createCanvasWithUniqueName({
    title: args.definition.title,
    existingNames,
    failureMessage: "Failed to create factory canvas",
    createCanvas: async (name) => {
      const result = await args.createCanvas({ name, description: args.definition.description, method: "ui" });
      const canvasId = result?.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create factory canvas");
      }
      return { canvasId };
    },
  });

  if (args.folder) {
    try {
      await args.updateCanvasFolderMembership(appendCanvasToFolderMembership(args.folder, created.canvasId));
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "App created, but failed to add it to folder"));
    }
  }

  return created;
}

async function finishFactoryInstall(args: {
  organizationId: string;
  canvasId: string;
  definition: FactoryDefinition;
  startingTaskPrompt: string;
  queryClient: QueryClient;
  navigate: (path: string) => void;
}) {
  const shouldTriggerRun = args.startingTaskPrompt.length > 0;
  if (shouldTriggerRun) {
    await invokeFactoryRun(args.canvasId, args.definition, args.startingTaskPrompt);
    args.queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteRuns(args.canvasId) });
  }

  if (args.definition.agentSuggestions?.length) {
    setAgentSuggestions(args.canvasId, args.definition.agentSuggestions);
  }
  writeCanvasAgentSidebarOpen(args.canvasId, false);
  writeCanvasRunsSidebarOpen(args.canvasId, shouldTriggerRun);
  localStorage.setItem("canvasSidebarOpen", "false");
  args.queryClient.invalidateQueries({ queryKey: canvasKeys.list(args.organizationId) });
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
  const pendingCanvasRef = useRef<{ canvasId: string; canvasName: string } | null>(null);

  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");

  const installFactory = useCallback(
    async (input: InstallFactoryInput) => {
      if (!organizationId || isInstallingRef.current) return;
      if (!canCreateCanvases) {
        showErrorToast("You don't have permission to create canvases.");
        return;
      }
      if (folder && !canUpdateCanvases) {
        showErrorToast("You don't have permission to update canvases.");
        return;
      }

      const definition = getFactoryDefinition(input.factoryId);
      isInstallingRef.current = true;
      setIsInstalling(true);

      try {
        const { canvasId, canvasName } = await ensureFactoryCanvas({
          pending: pendingCanvasRef.current,
          organizationId,
          queryClient,
          definition,
          folder,
          createCanvas,
          updateCanvasFolderMembership,
        });
        pendingCanvasRef.current = { canvasId, canvasName };

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
          queryClient,
          navigate,
        });
        pendingCanvasRef.current = null;
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
