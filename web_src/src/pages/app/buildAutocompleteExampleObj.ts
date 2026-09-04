import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { CanvasesCanvas, CanvasesCanvasEvent, CanvasesCanvasNodeExecution } from "@/api-client";
import type { ActionsAction, TriggersTrigger } from "@/api-client";

export type AutocompleteAppExample = {
  id: string;
  name: string;
  description?: string;
};

export function buildAutocompleteAppExample(
  canvas: CanvasesCanvas | null | undefined,
): AutocompleteAppExample | undefined {
  const id = canvas?.metadata?.id;
  if (!id) {
    return undefined;
  }

  return {
    id,
    name: canvas?.metadata?.name ?? "",
    description: canvas?.metadata?.description ?? "",
  };
}

/**
 * Where a node's authoring payload came from. Kept out of the expression
 * environment so it can drive UI labels without adding fake keys to the `$`
 * payload.
 */
export type AutocompletePayloadSource =
  | { kind: "execution"; executionId?: string; observedAt?: string }
  | { kind: "event"; eventId?: string; observedAt?: string }
  | { kind: "example" };

/**
 * Result of building the autocomplete example object for a node.
 *
 * `context` is the expression environment (the `$` payload). `sourcesByNodeId`
 * records, per upstream chain node, where that node's payload came from so the
 * UI can label it.
 */
export type AutocompleteExampleResult = {
  context: Record<string, unknown> | null;
  sourcesByNodeId: Record<string, AutocompletePayloadSource>;
};

export type AutocompleteExampleContext = {
  canvasNodes: ComponentsNode[];
  canvasNodesById: Map<string, ComponentsNode>;
  incomingNodeIdsByTargetId: Map<string, string[]>;
  // Raw execution/event maps from the store. Authoring reads these directly;
  // the canvas keeps separate visibility-filtered maps for overlays.
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  allComponentsByName: Map<string | undefined, ActionsAction>;
  allTriggersByName: Map<string | undefined, TriggersTrigger>;
  app?: AutocompleteAppExample;
};

// Representative run id used purely to preview the shape of run() in the editor;
// the real id is only known at runtime.
const EXAMPLE_RUN_ID = "f47ac10b-58cc-4372-a567-0e02b2c3d479";

function currentAppPath(): { origin: string; appPath: string } | null {
  if (typeof window === "undefined") {
    return null;
  }

  const { origin, pathname } = window.location;
  const appPath = pathname.match(/^\/[^/]+\/apps\/[^/]+/)?.[0] ?? pathname;
  return { origin, appPath };
}

// buildAppExample mirrors the server's app() payload so the autocomplete can
// surface app().id / app().name / app().description / app().url.
function buildAppExample(app?: AutocompleteAppExample): Record<string, unknown> {
  const location = currentAppPath();
  const id = app?.id || location?.appPath.split("/").pop() || "";
  const url = location ? `${location.origin}${location.appPath}` : "";

  return {
    id,
    name: app?.name ?? "Example App",
    description: app?.description ?? "",
    url,
  };
}

// buildRunExample mirrors the server's run() payload so the autocomplete can
// surface run().id / run().url / run().started_at and show a representative preview.
// The example url is derived from the current app page location
// (`/{org}/apps/{appId}`), which matches the real run link format.
function buildRunExample(): Record<string, unknown> {
  const location = currentAppPath();
  const url = location ? `${location.origin}${location.appPath}?run=${EXAMPLE_RUN_ID}` : "";

  return {
    id: EXAMPLE_RUN_ID,
    url,
    started_at: new Date().toISOString(),
  };
}

// Representative work-order shape for order() autocomplete / preview.
// Real values come from the factory execution at runtime; this is only a stub.
const EXAMPLE_ORDER_ID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890";
const EXAMPLE_FACTORY_ID = "b2c3d4e5-f6a7-8901-bcde-f12345678901";
const EXAMPLE_ORDER_NUMBER = 12;

