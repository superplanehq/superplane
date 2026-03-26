import * as yaml from "js-yaml";

type CanvasNodeLike = Record<string, unknown>;

export function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

export function normalizeForCompare(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => normalizeForCompare(item));
  }

  if (!value || typeof value !== "object") {
    return value;
  }

  const entries = Object.entries(value as Record<string, unknown>).sort(([left], [right]) => left.localeCompare(right));
  const normalized: Record<string, unknown> = {};
  entries.forEach(([key, entryValue]) => {
    normalized[key] = normalizeForCompare(entryValue);
  });
  return normalized;
}

export function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function prettyYAML(value: unknown): string {
  const normalized = normalizeForCompare(value === undefined ? null : value);
  return yaml.dump(normalized, {
    noRefs: true,
    lineWidth: 120,
    sortKeys: true,
  });
}

export function parseNodeYAML(input: string, nodeID: string): { node: CanvasNodeLike | null; error?: string } {
  const trimmed = input.trim();
  if (!trimmed) {
    return { node: null };
  }

  if (trimmed.includes("<<<<<<<") || trimmed.includes("=======") || trimmed.includes(">>>>>>>")) {
    return { node: null, error: "Resolve conflict markers before applying YAML." };
  }

  try {
    const parsed = yaml.load(trimmed);
    if (parsed === null || parsed === undefined) {
      return { node: null };
    }

    if (typeof parsed !== "object" || Array.isArray(parsed)) {
      return { node: null, error: "Final Result must be a YAML object or null." };
    }

    return { node: { ...(parsed as CanvasNodeLike), id: nodeID } };
  } catch {
    return { node: null, error: "Invalid YAML format." };
  }
}

export function renderTopLevelFieldYAMLLines(key: string, value: unknown, hasKey: boolean): string[] {
  if (!hasKey) {
    return [`# ${key} is absent`];
  }

  const dumped = yaml.dump({ [key]: value }, { noRefs: true, lineWidth: 120, sortKeys: false }).trimEnd();
  if (!dumped) {
    return [];
  }
  return dumped.split("\n");
}

export function buildConflictMarkerYAML(
  currentNode: CanvasNodeLike | undefined,
  incomingNode: CanvasNodeLike | undefined,
  currentLabel: string,
  incomingLabel: string,
): string {
  const currentObject = isPlainObject(currentNode) ? (normalizeForCompare(currentNode) as Record<string, unknown>) : {};
  const incomingObject = isPlainObject(incomingNode)
    ? (normalizeForCompare(incomingNode) as Record<string, unknown>)
    : {};

  const keys = Array.from(new Set([...Object.keys(currentObject), ...Object.keys(incomingObject)])).sort(
    (left, right) => left.localeCompare(right),
  );

  const lines: string[] = [];
  keys.forEach((key) => {
    const currentHasKey = Object.prototype.hasOwnProperty.call(currentObject, key);
    const incomingHasKey = Object.prototype.hasOwnProperty.call(incomingObject, key);

    if (!currentHasKey && !incomingHasKey) {
      return;
    }

    const currentValue = currentObject[key];
    const incomingValue = incomingObject[key];
    const valuesAreEqual =
      currentHasKey &&
      incomingHasKey &&
      JSON.stringify(normalizeForCompare(currentValue)) === JSON.stringify(normalizeForCompare(incomingValue));

    if (valuesAreEqual) {
      lines.push(...renderTopLevelFieldYAMLLines(key, currentValue, true));
      return;
    }

    lines.push(`<<<<<<< ${currentLabel}`);
    lines.push(...renderTopLevelFieldYAMLLines(key, currentValue, currentHasKey));
    lines.push("=======");
    lines.push(...renderTopLevelFieldYAMLLines(key, incomingValue, incomingHasKey));
    lines.push(`>>>>>>> ${incomingLabel}`);
  });

  return `${lines.join("\n").trimEnd()}\n`;
}

