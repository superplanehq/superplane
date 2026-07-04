import * as yaml from "js-yaml";

import { isWorkflowSpecPath } from "../../lib/workflow-spec-paths";

// Renders a virtual spec file (canvas.yaml / console.yaml) in a canonical form
// so the Files diff only surfaces real changes. The committed side is
// server-serialized while the effective side is client-serialized, so without
// normalization the two differ in key ordering, quote style, and cosmetic
// defaults the client emits — drowning the actual edit in noise.
//
// Repository files (e.g. README.md) are raw text and returned unchanged.
export function normalizeSpecFileContentForDiff(path: string, text: string): string {
  if (!isWorkflowSpecPath(path)) {
    return text;
  }

  const trimmed = text.trim();
  if (!trimmed) {
    return text;
  }

  try {
    const parsed = yaml.load(trimmed);
    if (parsed === undefined || parsed === null || typeof parsed !== "object") {
      return text;
    }

    const cleaned = stripCosmeticDefaults(parsed);
    return yaml.dump(cleaned, { sortKeys: true, lineWidth: -1, noRefs: true });
  } catch {
    return text;
  }
}

// isCollapsed:false is the node default; the client serializer emits it while
// the committed server content omits it, so drop it on both sides to keep the
// diff focused on meaningful changes. A truthy isCollapsed is kept so real
// collapse/expand changes still surface.
function stripCosmeticDefaults(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(stripCosmeticDefaults);
  }

  if (value && typeof value === "object") {
    const result: Record<string, unknown> = {};
    for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
      if (key === "isCollapsed" && entry === false) {
        continue;
      }
      result[key] = stripCosmeticDefaults(entry);
    }
    return result;
  }

  return value;
}
