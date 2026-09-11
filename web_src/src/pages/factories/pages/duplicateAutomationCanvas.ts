import {
  canvasesCommitCanvasStaging,
  canvasesDescribeCanvas,
  canvasesPutCanvasStaging,
  type CanvasesCanvas,
  type FactoryApp,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { isNotFoundError } from "@/lib/errors";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { fetchRepositorySpecFileContent } from "@/pages/app/lib/repository-spec-files";
import { dematerializeConsoleSpec, materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "@/pages/home/uniqueCanvasName";
import * as yaml from "js-yaml";

import { duplicateAutomationName } from "./automationCardActions";

const MAX_NAME_RETRY_ATTEMPTS = 20;

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
  /** Names already taken in the workspace (used to pick "X copy", "X copy (2)", …). */
  existingCanvasNames?: Iterable<string>;
  /** When set, skip CreateCanvas and reuse this id (retry after failed stage/commit). */
  pendingCanvasId?: string;
  /** Name already assigned to the pending canvas (keeps yaml metadata in sync on retry). */
  pendingCanvasName?: string;
  /** Called once a new empty canvas is created, before stage/commit. */
  onCanvasCreated?: (canvasId: string, canvasName: string) => void;
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
    return consoleYamlHasContent(consoleYaml) ? consoleYaml : undefined;
  } catch (error) {
    if (isNotFoundError(error)) {
      return undefined;
    }
    throw error;
  }
}

/** True when console.yaml has at least one panel or layout item (not the empty default shell). */
export function consoleYamlHasContent(consoleYaml: string): boolean {
  if (!consoleYaml.trim()) {
    return false;
  }
  const parsed = dematerializeConsoleSpec(consoleYaml);
  if (!parsed) {
    return false;
  }
  return parsed.panels.length > 0 || parsed.layout.length > 0;
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

export function rematerializeDuplicateConsoleYaml(consoleYaml: string, canvasId: string, canvasName: string): string {
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
      nodes: rewriteSelfCanvasRefs(args.sourceCanvas?.spec?.nodes, args.sourceCanvasId, args.canvasId, args.name),
      edges: args.sourceCanvas?.spec?.edges ?? [],
    },
  });
}

async function createDuplicateCanvasShell(
  deps: DuplicateAutomationCanvasDeps,
  preferredName: string,
  description: string,
): Promise<{ canvasId: string; name: string }> {
  const taken = new Set(
    [...(deps.existingCanvasNames ?? [])].map((name) => name.trim()).filter((name): name is string => Boolean(name)),
  );
  let canvasName = uniqueCanvasName(preferredName, taken);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const created = await deps.createCanvas({
        name: canvasName,
        description,
        factoryId: deps.factoryId,
        method: "ui",
      });
      const canvasId = created.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create automation canvas");
      }
      return { canvasId, name: canvasName };
    } catch (error) {
      if (!isCanvasNameAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(canvasName);
      canvasName = uniqueCanvasName(preferredName, taken);
    }
  }

  throw new Error("Failed to create automation canvas");
}

async function ensureDuplicateCanvasId(
  deps: DuplicateAutomationCanvasDeps,
  preferredName: string,
  description: string,
): Promise<{ canvasId: string; name: string }> {
  if (deps.pendingCanvasId) {
    return {
      canvasId: deps.pendingCanvasId,
      name: deps.pendingCanvasName?.trim() || preferredName,
    };
  }

  const created = await createDuplicateCanvasShell(deps, preferredName, description);
  deps.onCanvasCreated?.(created.canvasId, created.name);
  return created;
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
  const preferredName = duplicateAutomationName(deps.app.name);
  const description = deps.app.description ?? sourceCanvas?.metadata?.description ?? "";
  const { canvasId, name: duplicateName } = await ensureDuplicateCanvasId(deps, preferredName, description);

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
