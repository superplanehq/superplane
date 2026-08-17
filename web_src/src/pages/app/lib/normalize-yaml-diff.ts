import * as yaml from "js-yaml";

// normalizeYamlForDiff re-serializes a YAML document with a deterministic,
// alphabetically-sorted key ordering so the Live vs Draft diff view compares
// semantic content rather than incidental field ordering. Two documents that
// differ only in the order their keys were emitted normalize to identical text,
// keeping the diff focused on real value changes.
//
// Key sorting alone does not reorder sequences, so the `nodes` and `edges`
// sections are also sorted by a stable identity: nodes by node id, edges by
// their source/target/channel. This keeps the diff stable when Live and Draft
// list the same nodes or edges in a different order.
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

  return yaml.dump(sortDiffSequences(parsed), { sortKeys: true, lineWidth: -1, noRefs: true });
}

// sortDiffSequences returns a structural copy of the parsed document with the
// `nodes` and `edges` sequences ordered by a stable identity, recursing through
// nested mappings and sequences so the ordering is applied wherever those
// sections appear.
function sortDiffSequences(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortDiffSequences);
  }

  if (value === null || typeof value !== "object") {
    return value;
  }

  const result: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    const sortKey = SEQUENCE_SORT_KEYS[key];
    if (sortKey && Array.isArray(entry)) {
      result[key] = entry.map(sortDiffSequences).sort((a, b) => sortKey(a).localeCompare(sortKey(b)));
    } else if (key === "integration") {
      result[key] = normalizeIntegrationRef(entry);
    } else {
      result[key] = sortDiffSequences(entry);
    }
  }
  return result;
}

// normalizeIntegrationRef drops a blank `name` from a node's integration
// reference. The backend only persists the integration id, so historically it
// serialized `integration: { id, name: "" }` while the frontend serialized
// `integration: { id }`. That mismatch surfaced a spurious `name: ""` line in
// the Live vs Draft diff for every integration-backed node. Treating the empty
// name as absent keeps the diff focused on real changes, including for content
// that was committed before the backend stopped emitting the empty name.
function normalizeIntegrationRef(value: unknown): unknown {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return sortDiffSequences(value);
  }

  const ref = value as Record<string, unknown>;
  if (!("name" in ref) || (ref.name !== "" && ref.name !== null && ref.name !== undefined)) {
    return sortDiffSequences(value);
  }

  const { name: _blankName, ...rest } = ref;
  return sortDiffSequences(rest);
}

// SEQUENCE_SORT_KEYS maps a mapping key to the identity used to order its
// sequence entries, so both `nodes` and `edges` sections normalize to a
// deterministic order regardless of how Live and Draft emitted them.
const SEQUENCE_SORT_KEYS: Record<string, (entry: unknown) => string> = {
  nodes: nodeSortKey,
  edges: edgeSortKey,
};

function nodeSortKey(node: unknown): string {
  return fieldString(node, "id");
}

function edgeSortKey(edge: unknown): string {
  // Edges have no id, so combine the fields that make one edge distinct.
  return [fieldString(edge, "sourceId"), fieldString(edge, "targetId"), fieldString(edge, "channel")].join("\0");
}

function fieldString(entry: unknown, field: string): string {
  if (entry !== null && typeof entry === "object" && field in entry) {
    return String((entry as Record<string, unknown>)[field] ?? "");
  }
  return "";
}
