import { describe, expect, it } from "bun:test";

import { DEFAULT_LOG_PERCENT } from "./useSplitRunPanePercent";

describe("useSplitRunPanePercent", () => {
  it("opens the automation run log at 65 percent width", () => {
    expect(DEFAULT_LOG_PERCENT).toBe(65);
  });
});
