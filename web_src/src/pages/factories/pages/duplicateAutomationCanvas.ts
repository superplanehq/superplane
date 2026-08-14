import {
  canvasesCommitCanvasStaging,
  canvasesDescribeCanvas,
  canvasesPutCanvasStaging,
  type CanvasesCanvas,
  type FactoryApp,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";

import { duplicateAutomationName } from "./automationCardActions";

type CreateCanvasResult = {
  data?: {
    canvas?: {
      metadata?: {
        id?: string;
      };
    };
  };
};

type CreateCanvasInput = {
  name: string;
  description?: string;
  factoryId?: string;
  method?: "ui" | "cli" | "yaml_import" | "template";
};

export type DuplicateAutomationCanvasDeps = {
  factoryId: string;
  app: FactoryApp;
  createCanvas: (input: CreateCanvasInput) => Promise<CreateCanvasResult>;
  describeCanvas?: (sourceCanvasId: string) => Promise<{ data?: { canvas?: CanvasesCanvas } }>;
  putCanvasStaging?: (canvasId: string, canvasYaml: string) => Promise<unknown>;
  commitCanvasStaging?: (canvasId: string) => Promise<unknown>;
};

async function defaultDescribeCanvas(sourceCanvasId: string) {
  return canvasesDescribeCanvas(
    withOrganizationHeader({
      path: { id: sourceCanvasId },
    }),
  );
}

async function defaultPutCanvasStaging(canvasId: string, canvasYaml: string) {
  return canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: {
        operations: [
          {
            path: CANVAS_YAML_PATH,
            content: encodeRepositoryFileContent(canvasYaml),
          },
        ],
      },
    }),
  );
}

async function defaultCommitCanvasStaging(canvasId: string) {
  return canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: { commitMessage: "Duplicate automation" },
    }),
  );
}

/**
 * Creates a factory automation clone: empty CreateCanvas, then stage+commit
 * the source live graph as canvas.yaml (same pattern as factory template install).
 */
export async function duplicateAutomationCanvas(deps: DuplicateAutomationCanvasDeps): Promise<string> {
  const sourceCanvasId = deps.app.id?.trim();
  if (!sourceCanvasId) {
    throw new Error("source automation id is required");
  }

  const describeCanvas = deps.describeCanvas ?? defaultDescribeCanvas;
  const putCanvasStaging = deps.putCanvasStaging ?? defaultPutCanvasStaging;
  const commitCanvasStaging = deps.commitCanvasStaging ?? defaultCommitCanvasStaging;

  const sourceResponse = await describeCanvas(sourceCanvasId);
  const sourceCanvas = sourceResponse.data?.canvas;
  const duplicateName = duplicateAutomationName(deps.app.name);
  const description = deps.app.description ?? sourceCanvas?.metadata?.description ?? "";

  const created = await deps.createCanvas({
    name: duplicateName,
    description,
    factoryId: deps.factoryId,
    method: "ui",
  });
  const canvasId = created.data?.canvas?.metadata?.id;
  if (!canvasId) {
    throw new Error("Failed to create automation canvas");
  }

  const nodes = sourceCanvas?.spec?.nodes ?? [];
  const edges = sourceCanvas?.spec?.edges ?? [];
  if (nodes.length === 0 && edges.length === 0) {
    return canvasId;
  }

  const canvasYaml = materializeCanvasSpec({
    metadata: {
      id: canvasId,
      name: duplicateName,
      description,
    },
    spec: {
      nodes,
      edges,
    },
  });

  await putCanvasStaging(canvasId, canvasYaml);
  await commitCanvasStaging(canvasId);
  return canvasId;
}
