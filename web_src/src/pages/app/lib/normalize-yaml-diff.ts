import * as yaml from "js-yaml";

// normalizeYamlForDiff re-serializes a YAML document with a deterministic,
// alphabetically-sorted key ordering so the Live vs Draft diff view compares
// semantic content rather than incidental field ordering. Two documents that
// differ only in the order their keys were emitted normalize to identical text,
// keeping the diff focused on real value changes.
//
// Key sorting alone does not reorder sequences, so any `nodes` section is also
// sorted by node id. This keeps the diff stable when Live and Draft list the
// same nodes in a different order.
//
// The input is returned unchanged when it is empty or cannot be parsed as a
// YAML mapping/sequence, so malformed content still renders in the diff instead
// of being silently dropped.
export function normalizeYamlForDiff(yamlText: string): string {
  const trimmed = yamlText.trim();
  if (!trimmed) {
    return yamlText;
  }

  let parsed: unknown;
  try {
    parsed = yaml.load(trimmed);
  } catch {
    return yamlText;
  }

  if (parsed === null || typeof parsed !== "object") {
    return yamlText;
  }

  return yaml.dump(sortNodeSections(parsed), { sortKeys: true, lineWidth: -1, noRefs: true });
}

// sortNodeSections returns a structural copy of the parsed document with every
// `nodes` sequence ordered by node id, recursing through nested mappings and
// sequences so the ordering is applied wherever a nodes section appears.
function sortNodeSections(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortNodeSections);
  }

  if (value === null || typeof value !== "object") {
    return value;
  }

  const result: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    if (key === "nodes" && Array.isArray(entry)) {
      result[key] = entry
        .map(sortNodeSections)
        .sort((a, b) => nodeIdSortKey(a).localeCompare(nodeIdSortKey(b)));
    } else {
      result[key] = sortNodeSections(entry);
    }
  }
  return result;
}

function nodeIdSortKey(node: unknown): string {
  if (node !== null && typeof node === "object" && "id" in node) {
    return String((node as { id: unknown }).id ?? "");
  }
  return "";
}