// Task permalinks are workspace-scoped
// (`/{org}/workspaces/{workspaceKey}/work-order/{number}`), so the example is
// only meaningful on a workspace app page, where order() also resolves.
function exampleOrderUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }

  const { origin, pathname } = window.location;
  const workspacePath = pathname.match(/^\/[^/]+\/workspaces\/[^/]+/)?.[0];
  return workspacePath ? `${origin}${workspacePath}/work-order/${EXAMPLE_ORDER_NUMBER}` : "";
}

function buildOrderExample(): Record<string, unknown> {
  return {
    id: EXAMPLE_ORDER_ID,
    title: "Ship feature",
    description: "Implement and open PR",
    factory_id: EXAMPLE_FACTORY_ID,
    state: "open",
    result: "",
    repository: "acme/service",
    repository_url: "https://github.com/acme/service.git",
    default_branch: "main",
    url: exampleOrderUrl(),
    source: {
      issue: { number: 42, title: "Fix login" },
    },
    artifacts: [
      {
        id: "c3d4e5f6-a7b8-9012-cdef-123456789012",
        type: "pr",
        data: { url: "https://github.com/org/repo/pull/7", number: 7 },
      },
    ],
    comments: [
      {
        id: "d4e5f6a7-b8c9-0123-defa-234567890123",
        body: "Looks good, merging.",
        author: { kind: "user", user_id: "e5f6a7b8-c9d0-1234-efab-345678901234" },
        created_at: "2024-01-01T00:00:00Z",
      },
    ],
  };
}

function buildWorkspaceExample(): Record<string, unknown> {
  return {
    id: EXAMPLE_FACTORY_ID,
    key: "SP",
    name: "Example workspace",
    repository: "acme/service",
    backlog_repository: "acme/service",
    default_branch: "main",
  };
}

function collectChainNodeIds(
  nodeId: string,
  currentNode: ComponentsNode | undefined,
  incomingNodeIdsByTargetId: Map<string, string[]>,
): Set<string> {
  const chainNodeIds = new Set<string>();
  if (currentNode?.type === "TYPE_TRIGGER") {
    chainNodeIds.add(nodeId);
  }

  const stack = [...(incomingNodeIdsByTargetId.get(nodeId) || [])];
  while (stack.length > 0) {
    const nextId = stack.pop();
    if (!nextId || chainNodeIds.has(nextId)) continue;
    chainNodeIds.add(nextId);
    incomingNodeIdsByTargetId.get(nextId)?.forEach((sourceId) => stack.push(sourceId));
  }

  return chainNodeIds;
}

// Outputs are keyed by channel. Return the first item of the first non-empty
// array, which is the delivered payload.
function firstUsableOutput(execution: CanvasesCanvasNodeExecution): unknown | undefined {
  const outputs = execution.outputs;
  if (!outputs) {
    return undefined;
  }
  const found = Object.values(outputs).find((output) => Array.isArray(output) && output.length > 0) as
    | unknown[]
    | undefined;
  return found?.[0];
}

// Executions are newest first (created_at DESC; websocket updates prepend at
// index 0). Check state and output in the same pass so a newer execution without
// output does not hide an older one that still has a usable payload.
function selectUsableExecutionWithOutput(
  executions: CanvasesCanvasNodeExecution[] | undefined,
): { execution: CanvasesCanvasNodeExecution; output: unknown } | undefined {
  for (const execution of executions ?? []) {
    if (execution.state !== "STATE_FINISHED") continue;
    if (execution.resultReason === "RESULT_REASON_ERROR") continue;
    const output = firstUsableOutput(execution);
    if (output !== undefined) {
      return { execution, output };
    }
  }
  return undefined;
}

// Clone outputs and examples the same way. Spreading an array into `{...}`
// would turn it into a numeric-keyed object.
function clonePayload(value: unknown): unknown {
  if (Array.isArray(value)) return [...value];
  if (value && typeof value === "object") return { ...(value as Record<string, unknown>) };
  return value;
}

