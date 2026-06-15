import { describe, expect, it } from "vitest";

import { normalizeSpecFileContentForDiff } from "./spec-yaml-normalize";

describe("normalizeSpecFileContentForDiff", () => {
  it("collapses serialization-only differences in canvas.yaml so only real edits remain", () => {
    // Server-rendered committed content: alphabetical keys, double quotes, no isCollapsed.
    const committed = [
      "apiVersion: v1",
      "kind: Canvas",
      "metadata:",
      '  description: ""',
      "  id: canvas-1",
      "  name: fierce-apex",
      "spec:",
      "  edges: []",
      "  nodes:",
      "    - configuration: {}",
      "      id: node-1",
      "      name: New Component",
      "      position:",
      "        x: 400",
      '        "y": 300',
      "      type: TYPE_ACTION",
      "",
    ].join("\n");

    // Client-rendered staged content: natural client key order, single quotes,
    // emits isCollapsed: false, and the node name was edited.
    const effective = [
      "apiVersion: v1",
      "kind: Canvas",
      "metadata:",
      "  id: canvas-1",
      "  name: fierce-apex",
      "  description: ''",
      "spec:",
      "  nodes:",
      "    - configuration: {}",
      "      id: node-1",
      "      name: New Componen",
      "      position:",
      "        x: 400",
      '        "y": 300',
      "      type: TYPE_ACTION",
      "      isCollapsed: false",
      "  edges: []",
      "",
    ].join("\n");

    const normalizedCommitted = normalizeSpecFileContentForDiff("canvas.yaml", committed);
    const normalizedEffective = normalizeSpecFileContentForDiff("canvas.yaml", effective);

    const committedLines = normalizedCommitted.split("\n");
    const effectiveLines = normalizedEffective.split("\n");
    expect(committedLines.length).toBe(effectiveLines.length);

    const differingLines = committedLines.filter((line, index) => line !== effectiveLines[index]);
    expect(differingLines).toEqual(["      name: New Component"]);
    expect(normalizedEffective).toContain("      name: New Componen");
    expect(normalizedEffective).not.toContain("isCollapsed");
  });

  it("keeps a truthy isCollapsed so real collapse changes still surface", () => {
    const collapsed = normalizeSpecFileContentForDiff(
      "canvas.yaml",
      ["spec:", "  nodes:", "    - id: node-1", "      isCollapsed: true", "  edges: []", ""].join("\n"),
    );

    expect(collapsed).toContain("isCollapsed: true");
  });

  it("returns repository (non-spec) file content unchanged", () => {
    const text = "line two\nline one\n";
    expect(normalizeSpecFileContentForDiff("README.md", text)).toBe(text);
  });

  it("returns the original text when the YAML cannot be parsed", () => {
    const invalid = "spec: [unterminated";
    expect(normalizeSpecFileContentForDiff("canvas.yaml", invalid)).toBe(invalid);
  });
});
