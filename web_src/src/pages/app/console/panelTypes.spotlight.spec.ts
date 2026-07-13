import { describe, expect, it } from "vitest";

import { templateForPanelType, validatePanelContent } from "./panelTypes";

describe("spotlight panel type", () => {
  it("seeds a runnable spotlight template with a title field", () => {
    const template = templateForPanelType("spotlight", "Latest merge");
    expect(template.title).toBe("Latest merge");
    expect(template.dataSource).toEqual({ kind: "runs" });
    expect(typeof template.titleField).toBe("string");
    expect(String(template.titleField).length).toBeGreaterThan(0);
  });

  it("accepts a memory spotlight with mapped title field", () => {
    const error = validatePanelContent("spotlight", {
      title: "Latest merge",
      dataSource: { kind: "memory", namespace: "recentMerges" },
      titleField: "title",
      actorNameField: "authorName",
    });
    expect(error).toBeNull();
  });

  it("rejects memory spotlight without a namespace", () => {
    const error = validatePanelContent("spotlight", {
      dataSource: { kind: "memory", namespace: "" },
      titleField: "title",
    });
    expect(error).toMatch(/memory namespace/i);
  });

  it("rejects spotlight without title or actor mapping", () => {
    const error = validatePanelContent("spotlight", {
      dataSource: { kind: "runs" },
      titleField: "",
      actorNameField: "",
    });
    expect(error).toMatch(/title or an actor/i);
  });
});
