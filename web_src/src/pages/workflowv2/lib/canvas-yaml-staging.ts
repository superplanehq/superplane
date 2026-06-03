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

  if (!parsed.spec || !Array.isArray(parsed.spec.nodes)) {
    return null;
  }

  return {
    nodes: parsed.spec.nodes,
    edges: parsed.spec.edges ?? [],
    changeManagement: parsed.spec.changeManagement,
  };
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
      nodes: workflow.spec?.nodes ?? [],
      edges: workflow.spec?.edges ?? [],
      ...(workflow.spec?.changeManagement ? { changeManagement: workflow.spec.changeManagement } : {}),
    },
  };

  return yaml.dump(document, { lineWidth: -1, noRefs: true });
}
