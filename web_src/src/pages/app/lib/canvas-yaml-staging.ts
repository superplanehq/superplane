import * as yaml from "js-yaml";
import type { CanvasesCanvas } from "@/api-client";
import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

type ParsedCanvasYaml = {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    id?: string;
    name?: string;
    description?: string;
  };
  spec?: {
    nodes?: ComponentsNode[];
    edges?: ComponentsEdge[];
  };
};

export type ParsedCanvasYamlMetadata = {
  id?: string;
  name?: string;
  description?: string;
};

// loadCanvasYamlDocument parses and validates that the text is a Canvas YAML
// document, returning the parsed object or null. Shared by the spec and metadata
// readers to keep their individual complexity low.
function loadCanvasYamlDocument(text: string): ParsedCanvasYaml | null {
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

  return parsed;
}

// parseCanvasYamlMetadata extracts the canvas-level metadata (id/name/description)
// from a canvas.yaml document. Used so edit mode can source the canvas name from
// the repository file instead of the DescribeCanvas response.
export function parseCanvasYamlMetadata(text: string): ParsedCanvasYamlMetadata | null {
  const parsed = loadCanvasYamlDocument(text);
  const metadata = parsed?.metadata;
  if (!metadata || typeof metadata !== "object" || Array.isArray(metadata)) {
    return null;
  }

  return {
    ...(typeof metadata.id === "string" ? { id: metadata.id } : {}),
    ...(typeof metadata.name === "string" ? { name: metadata.name } : {}),
    ...(typeof metadata.description === "string" ? { description: metadata.description } : {}),
  };
}

export function parseCanvasYamlToSpec(text: string): CanvasesCanvas["spec"] | null {
  const parsed = loadCanvasYamlDocument(text);
  if (!parsed) {
    return null;
  }

  if (parsed.spec === undefined || (parsed.spec !== null && typeof parsed.spec !== "object")) {
    return null;
  }

  const spec = parsed.spec ?? {};

  if (spec.nodes !== undefined && !Array.isArray(spec.nodes)) {
    return null;
  }

  if (spec.edges !== undefined && !Array.isArray(spec.edges)) {
    return null;
  }

  return {
    nodes: (spec.nodes ?? []).map(normalizeCanvasYamlNode),
    edges: spec.edges ?? [],
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
    },
    spec: {
      nodes: (workflow.spec?.nodes ?? []).map(normalizeCanvasYamlNode),
      edges: workflow.spec?.edges ?? [],
    },
  };

  return quoteYamlPositionYKeys(yaml.dump(document, { lineWidth: -1, noRefs: true }));
}
