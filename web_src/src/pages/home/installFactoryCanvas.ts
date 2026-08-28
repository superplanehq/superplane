import {
  canvasesCommitCanvasStaging,
  canvasesDescribeCanvas,
  canvasesInvokeNodeTriggerHook,
  canvasesPutCanvasStaging,
} from "@/api-client";
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

export type FactoryCanvasHandle = {
  canvasId: string;
  canvasName: string;
};

export type CreateFactoryCanvasFn = (input: {
  name: string;
  description?: string;
  factoryId?: string;
  uniqueName: boolean;
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
  canvasId: string;
  canvasName: string;
  definition: FactoryDefinition;
  installParams: Record<string, string>;
  integrations: IntegrationSelections;
  agentRewrite?: FactoryAgentRewrite;
}) {
  const canvasYaml = materializeFactoryCanvas({
    definition: args.definition,
    canvasName: args.canvasName,
    canvasId: args.canvasId,
    installParams: args.installParams,
    integrations: args.integrations,
    agentRewrite: args.agentRewrite,
  });
  const consoleYaml = materializeFactoryConsole(args.definition, args.canvasName, args.canvasId);
  await stageAndCommitFactorySpecs(args.canvasId, canvasYaml, consoleYaml);
}

// The server suffixes the title when the organization already holds it. The
// client cannot pick the name itself: canvas names are unique per organization,
// but the canvas list hides the factory-owned canvases that apps install into.
async function createFactoryCanvas(args: {
  title: string;
  description?: string;
  workspaceFactoryId?: string;
  createCanvas: CreateFactoryCanvasFn;
}): Promise<FactoryCanvasHandle> {
  const result = await args.createCanvas({
    name: args.title,
    description: args.description,
    factoryId: args.workspaceFactoryId,
    uniqueName: true,
    method: "ui",
  });

  const canvasId = result?.data?.canvas?.metadata?.id;
  if (!canvasId) {
    throw new Error("Failed to create factory canvas");
  }

  return {
    canvasId,
    canvasName: result?.data?.canvas?.metadata?.name?.trim() || args.title,
  };
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

  const created = await createFactoryCanvas({
    title: args.definition.title,
    description: args.definition.description,
    workspaceFactoryId: args.workspaceFactoryId,
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
