import {
  canvasesCommitCanvasStaging,
  canvasesDescribeCanvas,
  canvasesPutCanvasStaging,
  type CanvasesCanvas,
  type FactoryApp,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { fetchRepositorySpecFileContent } from "@/pages/app/lib/repository-spec-files";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
import { isNotFoundError } from "@/pages/app/workflowPageHelpers";
import * as yaml from "js-yaml";

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

type CanvasNodes = NonNullable<NonNullable<CanvasesCanvas["spec"]>["nodes"]>;

export type DuplicateAutomationCanvasDeps = {
  factoryId: string;
  app: FactoryApp;
  createCanvas: (input: CreateCanvasInput) => Promise<CreateCanvasResult>;
  /** When set, skip CreateCanvas and reuse this id (retry after failed stage/commit). */
  pendingCanvasId?: string;
  /** Called once a new empty canvas is created, before stage/commit. */
  onCanvasCreated?: (canvasId: string) => void;
  describeCanvas?: (sourceCanvasId: string) => Promise<{ data?: { canvas?: CanvasesCanvas } }>;
  fetchConsoleYaml?: (sourceCanvasId: string) => Promise<string | undefined>;
  putCanvasStaging?: (canvasId: string, canvasYaml: string, consoleYaml?: string) => Promise<unknown>;
  commitCanvasStaging?: (canvasId: string) => Promise<unknown>;
};

async function defaultDescribeCanvas(sourceCanvasId: string) {
  return canvasesDescribeCanvas(
    withOrganizationHeader({
      path: { id: sourceCanvasId },
    }),
  );
}

async function defaultFetchConsoleYaml(sourceCanvasId: string): Promise<string | undefined> {
  try {
    const consoleYaml = await fetchRepositorySpecFileContent(sourceCanvasId, CONSOLE_YAML_PATH);
    return consoleYaml.trim() ? consoleYaml : undefined;
  } catch (error) {
    if (isNotFoundError(error) || (error instanceof Error && /404|not found/i.test(error.message))) {
      return undefined;
    }
    throw error;
  }
}

async function defaultPutCanvasStaging(canvasId: string, canvasYaml: string, consoleYaml?: string) {
  const operations = [
    {
      path: CANVAS_YAML_PATH,
      content: encodeRepositoryFileContent(canvasYaml),
    },
  ];
  if (consoleYaml) {
    operations.push({
      path: CONSOLE_YAML_PATH,
      content: encodeRepositoryFileContent(consoleYaml),
    });
  }

  return canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: { operations },
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

/** Point self-runApp refs at the clone instead of the source canvas. */
export function rewriteSelfCanvasRefs(
  nodes: CanvasNodes | undefined,
  sourceCanvasId: string,
  newCanvasId: string,
  newCanvasName: string,
): CanvasNodes {
  return (nodes ?? []).map((node) => {
    const configuration = node.configuration;
    if (!configuration || typeof configuration !== "object" || Array.isArray(configuration)) {
      return node;
    }

    const configRecord = configuration as Record<string, unknown>;
    if (configRecord.app !== sourceCanvasId) {
      return node;
    }

    const metadata =
      node.metadata && typeof node.metadata === "object" && !Array.isArray(node.metadata)
        ? { ...(node.metadata as Record<string, unknown>) }
        : {};
    const existingAppMeta =
      metadata.app && typeof metadata.app === "object" && !Array.isArray(metadata.app)
        ? { ...(metadata.app as Record<string, unknown>) }
        : {};

    return {
      ...node,
      configuration: {
        ...configRecord,
        app: newCanvasId,
      },
      metadata: {
        ...metadata,
        app: {
          ...existingAppMeta,
          id: newCanvasId,
          name: newCanvasName,
        },
      },
    };
  });
}

export function rematerializeDuplicateConsoleYaml(
  consoleYaml: string,
  canvasId: string,
  canvasName: string,
): string {
  const doc = yaml.load(consoleYaml) as { metadata?: Record<string, unknown>; [key: string]: unknown } | null;
  if (!doc || typeof doc !== "object" || Array.isArray(doc)) {
    return consoleYaml;
  }

  doc.metadata = {
    ...(doc.metadata ?? {}),
    name: canvasName,
    canvasId,
  };
  return yaml.dump(doc, { lineWidth: -1, noRefs: true });
}

function buildDuplicateCanvasYaml(args: {
  canvasId: string;
  name: string;
  description: string;
  sourceCanvasId: string;
  sourceCanvas: CanvasesCanvas | undefined;
}): string {
  return materializeCanvasSpec({
    metadata: {
      id: args.canvasId,
      name: args.name,
      description: args.description,
    },
    spec: {
      nodes: rewriteSelfCanvasRefs(
        args.sourceCanvas?.spec?.nodes,
        args.sourceCanvasId,
        args.canvasId,
        args.name,
      ),
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

async function stageAndCommitDuplicateSpecs(
  deps: DuplicateAutomationCanvasDeps,
  canvasId: string,
  canvasYaml: string,
  consoleYaml?: string,
) {
  const putCanvasStaging = deps.putCanvasStaging ?? defaultPutCanvasStaging;
  const commitCanvasStaging = deps.commitCanvasStaging ?? defaultCommitCanvasStaging;
  await putCanvasStaging(canvasId, canvasYaml, consoleYaml);
  await commitCanvasStaging(canvasId);
}

/**
 * Creates a factory automation clone: empty CreateCanvas, then stage+commit
 * the source live graph as canvas.yaml (and console.yaml when present).
 * Secrets and run history are not copied.
 */
export async function duplicateAutomationCanvas(deps: DuplicateAutomationCanvasDeps): Promise<string> {
  const sourceCanvasId = deps.app.id?.trim();
  if (!sourceCanvasId) {
    throw new Error("source automation id is required");
  }

  const describeCanvas = deps.describeCanvas ?? defaultDescribeCanvas;
  const fetchConsoleYaml = deps.fetchConsoleYaml ?? defaultFetchConsoleYaml;
  const sourceCanvas = (await describeCanvas(sourceCanvasId)).data?.canvas;
  const duplicateName = duplicateAutomationName(deps.app.name);
  const description = deps.app.description ?? sourceCanvas?.metadata?.description ?? "";
  const canvasId = await ensureDuplicateCanvasId(deps, duplicateName, description);

  const sourceConsoleYaml = await fetchConsoleYaml(sourceCanvasId);
  const consoleYaml = sourceConsoleYaml
    ? rematerializeDuplicateConsoleYaml(sourceConsoleYaml, canvasId, duplicateName)
    : undefined;

  if (sourceGraphIsEmpty(sourceCanvas) && !consoleYaml) {
    return canvasId;
  }

  await stageAndCommitDuplicateSpecs(
    deps,
    canvasId,
    buildDuplicateCanvasYaml({
      canvasId,
      name: duplicateName,
      description,
      sourceCanvasId,
      sourceCanvas,
    }),
    consoleYaml,
  );
  return canvasId;
}
