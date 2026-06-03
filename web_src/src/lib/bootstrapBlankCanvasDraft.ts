import type { CanvasesCanvas, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { canvasesCreateDraftBranch } from "@/api-client";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH, putStaging } from "@/lib/canvas-staging";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { writeLastDraftBranch } from "@/hooks/useActiveDraftBranch";
import { fetchCanvasRepositoryFileContent } from "@/pages/workflowv2/lib/canvas-repository-files";
import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "@/pages/workflowv2/lib/canvas-yaml-staging";

const DEFAULT_PLACEHOLDER_POSITION = { x: 400, y: 300 };

function generateNodeId(blockName: string, nodeName: string): string {
  const randomChars = Math.random().toString(36).substring(2, 8);
  const sanitizedBlock = blockName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  return `${sanitizedBlock}-${sanitizedName}-${randomChars}`;
}

function buildPlaceholderNode(position: { x: number; y: number }): ComponentsNode {
  return {
    id: generateNodeId("component", "node"),
    name: "New Component",
    type: "TYPE_ACTION",
    configuration: {},
    metadata: {},
    position: {
      x: Math.round(position.x),
      y: Math.round(position.y),
    },
  };
}

/**
 * Creates the first draft branch for a blank canvas and writes the starter placeholder
 * node into IndexedDB staging before navigation so the canvas page opens in edit mode
 * with a stable "uncommitted changes" state.
 */
export async function bootstrapBlankCanvasDraft(
  canvas: CanvasesCanvas,
  options?: { position?: { x: number; y: number } },
): Promise<string> {
  const canvasId = canvas.metadata?.id;
  if (!canvasId) {
    throw new Error("Canvas id is required to bootstrap a draft branch");
  }

  const response = await canvasesCreateDraftBranch(
    withOrganizationHeader({
      path: { canvasId },
      body: {},
    }),
  );

  const branch = response.data?.branch;
  const branchName = branch?.branchName;
  const tipSha = branch?.tipSha ?? "";
  if (!branchName) {
    throw new Error("Draft branch was not returned from the API");
  }

  const [canvasYaml, consoleYaml] = await Promise.all([
    fetchCanvasRepositoryFileContent(canvasId, CANVAS_YAML_PATH, branchName).catch(() => ""),
    fetchCanvasRepositoryFileContent(canvasId, CONSOLE_YAML_PATH, branchName).catch(() => ""),
  ]);

  const position = options?.position ?? DEFAULT_PLACEHOLDER_POSITION;
  const baseSpec = parseCanvasYamlToSpec(canvasYaml) ?? canvas.spec ?? { nodes: [], edges: [] };
  const placeholderNode = buildPlaceholderNode(position);

  const workflowWithPlaceholder: CanvasesCanvas = {
    ...canvas,
    spec: {
      nodes: [...(baseSpec.nodes ?? []), placeholderNode],
      edges: baseSpec.edges ?? [],
      changeManagement: baseSpec.changeManagement,
    },
  };

  const stagedFiles: Record<string, string> = {
    [CANVAS_YAML_PATH]: buildCanvasYamlFromWorkflow(workflowWithPlaceholder),
  };
  if (consoleYaml) {
    stagedFiles[CONSOLE_YAML_PATH] = consoleYaml;
  }

  await putStaging({
    canvasId,
    branch: branchName,
    baseHeadSha: tipSha,
    files: stagedFiles,
    updatedAt: Date.now(),
  });

  writeLastDraftBranch(canvasId, branchName);
  return branchName;
}
