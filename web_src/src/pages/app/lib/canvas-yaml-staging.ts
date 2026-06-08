import * as yaml from "js-yaml";
import type { CanvasesCanvas } from "@/api-client";
import type {
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";

type ParsedCanvasYaml = {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    description?: string;
  };
  spec?: {
    nodes?: ComponentsNode[];
    edges?: ComponentsEdge[];
    changeManagement?: CanvasesCanvas["spec"] extends infer Spec
      ? Spec extends { changeManagement?: infer ChangeManagement }
        ? ChangeManagement
        : never
      : never;
  };
};

export function parseCanvasYamlToSpec(text: string): CanvasesCanvas["spec"] | null {
  const trimmed = text.trim();
  if (!trimmed) {
    return null;
  }

  let parsed: ParsedCanvasYaml;
  try {
    parsed = yaml.load(trimmed) as ParsedCanvasYaml;
  } catch {
    return null;
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return null;
  }

  if (parsed.kind && parsed.kind !== "Canvas") {
    return null;
  }

  if (!parsed.spec || typeof parsed.spec !== "object") {
    return null;
  }

  if (parsed.spec.nodes !== undefined && !Array.isArray(parsed.spec.nodes)) {
    return null;
  }

  return {
    nodes: (parsed.spec.nodes ?? []).map(normalizeCanvasYamlNode),
    edges: (parsed.spec.edges ?? []).map(normalizeCanvasYamlEdge),
    changeManagement: parsed.spec.changeManagement,
  };
}

function normalizeCanvasYamlEdge(edge: ComponentsEdge): ComponentsEdge {
  const raw = edge as ComponentsEdge & { source_id?: string; target_id?: string };
  const { source_id: sourceIdSnake, target_id: targetIdSnake, ...rest } = raw;
  return {
    ...rest,
    sourceId: edge.sourceId || sourceIdSnake || "",
    targetId: edge.targetId || targetIdSnake || "",
    channel: edge.channel || "default",
  };
}

function normalizeCanvasYamlPosition(position: ComponentsNode["position"]): ComponentsNode["position"] {
  if (!position || typeof position !== "object") {
    return position;
  }

  const raw = position as Record<string, unknown> & { true?: number };
  const x = typeof raw.x === "number" ? raw.x : undefined;
  const y = typeof raw.y === "number" ? raw.y : typeof raw.true === "number" ? raw.true : undefined;
  if (x === undefined || y === undefined) {
    return position;
  }

  return { x, y };
}

function normalizeCanvasYamlNode(node: ComponentsNode): ComponentsNode {
  const raw = node as ComponentsNode & { componentName?: string; is_collapsed?: boolean };
  const { componentName: componentNameLegacy, is_collapsed: isCollapsedSnake, ...rest } = raw;

  const componentName =
    typeof rest.component === "string"
      ? rest.component
      : typeof componentNameLegacy === "string"
        ? componentNameLegacy
        : undefined;

  const normalized: ComponentsNode = {
    ...rest,
    ...(componentName ? { component: componentName } : {}),
    ...(rest.position ? { position: normalizeCanvasYamlPosition(rest.position) } : {}),
    ...(typeof rest.isCollapsed === "boolean"
      ? { isCollapsed: rest.isCollapsed }
      : typeof isCollapsedSnake === "boolean"
        ? { isCollapsed: isCollapsedSnake }
        : {}),
  };

  if (!normalized.type && componentName) {
    normalized.type = "TYPE_ACTION";
  }

  return normalized;
}

function quoteYamlPositionYKeys(text: string): string {
  return text.replace(/^(\s+)y: /gm, '$1"y": ');
}

export function buildCanvasYamlFromWorkflow(workflow: CanvasesCanvas): string {
  const document = {
    apiVersion: "v1",
    kind: "Canvas",
    metadata: {
      id: workflow.metadata?.id || "",
      name: workflow.metadata?.name || "Canvas",
      description: workflow.metadata?.description || "",
      isTemplate: workflow.metadata?.isTemplate ?? false,
    },
    spec: {
      nodes: (workflow.spec?.nodes ?? []).map(normalizeCanvasYamlNode),
      edges: (workflow.spec?.edges ?? []).map(normalizeCanvasYamlEdge),
      ...(workflow.spec?.changeManagement ? { changeManagement: workflow.spec.changeManagement } : {}),
    },
  };

  return quoteYamlPositionYKeys(yaml.dump(document, { lineWidth: -1, noRefs: true }));
}
