import { describe, expect, it } from "vitest";

import { factoryAppRunPath } from "../lib/factoryPagePaths";
import {
  LINE_RUN_IMPLEMENT_FAILED_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_FACTORY_APPS,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { canvasExpandHrefForLine } from "./LinesPage";

describe("canvasExpandHrefForLine", () => {
  const line = REFUND_FACTORY.lines?.[0];

  it("opens the factory run inspector when the canvas has a run", () => {
    expect(line).toBeDefined();
    const hrefFor = canvasExpandHrefForLine(
      "org-1",
      PRIMARY_FACTORY_KEY,
      line!,
      REFUND_FACTORY_APPS,
      BOARD_IMPLEMENT_FAILED_ORDER,
    );

    expect(hrefFor("implementation")).toBe(
      factoryAppRunPath("org-1", PRIMARY_FACTORY_KEY, "app-refund-implementer", LINE_RUN_IMPLEMENT_FAILED_ID, {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
        orderNumber: BOARD_IMPLEMENT_FAILED_ORDER.number,
      }),
    );
    expect(hrefFor("implementation")).toContain(`run=${LINE_RUN_IMPLEMENT_FAILED_ID}`);
    expect(hrefFor("planning")).toBeUndefined();
  });

  it("uses the selected phase run when the canvas name is not a known key", () => {
    expect(line).toBeDefined();
    const hrefFor = canvasExpandHrefForLine("org-1", PRIMARY_FACTORY_KEY, line!, REFUND_FACTORY_APPS, {
      ...BOARD_IMPLEMENT_FAILED_ORDER,
      lineDispatches: [],
    });

    expect(hrefFor("closure", { appId: "app-pr-creation", runId: "run-pr-creation" })).toBe(
      factoryAppRunPath("org-1", PRIMARY_FACTORY_KEY, "app-pr-creation", "run-pr-creation", {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
        orderNumber: BOARD_IMPLEMENT_FAILED_ORDER.number,
      }),
    );
  });

  it("hides expand when the canvas has no run", () => {
    expect(line).toBeDefined();
    const hrefFor = canvasExpandHrefForLine("org-1", PRIMARY_FACTORY_KEY, line!, REFUND_FACTORY_APPS, {
      ...BOARD_IMPLEMENT_FAILED_ORDER,
      lineDispatches: [],
    });

    expect(hrefFor("implementation")).toBeUndefined();
  });

  it("hides expand when the line has no app", () => {
    expect(line).toBeDefined();
    const hrefFor = canvasExpandHrefForLine("org-1", PRIMARY_FACTORY_KEY, { ...line!, steps: [] }, [], undefined);

    expect(hrefFor("implementation")).toBeUndefined();
  });
});