function isUsableExample(value: unknown): value is Record<string, unknown> | unknown[] {
  return (
    typeof value === "object" &&
    value !== null &&
    (Array.isArray(value) ? value.length > 0 : Object.keys(value).length > 0)
  );
}

// Attach config to a payload unless the payload already has one. configData
// must come from the same source as payload: the selected execution for a real
// payload, or the node's own config for an example fallback.
function attachConfig(payload: unknown, configData: unknown): void {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return;
  if ("config" in (payload as Record<string, unknown>)) return;
  if (configData && typeof configData === "object" && Object.keys(configData).length > 0) {
    (payload as Record<string, unknown>).config = configData;
  }
}

type ChainNodeExamplesResult = {
  exampleObj: Record<string, unknown>;
  sourcesByNodeId: Record<string, AutocompletePayloadSource>;
};

function buildChainNodeExamples(
  chainNodeIds: Set<string>,
  context: AutocompleteExampleContext,
  nodeNamesById: Record<string, string>,
  nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }>,
): ChainNodeExamplesResult {
  const exampleObj: Record<string, unknown> = {};
  const sourcesByNodeId: Record<string, AutocompletePayloadSource> = {};

  chainNodeIds.forEach((chainNodeId) => {
    const chainNode = context.canvasNodesById.get(chainNodeId);
    if (!chainNode) return;

    const nodeName = (chainNode.name || "").trim();
    if (nodeName) {
      nodeNamesById[chainNodeId] = nodeName;
    }

    if (chainNode.type === "TYPE_TRIGGER") {
      const triggerMetadata = context.allTriggersByName.get(chainNode.component);
      nodeMetadata[chainNodeId] = {
        name: nodeName || undefined,
        componentType: triggerMetadata?.label || "Trigger",
        description: triggerMetadata?.description,
      };

      const latestEvent = context.nodeEventsMap[chainNodeId]?.[0];
      if (latestEvent?.data && typeof latestEvent.data === "object") {
        exampleObj[chainNodeId] = clonePayload(latestEvent.data);
        sourcesByNodeId[chainNodeId] = {
          kind: "event",
          eventId: latestEvent.id,
          observedAt: latestEvent.createdAt,
        };
        return;
      }

      const exampleData = triggerMetadata?.exampleData;
      if (exampleData && isUsableExample(exampleData)) {
        exampleObj[chainNodeId] = clonePayload(exampleData);
        sourcesByNodeId[chainNodeId] = { kind: "example" };
      }
      return;
    }

    const componentMetadata = context.allComponentsByName.get(chainNode.component);
    nodeMetadata[chainNodeId] = {
      name: nodeName || undefined,
      componentType: componentMetadata?.label || "Component",
      description: componentMetadata?.description,
    };

    const selected = selectUsableExecutionWithOutput(context.nodeExecutionsMap[chainNodeId]);
    if (selected) {
      const payload = clonePayload(selected.output);
      exampleObj[chainNodeId] = payload;
      sourcesByNodeId[chainNodeId] = {
        kind: "execution",
        executionId: selected.execution.id,
        observedAt: selected.execution.createdAt,
      };
      // Config from the same execution that produced the payload.
      attachConfig(payload, selected.execution.configuration);
      return;
    }

    const exampleOutput = componentMetadata?.exampleOutput;
    if (exampleOutput && isUsableExample(exampleOutput)) {
      const payload = clonePayload(exampleOutput);
      exampleObj[chainNodeId] = payload;
      sourcesByNodeId[chainNodeId] = { kind: "example" };
      // Use the node's own config here, not the unrelated execution's.
      attachConfig(payload, chainNode.configuration);
    }
  });

  return { exampleObj, sourcesByNodeId };
}

