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
  /** When set, skip CreateCanvas and reuse this id (retry after failed stage/commit). */
  pendingCanvasId?: string;
  /** Called once a new empty canvas is created, before stage/commit. */
  onCanvasCreated?: (canvasId: string) => void;
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

function sourceGraphIsEmpty(canvas: CanvasesCanvas | undefined): boolean {
  const nodes = canvas?.spec?.nodes ?? [];
  const edges = canvas?.spec?.edges ?? [];
  return nodes.length === 0 && edges.length === 0;
}

function buildDuplicateCanvasYaml(args: {
  canvasId: string;
  name: string;
  description: string;
  sourceCanvas: CanvasesCanvas | undefined;
}): string {
  return materializeCanvasSpec({
    metadata: {
      id: args.canvasId,
      name: args.name,
      description: args.description,
    },
    spec: {
      nodes: args.sourceCanvas?.spec?.nodes ?? [],
      edges: args.sourceCanvas?.spec?.edges ?? [],
    },
  });
}

async function createDuplicateCanvasShell(deps: DuplicateAutomationCanvasDeps, name: string, description: string) {
  const created = await deps.createCanvas({
    name,
    description,
    factoryId: deps.factoryId,
    method: "ui",
  });
  const canvasId = created.data?.canvas?.metadata?.id;
  if (!canvasId) {
    throw new Error("Failed to create automation canvas");
  }
  return canvasId;
}

async function ensureDuplicateCanvasId(
  deps: DuplicateAutomationCanvasDeps,
  name: string,
  description: string,
): Promise<string> {
  if (deps.pendingCanvasId) {
    return deps.pendingCanvasId;
  }

  const canvasId = await createDuplicateCanvasShell(deps, name, description);
  deps.onCanvasCreated?.(canvasId);
  return canvasId;
}

async function stageAndCommitDuplicateGraph(
  deps: DuplicateAutomationCanvasDeps,
  canvasId: string,
  canvasYaml: string,
) {
  const putCanvasStaging = deps.putCanvasStaging ?? defaultPutCanvasStaging;
  const commitCanvasStaging = deps.commitCanvasStaging ?? defaultCommitCanvasStaging;
  await putCanvasStaging(canvasId, canvasYaml);
  await commitCanvasStaging(canvasId);
}

/**
 * Creates a factory automation clone: empty CreateCanvas, then stage+commit
 * the source live graph as canvas.yaml (same pattern as factory template install).
 * Secrets and run history are not copied.
 */
export async function duplicateAutomationCanvas(deps: DuplicateAutomationCanvasDeps): Promise<string> {
  const sourceCanvasId = deps.app.id?.trim();
  if (!sourceCanvasId) {
    throw new Error("source automation id is required");
  }

  const describeCanvas = deps.describeCanvas ?? defaultDescribeCanvas;
  const sourceCanvas = (await describeCanvas(sourceCanvasId)).data?.canvas;
  const duplicateName = duplicateAutomationName(deps.app.name);
  const description = deps.app.description ?? sourceCanvas?.metadata?.description ?? "";
  const canvasId = await ensureDuplicateCanvasId(deps, duplicateName, description);

  if (sourceGraphIsEmpty(sourceCanvas)) {
    return canvasId;
  }

  await stageAndCommitDuplicateGraph(
    deps,
    canvasId,
    buildDuplicateCanvasYaml({
      canvasId,
      name: duplicateName,
      description,
      sourceCanvas,
    }),
  );
  return canvasId;
}
