import {
  canvasesCommitCanvasStaging,
  canvasesDescribeCanvas,
  canvasesInvokeNodeTriggerHook,
  canvasesListCanvases,
  canvasesPutCanvasStaging,
  factoriesMaterializeFactoryAppTemplate,
  factoriesListFactoryApps,
  type CanvasesCanvasSummary,
  type FactoryApp,
} from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import { canvasKeys } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { appendCanvasToFolderMembership } from "./canvasFolderMembership";
import {
  buildFactoryRunParameters,
  materializeFactoryCanvas,
  materializeFactoryConsole,
  type FactoryAgentRewrite,
  type FactoryDefinition,
} from "./factories";
import type { IntegrationSelections } from "./homeIntegrationStatus";
import type { CanvasFolderData } from "./types";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "./uniqueCanvasName";

const MAX_NAME_RETRY_ATTEMPTS = 20;

export type FactoryCanvasHandle = {
  canvasId: string;
  canvasName: string;
};

export type CreateFactoryCanvasFn = (input: {
  name: string;
  description?: string;
  factoryId?: string;
  method: "ui";
}) => Promise<{
  data?: { canvas?: { metadata?: { id?: string; name?: string } } };
}>;

export type UpdateCanvasFolderMembershipFn = (
  membership: ReturnType<typeof appendCanvasToFolderMembership>,
) => Promise<unknown>;

export async function stageAndCommitFactorySpecs(canvasId: string, canvasYaml: string, consoleYaml: string) {
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

export async function invokeFactoryRun(canvasId: string, definition: FactoryDefinition, startingTaskPrompt: string) {
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

export async function materializeAndCommitFactoryTemplate(args: {
  organizationId: string;
  workspaceFactoryId?: string;
  canvasId: string;
  canvasName: string;
  definition: FactoryDefinition;
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
  agentRewrite?: FactoryAgentRewrite;
}) {
  let canvasYaml: string;
  let consoleYaml: string;
  if (args.workspaceFactoryId) {
    const integrations = Object.entries(args.integrations).flatMap(([type, integration]) =>
      integration ? [{ type, id: integration.id, name: integration.name }] : [],
    );
    const agent = args.agentRewrite
      ? {
          component: args.agentRewrite.component,
          model: args.agentRewrite.model,
          planningModel: args.agentRewrite.planningModel,
          credentialSource: args.agentRewrite.credentials.source,
          credentialIntegrationName:
            args.agentRewrite.credentials.source === "integration" ? args.agentRewrite.credentials.name : undefined,
        }
      : undefined;
    const response = await factoriesMaterializeFactoryAppTemplate(
      withOrganizationHeader({
        organizationId: args.organizationId,
        path: { factoryId: args.workspaceFactoryId, templateId: args.definition.id },
        body: {
          appId: args.canvasId,
          installParams: args.installParams,
          integrations,
          agent,
        },
      }),
    );
    canvasYaml = response.data?.canvasYaml ?? "";
    consoleYaml = response.data?.consoleYaml ?? "";
    if (!canvasYaml || !consoleYaml) {
      throw new Error("Failed to materialize factory app template");
    }
  } else {
    canvasYaml = materializeFactoryCanvas({
      definition: args.definition,
      canvasName: args.canvasName,
      canvasId: args.canvasId,
      installParams: args.installParams,
      integrations: args.integrations,
      agentRewrite: args.agentRewrite,
    });
    consoleYaml = materializeFactoryConsole(args.definition, args.canvasName, args.canvasId);
  }
  await stageAndCommitFactorySpecs(args.canvasId, canvasYaml, consoleYaml);
}

function presentNames(items: { name?: string }[]): string[] {
  return items.map((item) => item.name).filter((name): name is string => Boolean(name));
}

/**
 * Lists the names already taken in the scope the new canvas competes with:
 * the workspace when the canvas belongs to one, the organization otherwise.
 */
async function listExistingCanvasNames(organizationId: string, queryClient: QueryClient, workspaceFactoryId?: string) {
  if (workspaceFactoryId) {
    const cachedApps = queryClient.getQueryData<FactoryApp[]>(factoryAppsKey(organizationId, workspaceFactoryId));
    if (cachedApps) {
      return presentNames(cachedApps);
    }

    const appsResponse = await factoriesListFactoryApps(
      withOrganizationHeader({ organizationId, path: { factoryId: workspaceFactoryId } }),
    );
    return presentNames(appsResponse.data?.apps ?? []);
  }

  const cached = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
  if (cached) {
    return presentNames(cached);
  }

  const response = await canvasesListCanvases(withOrganizationHeader({ organizationId }));
  return presentNames(response.data?.canvases ?? []);
}

async function createCanvasWithUniqueName(args: {
  title: string;
  description?: string;
  workspaceFactoryId?: string;
  existingNames: Set<string>;
  createCanvas: CreateFactoryCanvasFn;
}): Promise<FactoryCanvasHandle> {
  let canvasName = uniqueCanvasName(args.title, args.existingNames);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const result = await args.createCanvas({
        name: canvasName,
        description: args.description,
        factoryId: args.workspaceFactoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create factory canvas");
      }
      return {
        canvasId,
        canvasName: result?.data?.canvas?.metadata?.name?.trim() || canvasName,
      };
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

async function resolveExistingFactoryCanvas(canvasId: string, fallbackName: string): Promise<FactoryCanvasHandle> {
  try {
    const response = await canvasesDescribeCanvas(
      withOrganizationHeader({
        path: { id: canvasId },
      }),
    );
    const name = response.data?.canvas?.metadata?.name?.trim();
    return { canvasId, canvasName: name || fallbackName };
  } catch {
    return { canvasId, canvasName: fallbackName };
  }
}

export async function ensureFactoryCanvas(args: {
  pending: FactoryCanvasHandle | null;
  organizationId: string;
  queryClient: QueryClient;
  definition: FactoryDefinition;
  folder?: CanvasFolderData;
  workspaceFactoryId?: string;
  existingCanvasId?: string;
  createCanvas: CreateFactoryCanvasFn;
  updateCanvasFolderMembership: UpdateCanvasFolderMembershipFn;
}): Promise<FactoryCanvasHandle> {
  if (args.existingCanvasId) {
    if (args.pending?.canvasId === args.existingCanvasId) {
      return args.pending;
    }
    return resolveExistingFactoryCanvas(args.existingCanvasId, args.definition.title);
  }

  if (args.pending) return args.pending;

  const existingNames = new Set(
    await listExistingCanvasNames(args.organizationId, args.queryClient, args.workspaceFactoryId),
  );
  const created = await createCanvasWithUniqueName({
    title: args.definition.title,
    description: args.definition.description,
    workspaceFactoryId: args.workspaceFactoryId,
    existingNames,
    createCanvas: args.createCanvas,
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
