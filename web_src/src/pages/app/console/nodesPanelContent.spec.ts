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
