import { describe, expect, it } from "bun:test";

import { buildColumnAutomations, catalogForColumn } from "./columnAutomations";

describe("custom stage automations", () => {
  it("lists an attached automation", () => {
    const automations = buildColumnAutomations("custom-stage-id", {
      columnTitle: "Deploy to prod",
      apps: [{ id: "app-deploy", name: "Deploy", columnKey: "custom-stage-id" }],
    });

    expect(automations).toEqual([expect.objectContaining({ kind: "custom", name: "Deploy", canvasId: "app-deploy" })]);
  });

  it("offers a generic automation", () => {
    expect(catalogForColumn("custom-stage-id").map((entry) => entry.id)).toEqual(["custom"]);
  });
});
