import { describe, expect, it } from "bun:test";

import {
  DEFAULT_INTENT_LEFT_PERCENT,
  DEFAULT_LOG_PERCENT,
  DEFAULT_REFINE_INTENT_LEFT_PERCENT,
} from "./useSplitRunPanePercent";

describe("useSplitRunPanePercent", () => {
  it("opens the automation run log at 65 percent width", () => {
    expect(DEFAULT_LOG_PERCENT).toBe(65);
  });

  it("opens the request pane at two fifths width", () => {
    expect(DEFAULT_INTENT_LEFT_PERCENT).toBe(40);
  });

  it("opens refine chat and plan at half width", () => {
    expect(DEFAULT_REFINE_INTENT_LEFT_PERCENT).toBe(50);
  });
});