function buildPreviousByDepth(
  nodeId: string,
  exampleObj: Record<string, unknown>,
  incomingNodeIdsByTargetId: Map<string, string[]>,
): Record<string, unknown> {
  const previousByDepth: Record<string, unknown> = {};
  let frontier = [nodeId];
  const visited = new Set<string>([nodeId]);
  let depth = 0;

  while (frontier.length > 0) {
    const next: string[] = [];
    frontier.forEach((current) => {
      (incomingNodeIdsByTargetId.get(current) || []).forEach((sourceId) => {
        if (visited.has(sourceId)) return;
        visited.add(sourceId);
        next.push(sourceId);
      });
    });

    if (next.length === 0) {
      break;
    }

    depth += 1;
    const firstAtDepth = next[0];
    if (firstAtDepth && exampleObj[firstAtDepth]) {
      previousByDepth[String(depth)] = exampleObj[firstAtDepth];
    }

    frontier = next;
  }

  return previousByDepth;
}

type BuildNamedExampleObjInput = {
  currentNode: ComponentsNode | undefined;
  chainNodeIds: Set<string>;
  exampleObj: Record<string, unknown>;
  nodeNamesById: Record<string, string>;
  nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }>;
  previousByDepth: Record<string, unknown>;
  canvasNodes: ComponentsNode[];
  incomingNodeIdsByTargetId: Map<string, string[]>;
  appExample: Record<string, unknown>;
  runExample: Record<string, unknown>;
  orderExample: Record<string, unknown>;
  workspaceExample: Record<string, unknown>;
};

function buildNamedExampleObj({
  currentNode,
  chainNodeIds,
  exampleObj,
  nodeNamesById,
  nodeMetadata,
  previousByDepth,
  canvasNodes,
  incomingNodeIdsByTargetId,
  appExample,
  runExample,
  orderExample,
  workspaceExample,
}: BuildNamedExampleObjInput): Record<string, unknown> | null {
  const rootNodeId = canvasNodes.find((node) => {
    if (!node.id || !chainNodeIds.has(node.id)) return false;
    return !(incomingNodeIdsByTargetId.get(node.id) || []).some((sourceId) => chainNodeIds.has(sourceId));
  })?.id;

  if (rootNodeId && exampleObj[rootNodeId]) {
    exampleObj.__root = exampleObj[rootNodeId];
  }

  if (Object.keys(previousByDepth).length > 0) {
    exampleObj.__previousByDepth = previousByDepth;
  }

  const nameToNodeId = new Map<string, string>();
  for (const [nId, nodeName] of Object.entries(nodeNamesById)) {
    if (!nodeName || nodeName === "__nodeNames") {
      continue;
    }

    if (!nameToNodeId.has(nodeName)) {
      nameToNodeId.set(nodeName, nId);
    }
  }

  const namedExampleObj: Record<string, unknown> = {};
  for (const [nodeName, namedNodeId] of nameToNodeId.entries()) {
    if (nodeName === namedNodeId || namedExampleObj[nodeName] !== undefined) {
      continue;
    }

    const value = exampleObj[namedNodeId];
    if (value === undefined) {
      continue;
    }

    namedExampleObj[nodeName] = value;
  }

  if (exampleObj.__root) {
    namedExampleObj.__root = exampleObj.__root;
  }

  if (exampleObj.__previousByDepth) {
    namedExampleObj.__previousByDepth = exampleObj.__previousByDepth;
  }

  namedExampleObj.__app = appExample;
  namedExampleObj.__run = runExample;
  namedExampleObj.__order = orderExample;
  namedExampleObj.__workspace = workspaceExample;

  const currentNodeName = currentNode?.name?.trim();
  const currentNodeId = currentNode?.id;
  if (currentNodeName) {
    delete namedExampleObj[currentNodeName];
  }
  if (currentNodeId) {
    delete nodeMetadata[currentNodeId];
  }

  if (Object.keys(namedExampleObj).length === 0) {
    return null;
  }

  if (Object.keys(nodeMetadata).length > 0) {
    namedExampleObj.__nodeNames = nodeMetadata;
    Object.entries(nodeMetadata).forEach(([, metadata]) => {
      const value = namedExampleObj[metadata.name ?? ""];
      if (value && typeof value === "object" && !Array.isArray(value) && metadata.name) {
        (value as Record<string, unknown>).__nodeName = metadata.name;
      }
    });
  }

  return namedExampleObj;
}

