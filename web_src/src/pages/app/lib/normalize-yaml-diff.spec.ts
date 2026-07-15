import { describe, expect, it } from "vitest";
import * as yaml from "js-yaml";

import { normalizeYamlForDiff } from "./normalize-yaml-diff";

describe("normalizeYamlForDiff", () => {
  it("produces identical output for documents that differ only in key ordering", () => {
    const live = "name: deploy\ntype: TYPE_ACTION\nref: r\n";
    const draft = "ref: r\ntype: TYPE_ACTION\nname: deploy\n";

    expect(normalizeYamlForDiff(live)).toBe(normalizeYamlForDiff(draft));
  });

  it("sorts nested mapping keys recursively", () => {
    const input = "spec:\n  nodes:\n    - name: n\n      configuration:\n        b: 2\n        a: 1\n";

    const normalized = normalizeYamlForDiff(input);

    expect(normalized).toBe(yaml.dump(yaml.load(input), { sortKeys: true, lineWidth: -1, noRefs: true }));
    expect(normalized.indexOf("a: 1")).toBeLessThan(normalized.indexOf("b: 2"));
  });

  it("orders the nodes section by node id", () => {
    const live = "spec:\n  nodes:\n    - id: b\n      name: second\n    - id: a\n      name: first\n";
    const draft = "spec:\n  nodes:\n    - id: a\n      name: first\n    - id: b\n      name: second\n";

    const normalized = normalizeYamlForDiff(live);

    expect(normalized).toBe(normalizeYamlForDiff(draft));
    expect(normalized.indexOf("id: a")).toBeLessThan(normalized.indexOf("id: b"));
  });

  it("orders the edges section by source, target, and channel", () => {
    const live =
      "spec:\n  edges:\n    - sourceId: b\n      targetId: c\n      channel: default\n    - sourceId: a\n      targetId: b\n      channel: default\n";
    const draft =
      "spec:\n  edges:\n    - sourceId: a\n      targetId: b\n      channel: default\n    - sourceId: b\n      targetId: c\n      channel: default\n";

    const normalized = normalizeYamlForDiff(live);

    expect(normalized).toBe(normalizeYamlForDiff(draft));
    expect(normalized.indexOf("sourceId: a")).toBeLessThan(normalized.indexOf("sourceId: b"));
  });

  it("still reports a difference when values actually change", () => {
    const live = "name: old\nref: r\n";
    const draft = "ref: r\nname: new\n";

    expect(normalizeYamlForDiff(live)).not.toBe(normalizeYamlForDiff(draft));
  });

  it("returns the original text for empty input", () => {
    expect(normalizeYamlForDiff("")).toBe("");
    expect(normalizeYamlForDiff("   \n")).toBe("   \n");
  });

  it("returns the original text when the input cannot be parsed as YAML", () => {
    const invalid = "name: : :\n  - broken";
    expect(normalizeYamlForDiff(invalid)).toBe(invalid);
  });

  it("returns the original text for scalar (non-object) documents", () => {
    expect(normalizeYamlForDiff("just a string")).toBe("just a string");
    expect(normalizeYamlForDiff("42")).toBe("42");
  });
});
