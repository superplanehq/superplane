import type { FactoriesDescribeFactoryVelocityResponse } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  hasVelocityOutput,
  toVelocityReport,
  velocityBreakdownSeries,
  type VelocityIntakeSeries,
} from "./factoryVelocityReport";

const RESPONSE: FactoriesDescribeFactoryVelocityResponse = {
  totals: {
    superplaneMerged: 20,
    peopleMerged: 30,
    waste: 5,
    wastePct: 20,
    costCents: "12000",
    tokens: "450000",
    wasteCostCents: "2500",
  },
  previousTotals: {
    superplaneMerged: 10,
    peopleMerged: 25,
    waste: 6,
    wastePct: 38,
    costCents: "9000",
    tokens: "300000",
    wasteCostCents: "3000",
  },
  hasPreviousWindow: true,
  points: [
    {
      day: "1",
      superplaneMerged: 3,
      peopleMerged: 4,
      waste: 1,
      costCents: "800",
      tokens: "12000",
      wasteCostCents: "150",
      intake: [
        { key: "github-issues", merged: 2 },
        { key: "manual", merged: 1 },
      ],
    },
  ],
  intakeSources: [
    { key: "github-issues", label: "GitHub issue", merged: 14 },
    { key: "manual", label: "Manually created", merged: 6 },
  ],
  people: [
    {
      id: "user-1",
      name: "Igor Šarčević",
      email: "igor@superplane.com",
      avatarUrl: "https://avatars.example/igor.png",
      authoredMerged: 7,
      factoryMerged: 5,
      factoryWaste: 2,
      medianCycleHours: 18,
      costCents: "3000",
    },
  ],
  hasPeopleCohort: true,
  repository: "acme/refunds",
};

describe("toVelocityReport", () => {
  it("adds people and SuperPlane merges into one merged total", () => {
    const report = toVelocityReport(RESPONSE);

    expect(report.totals.merged).toBe(50);
    expect(report.totals.peopleMerged).toBe(30);
    expect(report.totals.superplaneMerged).toBe(20);
  });

  it("converts cents to dollars and derives cost per SuperPlane merge", () => {
    const report = toVelocityReport(RESPONSE);

    expect(report.totals.costUsd).toBe(120);
    expect(report.totals.wasteCostUsd).toBe(25);
    expect(report.totals.tokens).toBe(450_000);
    expect(report.totals.costPerMerge).toBe(6);
  });

  it("keeps the waste rate the API reports", () => {
    const report = toVelocityReport(RESPONSE);

    expect(report.totals.wasteRate).toBe(20);
  });

  it("derives the waste rate when the API omits it", () => {
    const report = toVelocityReport({
      ...RESPONSE,
      totals: { superplaneMerged: 3, peopleMerged: 0, waste: 1 },
    });

    expect(report.totals.wasteRate).toBe(25);
  });

  it("reports no previous totals when the API has no earlier window", () => {
    const report = toVelocityReport({ ...RESPONSE, hasPreviousWindow: false });

    expect(report.previous).toBeUndefined();
  });

  it("maps previous totals when the API has an earlier window", () => {
    const report = toVelocityReport(RESPONSE);

    expect(report.previous?.merged).toBe(35);
    expect(report.previous?.costPerMerge).toBe(9);
  });

  it("keys the intake counts of a day by source", () => {
    const [point] = toVelocityReport(RESPONSE).points;

    expect(point.merged).toBe(7);
    expect(point.costUsd).toBe(8);
    expect(point.intake).toEqual({ "github-issues": 2, manual: 1 });
  });

  it("gives every intake source a color and a label", () => {
    const report = toVelocityReport(RESPONSE);

    expect(report.intakeSeries).toHaveLength(2);
    expect(report.intakeSeries[0].label).toBe("GitHub issue");
    expect(report.intakeSeries[0].color).toMatch(/^#[0-9a-f]{6}$/i);
    expect(report.intakeSeries[1].color).not.toBe(report.intakeSeries[0].color);
  });

  it("labels an intake source by its key when the API sends no label", () => {
    const report = toVelocityReport({
      ...RESPONSE,
      intakeSources: [{ key: "linear-issues", merged: 4 }],
    });

    expect(report.intakeSeries[0].label).toBe("linear-issues");
  });

  it("names a person without a display name by email", () => {
    const report = toVelocityReport({
      ...RESPONSE,
      people: [{ id: "user-2", email: "pedro@superplane.com", authoredMerged: 1 }],
    });

    expect(report.people[0].name).toBe("pedro@superplane.com");
    expect(report.people[0].avatarUrl).toBeUndefined();
  });

  it("tolerates a response with no series at all", () => {
    const report = toVelocityReport({});

    expect(report.totals.merged).toBe(0);
    expect(report.points).toEqual([]);
    expect(report.intakeSeries).toEqual([]);
    expect(report.people).toEqual([]);
    expect(report.hasPeopleCohort).toBe(false);
    expect(report.repository).toBeUndefined();
  });
});

describe("hasVelocityOutput", () => {
  it("is false when the window holds no merges, no waste and no spend", () => {
    expect(hasVelocityOutput(toVelocityReport({}))).toBe(false);
  });

  it("is true when the window only holds spend", () => {
    const report = toVelocityReport({ totals: { costCents: "500" } });

    expect(hasVelocityOutput(report)).toBe(true);
  });

  it("is true when the window only holds waste", () => {
    const report = toVelocityReport({ totals: { waste: 2 } });

    expect(hasVelocityOutput(report)).toBe(true);
  });
});

describe("velocityBreakdownSeries", () => {
  const intakeSeries: VelocityIntakeSeries[] = [
    { key: "github-issues", label: "GitHub issue", color: "#3b82f6", merged: 4 },
  ];

  it("splits origin into people and SuperPlane", () => {
    expect(velocityBreakdownSeries("origin", intakeSeries).map((series) => series.key)).toEqual([
      "people",
      "superplane",
    ]);
  });

  it("splits outcome into merged and closed without merge", () => {
    expect(velocityBreakdownSeries("outcome", intakeSeries).map((series) => series.key)).toEqual(["merged", "waste"]);
  });

  it("takes the intake bands from the report, so unused sources stay hidden", () => {
    expect(velocityBreakdownSeries("intake", intakeSeries)).toEqual([
      { key: "github-issues", label: "GitHub issue", color: "#3b82f6" },
    ]);
    expect(velocityBreakdownSeries("intake", [])).toEqual([]);
  });
});