export function buildAutocompleteExampleResult(
  nodeId: string,
  context: AutocompleteExampleContext,
): AutocompleteExampleResult {
  const currentNode = context.canvasNodesById.get(nodeId);
  const chainNodeIds = collectChainNodeIds(nodeId, currentNode, context.incomingNodeIdsByTargetId);
  if (chainNodeIds.size === 0) {
    return { context: null, sourcesByNodeId: {} };
  }

  const nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }> = {};
  const nodeNamesById: Record<string, string> = {};
  const { exampleObj, sourcesByNodeId } = buildChainNodeExamples(chainNodeIds, context, nodeNamesById, nodeMetadata);
  const previousByDepth = buildPreviousByDepth(nodeId, exampleObj, context.incomingNodeIdsByTargetId);

  const namedExampleObj = buildNamedExampleObj({
    currentNode,
    chainNodeIds,
    exampleObj,
    nodeNamesById,
    nodeMetadata,
    previousByDepth,
    appExample: buildAppExample(context.app),
    runExample: buildRunExample(),
    orderExample: buildOrderExample(),
    workspaceExample: buildWorkspaceExample(),
    canvasNodes: context.canvasNodes,
    incomingNodeIdsByTargetId: context.incomingNodeIdsByTargetId,
  });

  if (!namedExampleObj) {
    return { context: null, sourcesByNodeId: {} };
  }

  // The authored node never contributes a payload; drop it from provenance.
  const sourcesForNamed = { ...sourcesByNodeId };
  if (currentNode?.id) {
    delete sourcesForNamed[currentNode.id];
  }

  return { context: namedExampleObj, sourcesByNodeId: sourcesForNamed };
}

/**
 * Backward-compatible wrapper that returns only the expression environment.
 * Prefer `buildAutocompleteExampleResult` when you also need payload provenance
 * for source labels.
 */
export function buildAutocompleteExampleObj(
  nodeId: string,
  context: AutocompleteExampleContext,
): Record<string, unknown> | null {
  return buildAutocompleteExampleResult(nodeId, context).context;
}

export type PayloadSourceSummary = {
  label: string;
  isExample: boolean;
};

/**
 * Reduces per-node provenance into a single label for the editor.
 *
 * - all executions        -> "Latest real payload"
 * - all events            -> "Latest trigger event"
 * - all examples          -> "Example payload"
 * - mixed real + example  -> "Includes example data"
 * - otherwise             -> "Latest real data"
 *
 * The execution label says "real payload" rather than "successful" because the
 * selector keeps any finished, non-error execution with output, including
 * cancelled or resolved-error ones.
 */
export function summarizePayloadSources(
  sources: Record<string, AutocompletePayloadSource>,
): PayloadSourceSummary | null {
  const values = Object.values(sources);
  if (values.length === 0) {
    return null;
  }

  const kinds = new Set(values.map((source) => source.kind));
  const onlyExamples = kinds.size === 1 && kinds.has("example");
  if (onlyExamples) {
    return { label: "Example payload", isExample: true };
  }
  if (kinds.has("example")) {
    return { label: "Includes example data", isExample: true };
  }
  if (kinds.size === 1 && kinds.has("execution")) {
    return { label: "Latest real payload", isExample: false };
  }
  if (kinds.size === 1 && kinds.has("event")) {
    return { label: "Latest trigger event", isExample: false };
  }
  return { label: "Latest real data", isExample: false };
}
