import * as yaml from "js-yaml";
import type { ComponentsNode, ComponentsEdge } from "@/api-client";

export interface ParsedCanvas {
  name: string;
  description?: string;
  nodes: ComponentsNode[];
  edges: ComponentsEdge[];
}

/*
   Parses a YAML string into a Canvas definition.
  
   Accepts two shapes:
    - UI export format:  { metadata, spec }
    - CLI resource format: { apiVersion, kind: "Canvas", metadata, spec }
  
   Returns a ParsedCanvas on success, or throws an Error with a user-friendly message.
 */
export function parseCanvasYaml(text: string): ParsedCanvas {
  if (!text.trim()) {
    throw new Error("YAML content is empty.");
  }

  let parsed: unknown;
  try {
    parsed = yaml.load(text);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`Invalid YAML syntax: ${message}`);
  }

  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    throw new Error("YAML must be an object with 'metadata' and optionally 'spec'.");
  }

  const doc = parsed as Record<string, unknown>;

  if ("kind" in doc) {
    if (doc.kind !== "Canvas") {
      throw new Error(`Unsupported resource kind "${doc.kind}". Expected "Canvas".`);
    }
  }

  if (!doc.metadata || typeof doc.metadata !== "object" || Array.isArray(doc.metadata)) {
    throw new Error("Missing or invalid 'metadata' section. It must include at least a 'name' field.");
  }

  const metadata = doc.metadata as Record<string, unknown>;

  const name = typeof metadata.name === "string" ? metadata.name.trim() : "";
  if (!name) {
    throw new Error("'metadata.name' is required and must be a non-empty string.");
  }

  const description = typeof metadata.description === "string" ? metadata.description.trim() : undefined;

  let nodes: ComponentsNode[] = [];
  let edges: ComponentsEdge[] = [];

  if (doc.spec && typeof doc.spec === "object" && !Array.isArray(doc.spec)) {
    const spec = doc.spec as Record<string, unknown>;

    if (spec.nodes !== undefined) {
      if (!Array.isArray(spec.nodes)) {
        throw new Error("'spec.nodes' must be an array.");
      }
      nodes = spec.nodes as ComponentsNode[];
    }

    if (spec.edges !== undefined) {
      if (!Array.isArray(spec.edges)) {
        throw new Error("'spec.edges' must be an array.");
      }
      edges = spec.edges as ComponentsEdge[];
    }
  }

  return { name, description, nodes, edges };
}

export function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(new Error("Failed to read the file."));
    reader.readAsText(file);
  });
}
