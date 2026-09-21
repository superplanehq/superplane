import { describe, expect, it } from "bun:test";

import {
  DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import {
  OPEN_WORK_ORDER_ARTIFACTS,
  OPEN_WORK_ORDER_PULL_REQUESTS,
} from "../../__fixtures__/factoryPageFixtureVariants";
import { OPEN_WORK_ORDER_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { buildTaskPageRecord, formatTaskSpendLabel, primaryActionForStatus } from "./taskPageModel";

describe("primaryActionForStatus", () => {
  it("returns Start for a draft", () => {
    expect(primaryActionForStatus("draft")).toBe("start");
  });

  it("returns Review when the task needs attention", () => {
    expect(primaryActionForStatus("waiting")).toBe("review");
  });

  it("returns Reopen for a closed task", () => {
    expect(primaryActionForStatus("completed")).toBe("reopen");
    expect(primaryActionForStatus("failed")).toBe("reopen");
  });

  it("returns no primary action while the task runs", () => {
    expect(primaryActionForStatus("running")).toBeNull();
  });
});

describe("formatTaskSpendLabel", () => {
  it("joins cost and tokens the same way as the overview row", () => {
    expect(formatTaskSpendLabel(RUNNING_WORK_ORDER)).toBe("$0.73 · 2.7k tokens");
  });

  it("returns an empty label when there is no spend", () => {
    expect(formatTaskSpendLabel(DRAFT_WORK_ORDER)).toBe("");
  });
});

describe("buildTaskPageRecord", () => {
  it("maps popup fields and extra property rows from an open task", () => {
    const record = buildTaskPageRecord({
      order: OPEN_WORK_ORDER,
      factoryKey: PRIMARY_FACTORY_KEY,
      missionName: "Checkout reliability",
      checks: OPEN_WORK_ORDER_CHECKS,
      artifacts: OPEN_WORK_ORDER_ARTIFACTS,
      pullRequests: OPEN_WORK_ORDER_PULL_REQUESTS,
    });

    expect(record.key).toBe("RF-101");
    expect(record.status).toBe("waiting");
    expect(record.statusLabel).toBe("Needs attention");
    expect(record.author).toBe("On Issue Label");
    expect(record.assignees).toEqual(["Leonardo DiCaprio"]);
    expect(record.mission).toBe("Checkout reliability");
    expect(record.primaryAction).toBe("review");
    expect(record.checks.length).toBeGreaterThan(0);
    expect(record.artifacts.map((item) => item.title)).toContain("Investigation notes");
    expect(record.pullRequests[0]?.title).toContain("#482");
    expect(record.source.label).toContain("payments-service#101");
  });

  it("omits mission and related lists when they are absent", () => {
    const record = buildTaskPageRecord({
      order: { ...DRAFT_WORK_ORDER, description: "", assignees: [] },
      factoryKey: PRIMARY_FACTORY_KEY,
    });

    expect(record.mission).toBeUndefined();
    expect(record.description).toBe("");
    expect(record.assignees).toEqual([]);
    expect(record.checks).toEqual([]);
    expect(record.artifacts).toEqual([]);
    expect(record.pullRequests).toEqual([]);
    expect(record.factoryLines).toEqual([]);
    expect(record.primaryAction).toBe("start");
    expect(record.spendLabel).toBe("");
  });
});
