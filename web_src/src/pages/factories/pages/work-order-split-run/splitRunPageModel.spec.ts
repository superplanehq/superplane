import { describe, expect, it } from "vitest";

import { OPEN_WORK_ORDER, RUNNING_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SPLIT_RUN_RUNNING } from "./splitRunMocks";
import {
  fixtureForSplitRunPage,
  phaseForSplitRunCanvas,
  readSplitRunQuery,
  resolveSplitRunOrder,
  splitRunMissingCopy,
  splitRunPageTitle,
  splitRunPhaseOnRoute,
} from "./splitRunPageModel";

describe("readSplitRunQuery", () => {
  it("reads canvas, run, and order from the URL", () => {
    const query = readSplitRunQuery(
      new URLSearchParams("from=lines&lineId=line-1&run=run-9&orderNumber=103&canvas=planning"),
    );
    expect(query).toEqual({
      from: "lines",
      lineId: "line-1",
      runId: "run-9",
      orderNumber: "103",
      canvasKey: "planning",
    });
  });

  it("reads the legacy orderId param and leaves canvas unset", () => {
    const query = readSplitRunQuery(new URLSearchParams("orderId=wo-1"));
    expect(query.canvasKey).toBeUndefined();
    expect(query.orderNumber).toBe("wo-1");
  });
});

describe("resolveSplitRunOrder", () => {
  it("matches a work order by number", () => {
    expect(resolveSplitRunOrder([OPEN_WORK_ORDER], String(OPEN_WORK_ORDER.number), null, false)?.id).toBe(
      OPEN_WORK_ORDER.id,
    );
  });
});

describe("fixtureForSplitRunPage", () => {
  it("returns null when no work order is selected", () => {
    expect(fixtureForSplitRunPage(null, [], null)).toBeNull();
  });

  it("maps a loaded work order", () => {
    expect(fixtureForSplitRunPage(RUNNING_WORK_ORDER, [], null)?.title).toBe(RUNNING_WORK_ORDER.title);
  });
});

describe("phaseForSplitRunCanvas", () => {
  it("picks the phase that matches the canvas key", () => {
    expect(phaseForSplitRunCanvas(SPLIT_RUN_RUNNING, "planning").id).toBe("plan");
  });

  it("uses the current phase when the canvas key is missing", () => {
    expect(phaseForSplitRunCanvas({ ...SPLIT_RUN_RUNNING, currentPhaseId: "plan" }).id).toBe("plan");
  });

  it("falls back to implement when the fixture is missing", () => {
    expect(phaseForSplitRunCanvas(null, "implementation").id).toBe("implement");
  });
});

describe("splitRunPhaseOnRoute", () => {
  it("fills a missing app id from the route and keeps the mapped run", () => {
    const implement = {
      ...phaseForSplitRunCanvas(SPLIT_RUN_RUNNING, "implementation"),
      appId: undefined,
      runId: "run-mapped",
    };
    expect(splitRunPhaseOnRoute(implement, "app-refund-implementer")).toMatchObject({
      appId: "app-refund-implementer",
      runId: "run-mapped",
    });
  });

  it("does not invent a run id when the phase has none", () => {
    const implement = { ...phaseForSplitRunCanvas(SPLIT_RUN_RUNNING, "implementation"), runId: undefined };
    expect(splitRunPhaseOnRoute(implement, "app-refund-implementer").runId).toBeUndefined();
  });
});

describe("splitRunPageTitle", () => {
  it("uses the canvas title once the run is ready", () => {
    expect(splitRunPageTitle(false, false, "Implementation")).toBe("Implementation");
  });
});

describe("splitRunMissingCopy", () => {
  it("explains a missing run", () => {
    expect(splitRunMissingCopy(false)).toEqual({
      title: "Run not found",
      body: "This run is not on the workspace.",
    });
  });

  it("explains a canvas or run that did not load", () => {
    expect(splitRunMissingCopy(false, true)).toEqual({
      title: "Run not found",
      body: "SuperPlane cannot load this canvas or run.",
    });
  });
});
