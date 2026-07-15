import * as yaml from "js-yaml";

// normalizeYamlForDiff re-serializes a YAML document with a deterministic,
// alphabetically-sorted key ordering so the Live vs Draft diff view compares
// semantic content rather than incidental field ordering. Two documents that
// differ only in the order their keys were emitted normalize to identical text,
// keeping the diff focused on real value changes.
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

  return yaml.dump(parsed, { sortKeys: true, lineWidth: -1, noRefs: true });
}
