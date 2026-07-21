import { describe, expect, it } from "vitest";

import { NODES_PANEL_FORM_MODES, validateNodesContent } from "./nodesPanelContent";

describe("validateNodesContent formMode", () => {
  it("accepts entries without a formMode (defaults to modal)", () => {
    expect(validateNodesContent({ nodes: [{ node: "start" }] })).toBeNull();
  });

  it.each(NODES_PANEL_FORM_MODES)("accepts formMode = %s", (mode) => {
    expect(validateNodesContent({ nodes: [{ node: "start", formMode: mode }] })).toBeNull();
  });

  it("rejects unknown formMode values", () => {
    const error = validateNodesContent({ nodes: [{ node: "start", formMode: "drawer" }] });
    expect(error).toMatch(/content\.nodes\[0\]\.formMode/);
    expect(error).toContain('"modal"');
    expect(error).toContain('"inline"');
  });

  it("rejects non-string formMode values", () => {
    const error = validateNodesContent({ nodes: [{ node: "start", formMode: true }] });
    expect(error).toMatch(/formMode/);
  });
});

describe("validateNodesContent inline presentation", () => {
  it("accepts concise inline-form presentation overrides", () => {
    expect(
      validateNodesContent({
        nodes: [
          {
            node: "start",
            formMode: "inline",
            showNodeLabel: false,
            showFieldLabels: false,
            submitLabel: "Create task",
          },
        ],
      }),
    ).toBeNull();
  });

  it.each(["showNodeLabel", "showFieldLabels"])("rejects a non-boolean %s", (field) => {
    expect(validateNodesContent({ nodes: [{ node: "start", [field]: "no" }] })).toMatch(
      new RegExp(`content\\.nodes\\[0\\]\\.${field}`),
    );
  });

  it("rejects a non-string submitLabel", () => {
    expect(validateNodesContent({ nodes: [{ node: "start", submitLabel: 42 }] })).toMatch(
      /content\.nodes\[0\]\.submitLabel/,
    );
  });
});
