import { describe, expect, it } from "vitest";

import { FACTORIES_NAV_ITEMS } from "../layout/factoriesNavItems";
import { STORYBOOK_FACTORIES_NAV_ITEMS } from "./storybookNavItems";

describe("STORYBOOK_FACTORIES_NAV_ITEMS", () => {
  it("inserts Missions after Work Orders and keeps the live items", () => {
    expect(FACTORIES_NAV_ITEMS.map((item) => item.id)).not.toContain("missions");
    expect(STORYBOOK_FACTORIES_NAV_ITEMS.map((item) => item.id)).toEqual([
      "overview",
      "work-orders",
      "missions",
      "lines",
      "automations",
      "wiki",
      "velocity",
    ]);
  });
});
