import { describe, expect, it } from "vitest";

import { ORG_SPENDING_ONLY_NAV_GROUPS } from "./storybookFactorySettingsNav";

describe("ORG_SPENDING_ONLY_NAV_GROUPS", () => {
  it("keeps a single Spending item under Organization", () => {
    const labels = ORG_SPENDING_ONLY_NAV_GROUPS.flatMap((group) =>
      group.items.map((item) => `${group.id}:${item.section}`),
    );
    expect(labels.filter((item) => item.endsWith(":spending"))).toEqual(["organization:spending"]);
    expect(labels).not.toContain("workspace:spending");
  });
});
