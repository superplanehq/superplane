import { useCallback, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  canvasesCommitCanvasStaging,
  canvasesInvokeNodeTriggerHook,
  canvasesListCanvases,
  canvasesPutCanvasStaging,
  type CanvasesCanvasSummary,
} from "@/api-client";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { usePermissions } from "@/contexts/usePermissions";
import { canvasKeys, useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { appPath } from "@/lib/appPaths";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";

import { appendCanvasToFolderMembership } from "./canvasFolderMembership";
import {
  buildFactoryRunParameters,
  getFactoryDefinition,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  type FactoryDefinition,
} from "./factories";
import type { IntegrationSelections } from "./homeIntegrationStatus";
import type { CanvasFolderData } from "./types";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "./uniqueCanvasName";

const MAX_NAME_RETRY_ATTEMPTS = 20;

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
    installParams: args.installParams,
    integrations: args.integrations,
  });
  const consoleYaml = materializeFactoryConsole(args.definition, args.canvasName);
  await stageAndCommitFactorySpecs(args.canvasId, canvasYaml, consoleYaml);
}

async function listExistingCanvasNames(organizationId: string, queryClient: QueryClient) {
  const cached = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
  if (cached) {
    return cached.map((canvas) => canvas.name).filter((name): name is string => Boolean(name));
  }

  const response = await canvasesListCanvases(withOrganizationHeader({ organizationId }));
  return (response.data?.canvases ?? []).map((canvas) => canvas.name).filter((name): name is string => Boolean(name));
}

async function createCanvasWithUniqueName(args: {
  title: string;
  description?: string;
  existingNames: Set<string>;
  createCanvas: (input: { name: string; description?: string; method: "ui" }) => Promise<{
    data?: { canvas?: { metadata?: { id?: string } } };
  }>;
}): Promise<{ canvasId: string; canvasName: string }> {
  let canvasName = uniqueCanvasName(args.title, args.existingNames);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const result = await args.createCanvas({
        name: canvasName,
        description: args.description,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create factory canvas");
      }
      return { canvasId, canvasName };
    } catch (error) {
      if (!isCanvasNameAlreadyExistsError(error)) {
        throw error;
      }
      args.existingNames.add(canvasName);
      canvasName = uniqueCanvasName(args.title, args.existingNames);
    }
  }

  throw new Error("Failed to create factory canvas");
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
        const existingNames = new Set(await listExistingCanvasNames(organizationId, queryClient));
        const { canvasId, canvasName } = await createCanvasWithUniqueName({
          title: definition.title,
          description: definition.description,
          existingNames,
          createCanvas,
        });

        if (folder) {
          try {
            await updateCanvasFolderMembership(appendCanvasToFolderMembership(folder, canvasId));
          } catch (error) {
            showErrorToast(getApiErrorMessage(error, "App created, but failed to add it to folder"));
          }
        }

        await materializeAndCommitFactoryTemplate({
          canvasId,
          canvasName,
          definition,
          installParams: input.installParams,
          integrations: input.integrations,
        });

        // Drop the empty canvas cached by create — page load must see the committed template.
        queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });

        const startingTaskPrompt = input.startingTaskPrompt.trim();
        const shouldTriggerRun = startingTaskPrompt.length > 0;
        if (shouldTriggerRun) {
          await invokeFactoryRun(canvasId, definition, startingTaskPrompt);
          queryClient.invalidateQueries({ queryKey: canvasKeys.infiniteRuns(canvasId) });
        }

        writeCanvasAgentSidebarOpen(canvasId, false);
        writeCanvasRunsSidebarOpen(canvasId, shouldTriggerRun);
        localStorage.setItem("canvasSidebarOpen", "false");
        queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
        navigate(appPath(organizationId, canvasId, shouldTriggerRun ? "?view=console" : ""));
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
