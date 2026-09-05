import { canvasesCommitCanvasStaging, canvasesDescribeCanvas, canvasesPutCanvasStaging } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { fetchRepositorySpecFileContent } from "@/pages/app/lib/repository-spec-files";
import { dematerializeCanvasSpec, materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
import { isNotFoundError } from "@/pages/app/workflowPageHelpers";
import {
  rewriteSelfCanvasRefs,
  rematerializeDuplicateConsoleYaml,
  consoleYamlHasContent,
} from "@/pages/factories/pages/duplicateAutomationCanvas";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "./uniqueCanvasName";

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

type CanvasSpec = {
  nodes?: {
    id?: string;
    name?: string;
    component?: string;
    configuration?: Record<string, unknown>;
    metadata?: Record<string, unknown>;
  }[];
  edges?: { sourceId?: string; targetId?: string; channel?: string }[];
};

export type DuplicateCanvasDeps = {
  sourceCanvasId: string;
  sourceName: string;
  sourceDescription?: string;
  createCanvas: (input: { name: string; description?: string }) => Promise<CreateCanvasResult>;
  existingCanvasNames?: Iterable<string>;
  /** When set, skip CreateCanvas and reuse this id (retry after failed stage/commit). */
  pendingCanvasId?: string;
  /** Name already assigned to the pending canvas (keeps yaml metadata in sync on retry). */
  pendingCanvasName?: string;
  /** Called once a new empty canvas is created, before stage/commit. */
  onCanvasCreated?: (canvasId: string, name: string) => void;
  /** Injectable I/O — defaults to real API calls. */
  describeCanvas?: (
    sourceCanvasId: string,
  ) => Promise<{ data?: { canvas?: { metadata?: { description?: string }; spec?: CanvasSpec } } }>;
  fetchSourceSpec?: (sourceCanvasId: string) => Promise<CanvasSpec | null | undefined>;
  fetchConsoleYaml?: (sourceCanvasId: string) => Promise<string | undefined>;
  putCanvasStaging?: (canvasId: string, canvasYaml: string, consoleYaml?: string) => Promise<unknown>;
  commitCanvasStaging?: (canvasId: string) => Promise<unknown>;
};

export async function duplicateCanvas(deps: DuplicateCanvasDeps): Promise<string> {
  const io = resolveIO(deps);
  const { sourceCanvasId, sourceName, sourceDescription } = deps;

  const sourceCanvas = (await io.describeCanvas(sourceCanvasId)).data?.canvas;
  const sourceSpec = await io.fetchSourceSpec(sourceCanvasId);

  const preferredName = `${sourceName} copy`;
  const description = sourceDescription ?? sourceCanvas?.metadata?.description ?? "";
  const { canvasId, name: duplicateName } = await ensureCanvasId(deps, preferredName, description);

  const sourceConsoleYaml = await io.fetchConsoleYaml(sourceCanvasId);

  if (specIsEmpty(sourceSpec) && !sourceConsoleYaml) {
    return canvasId;
  }

  const nodes = sourceSpec?.nodes ?? [];
  const edges = sourceSpec?.edges ?? [];

  const canvasYaml = materializeCanvasSpec({
    metadata: { id: canvasId, name: duplicateName, description },
    spec: {
      nodes: rewriteSelfCanvasRefs(nodes, sourceCanvasId, canvasId, duplicateName),
      edges,
    },
  });
  const consoleYaml = sourceConsoleYaml
    ? rematerializeDuplicateConsoleYaml(sourceConsoleYaml, canvasId, duplicateName)
    : undefined;

  await io.putCanvasStaging(canvasId, canvasYaml, consoleYaml);
  await io.commitCanvasStaging(canvasId);
  return canvasId;
}

function resolveIO(deps: DuplicateCanvasDeps) {
  return {
    describeCanvas: deps.describeCanvas ?? defaultDescribeCanvas,
    fetchSourceSpec: deps.fetchSourceSpec ?? defaultFetchSourceSpec,
    fetchConsoleYaml: deps.fetchConsoleYaml ?? defaultFetchConsoleYaml,
    putCanvasStaging: deps.putCanvasStaging ?? defaultPutCanvasStaging,
    commitCanvasStaging: deps.commitCanvasStaging ?? defaultCommitCanvasStaging,
  };
}

function specIsEmpty(spec: CanvasSpec | null | undefined): boolean {
  return (spec?.nodes ?? []).length === 0 && (spec?.edges ?? []).length === 0;
}

async function ensureCanvasId(
  deps: DuplicateCanvasDeps,
  preferredName: string,
  description: string,
): Promise<{ canvasId: string; name: string }> {
  if (deps.pendingCanvasId) {
    return {
      canvasId: deps.pendingCanvasId,
      name: deps.pendingCanvasName?.trim() || preferredName,
    };
  }
  const created = await createCanvasShell({
    preferredName,
    description,
    createCanvas: deps.createCanvas,
    existingCanvasNames: deps.existingCanvasNames,
  });
  deps.onCanvasCreated?.(created.canvasId, created.name);
  return created;
}

async function defaultDescribeCanvas(sourceCanvasId: string) {
  return canvasesDescribeCanvas(withOrganizationHeader({ path: { id: sourceCanvasId } }));
}

/** Reads the effective canvas spec (staged content, or committed if nothing staged). */
async function defaultFetchSourceSpec(canvasId: string): Promise<CanvasSpec | null | undefined> {
  try {
    const yaml = await fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, undefined, true);
    return yaml?.trim() ? dematerializeCanvasSpec(yaml) : undefined;
  } catch (error) {
    if (isNotFoundError(error) || (error instanceof Error && /404|not found/i.test(error.message))) {
      return undefined;
    }
    throw error;
  }
}