export function deepMergeObjects(current: unknown, incoming: unknown): unknown {
  if (!isPlainObject(current) || !isPlainObject(incoming)) {
    return incoming;
  }

  const merged: Record<string, unknown> = {};
  const keys = new Set([...Object.keys(current), ...Object.keys(incoming)]);
  keys.forEach((key) => {
    const currentValue = current[key];
    const incomingValue = incoming[key];

    if (incomingValue === undefined) {
      merged[key] = currentValue;
      return;
    }

    if (currentValue === undefined) {
      merged[key] = incomingValue;
      return;
    }

    merged[key] = deepMergeObjects(currentValue, incomingValue);
  });
  return merged;
}

export function mergeConflictBlockLines(currentLines: string[], incomingLines: string[]): string[] {
  if (currentLines.length === 0 && incomingLines.length === 0) {
    return [];
  }

  if (currentLines.length === 0) {
    return incomingLines;
  }

  if (incomingLines.length === 0) {
    return currentLines;
  }

  try {
    const currentParsed = yaml.load(currentLines.join("\n"));
    const incomingParsed = yaml.load(incomingLines.join("\n"));

    if (currentParsed == null && incomingParsed == null) {
      return [];
    }

    if (currentParsed == null) {
      return incomingLines;
    }

    if (incomingParsed == null) {
      return currentLines;
    }

    const merged =
      isPlainObject(currentParsed) && isPlainObject(incomingParsed)
        ? deepMergeObjects(currentParsed, incomingParsed)
        : incomingParsed;

    const mergedYAML = yaml.dump(merged, { noRefs: true, lineWidth: 120, sortKeys: true }).trimEnd();
    if (!mergedYAML) {
      return [];
    }

    return mergedYAML.split("\n");
  } catch {
    return [...currentLines, ...incomingLines];
  }
}

export function upsertNode(nodes: CanvasNodeLike[], nodeID: string, node: CanvasNodeLike | null): CanvasNodeLike[] {
  const index = nodes.findIndex((item) => String(item.id || "") === nodeID);
  if (!node) {
    if (index < 0) {
      return nodes;
    }
    const next = [...nodes];
    next.splice(index, 1);
    return next;
  }

  if (index < 0) {
    return [...nodes, node];
  }

  const next = [...nodes];
  next[index] = node;
  return next;
}

export function getNodeID(node: CanvasNodeLike | undefined): string {
  return String(node?.id || "");
}

export function buildNodeMap(nodes: CanvasNodeLike[]): Map<string, CanvasNodeLike> {
  const result = new Map<string, CanvasNodeLike>();
  nodes.forEach((node) => {
    const id = getNodeID(node);
    if (id) {
      result.set(id, node);
    }
  });
  return result;
}

type CanvasEdgeLike = Record<string, unknown>;

export function pruneEdgesByNodes(edges: CanvasEdgeLike[], nodes: CanvasNodeLike[]): CanvasEdgeLike[] {
  const nodeIDSet = new Set(nodes.map((node) => getNodeID(node)).filter(Boolean));
  return edges.filter((edge) => {
    const sourceID = String(edge.sourceId || "");
    const targetID = String(edge.targetId || "");
    if (!sourceID || !targetID) {
      return false;
    }
    return nodeIDSet.has(sourceID) && nodeIDSet.has(targetID);
  });
}

export function localResolutionLabel(
  currentNode: CanvasNodeLike | undefined,
  incomingNode: CanvasNodeLike | undefined,
  finalNode: CanvasNodeLike | undefined,
): string {
  if (!finalNode) {
    return "excluded";
  }

  if (JSON.stringify(normalizeForCompare(finalNode)) === JSON.stringify(normalizeForCompare(currentNode))) {
    return "current";
  }

  if (JSON.stringify(normalizeForCompare(finalNode)) === JSON.stringify(normalizeForCompare(incomingNode))) {
    return "incoming";
  }

  return "custom";
}
