import { describe, expect, it } from "vitest";
import { getPanelDef, listPanelDefs, panelRegistry } from "./panelRegistry";

describe("panelRegistry", () => {
  it("registers exactly the markdown panel for v1", () => {
    const types = listPanelDefs().map((d) => d.type);
    expect(types).toEqual(["markdown"]);
    expect(Object.keys(panelRegistry)).toEqual(["markdown"]);
  });

  it("returns the markdown panel def via getPanelDef", () => {
    const def = getPanelDef("markdown");
    expect(def).toBeDefined();
    expect(def?.label).toBe("Markdown");
    expect(def?.defaultSize.w).toBeGreaterThan(0);
    expect(def?.defaultSize.h).toBeGreaterThan(0);
  });

  it("returns undefined for an unknown panel type", () => {
    expect(getPanelDef("unknown-type")).toBeUndefined();
  });

  it("normalizes missing markdown content to a safe default", () => {
    const def = getPanelDef("markdown");
    const normalized = def?.normalize(undefined) as { body: string };
    expect(normalized).toEqual({ body: "" });
  });

  it("normalizes non-string body to empty string", () => {
    const def = getPanelDef("markdown");
    const normalized = def?.normalize({ body: 123 as unknown }) as { body: string };
    expect(normalized).toEqual({ body: "" });
  });
});