async function defaultFetchConsoleYaml(canvasId: string): Promise<string | undefined> {
  try {
    const raw = await fetchRepositorySpecFileContent(canvasId, CONSOLE_YAML_PATH, undefined, true);
    return consoleYamlHasContent(raw) ? raw : undefined;
  } catch (error) {
    if (isNotFoundError(error) || (error instanceof Error && /404|not found/i.test(error.message))) {
      return undefined;
    }
    throw error;
  }
}

async function defaultPutCanvasStaging(canvasId: string, canvasYaml: string, consoleYaml?: string) {
  const operations = [{ path: CANVAS_YAML_PATH, content: encodeRepositoryFileContent(canvasYaml) }];
  if (consoleYaml) {
    operations.push({ path: CONSOLE_YAML_PATH, content: encodeRepositoryFileContent(consoleYaml) });
  }
  return canvasesPutCanvasStaging(withOrganizationHeader({ path: { canvasId }, body: { operations } }));
}

async function defaultCommitCanvasStaging(canvasId: string) {
  return canvasesCommitCanvasStaging(
    withOrganizationHeader({ path: { canvasId }, body: { commitMessage: "Duplicate canvas" } }),
  );
}

async function createCanvasShell(params: {
  preferredName: string;
  description: string;
  createCanvas: (input: { name: string; description?: string }) => Promise<CreateCanvasResult>;
  existingCanvasNames?: Iterable<string>;
}): Promise<{ canvasId: string; name: string }> {
  const taken = new Set([...(params.existingCanvasNames ?? [])].map((n) => n.trim()).filter(Boolean));
  let canvasName = uniqueCanvasName(params.preferredName, taken);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const created = await params.createCanvas({
        name: canvasName,
        description: params.description,
      });
      const canvasId = created.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create canvas");
      }
      return { canvasId, name: canvasName };
    } catch (error) {
      if (!isCanvasNameAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(canvasName);
      canvasName = uniqueCanvasName(params.preferredName, taken);
    }
  }

  throw new Error("Failed to create canvas");
}
