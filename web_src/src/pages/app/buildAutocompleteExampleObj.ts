import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution } from "@/api-client";
import type { ActionsAction, TriggersTrigger } from "@/api-client";

export type AutocompleteExampleContext = {
  canvasNodes: ComponentsNode[];
  canvasNodesById: Map<string, ComponentsNode>;
  incomingNodeIdsByTargetId: Map<string, string[]>;
  visibleNodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  visibleNodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  allComponentsByName: Map<string | undefined, ActionsAction>;
  allTriggersByName: Map<string | undefined, TriggersTrigger>;
};

// Representative run id used purely to preview the shape of run() in the editor;
// the real id is only known at runtime.
const EXAMPLE_RUN_ID = "f47ac10b-58cc-4372-a567-0e02b2c3d479";

// buildRunExample mirrors the server's run() payload so the autocomplete can
// surface run().id / run().url / run().started_at and show a representative preview.
// The example url is derived from the current app page location
// (`/{org}/apps/{appId}`), which matches the real run link format.
function buildRunExample(): Record<string, unknown> {
  let url = "";
  if (typeof window !== "undefined") {
    const { origin, pathname } = window.location;
    const appPath = pathname.match(/^\/[^/]+\/apps\/[^/]+/)?.[0] ?? pathname;
    url = `${origin}${appPath}?run=${EXAMPLE_RUN_ID}`;
  }

  return {
    id: EXAMPLE_RUN_ID,
    url,
    started_at: new Date().toISOString(),
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

function buildChainNodeExamples(
  chainNodeIds: Set<string>,
  context: AutocompleteExampleContext,
  nodeNamesById: Record<string, string>,
  nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }>,
): Record<string, unknown> {
  const exampleObj: Record<string, unknown> = {};

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

      const latestEvent = context.visibleNodeEventsMap[chainNodeId]?.[0];
      if (latestEvent?.data) {
        exampleObj[chainNodeId] = { ...(latestEvent.data || {}) } as Record<string, unknown>;
      }
      if (exampleObj[chainNodeId]) {
        return;
      }

      const exampleData = triggerMetadata?.exampleData;
      if (exampleData && typeof exampleData === "object") {
        exampleObj[chainNodeId] = Array.isArray(exampleData)
          ? [...exampleData]
          : ({ ...exampleData } as Record<string, unknown>);
      }
      return;
    }

    const componentMetadata = context.allComponentsByName.get(chainNode.component);
    nodeMetadata[chainNodeId] = {
      name: nodeName || undefined,
      componentType: componentMetadata?.label || "Component",
      description: componentMetadata?.description,
    };

    const latestExecution = context.visibleNodeExecutionsMap[chainNodeId]?.find(
      (execution) => execution.state === "STATE_FINISHED" && execution.resultReason !== "RESULT_REASON_ERROR",
    );
    if (!latestExecution?.outputs) {
      const exampleOutput = componentMetadata?.exampleOutput;
      if (exampleOutput && typeof exampleOutput === "object") {
        exampleObj[chainNodeId] = Array.isArray(exampleOutput)
          ? [...exampleOutput]
          : ({ ...exampleOutput } as Record<string, unknown>);
      }
      return;
    }

    const outputData: unknown[] = Object.values(latestExecution.outputs)?.find((output) => {
      return Array.isArray(output) && output.length > 0;
    }) as unknown[];

    if (outputData?.length > 0) {
      exampleObj[chainNodeId] = { ...(outputData?.[0] || {}) } as Record<string, unknown>;
      return;
    }

    const exampleOutput = componentMetadata?.exampleOutput;
    if (exampleOutput && typeof exampleOutput === "object" && Object.keys(exampleOutput).length > 0) {
      exampleObj[chainNodeId] = { ...exampleOutput } as Record<string, unknown>;
    }
  });

  return exampleObj;
}

function injectConfigIntoExamples(
  chainNodeIds: Set<string>,
  exampleObj: Record<string, unknown>,
  context: AutocompleteExampleContext,
): void {
  chainNodeIds.forEach((chainNodeId) => {
    const chainNode = context.canvasNodesById.get(chainNodeId);
    if (!chainNode || chainNode.type !== "TYPE_ACTION") return;

    const obj = exampleObj[chainNodeId];
    if (!obj || typeof obj !== "object" || Array.isArray(obj)) return;

    const latestExecution = context.visibleNodeExecutionsMap[chainNodeId]?.find(
      (execution) => execution.state === "STATE_FINISHED" && execution.resultReason !== "RESULT_REASON_ERROR",
    );
    if ("config" in (obj as Record<string, unknown>)) return;

    const configData = latestExecution?.configuration || chainNode.configuration;
    if (configData && typeof configData === "object" && Object.keys(configData).length > 0) {
      (obj as Record<string, unknown>).config = configData;
    }
  });
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
  runExample: Record<string, unknown>;
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
  runExample,
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

  namedExampleObj.__run = runExample;

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

export function buildAutocompleteExampleObj(
  nodeId: string,
  context: AutocompleteExampleContext,
): Record<string, unknown> | null {
  const currentNode = context.canvasNodesById.get(nodeId);
  const chainNodeIds = collectChainNodeIds(nodeId, currentNode, context.incomingNodeIdsByTargetId);
  if (chainNodeIds.size === 0) {
    return null;
  }

  const nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }> = {};
  const nodeNamesById: Record<string, string> = {};
  const exampleObj = buildChainNodeExamples(chainNodeIds, context, nodeNamesById, nodeMetadata);
  injectConfigIntoExamples(chainNodeIds, exampleObj, context);
  const previousByDepth = buildPreviousByDepth(nodeId, exampleObj, context.incomingNodeIdsByTargetId);

  return buildNamedExampleObj({
    currentNode,
    chainNodeIds,
    exampleObj,
    nodeNamesById,
    nodeMetadata,
    previousByDepth,
    runExample: buildRunExample(),
    canvasNodes: context.canvasNodes,
    incomingNodeIdsByTargetId: context.incomingNodeIdsByTargetId,
  });
}
