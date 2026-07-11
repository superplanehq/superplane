import { describe, expect, it } from "vitest";

import { validatePanelContent } from "./panelTypes";

describe("validatePanelContent — scorecard", () => {
  it("accepts a fully configured scorecard panel", () => {
    expect(
      validatePanelContent("scorecard", {
        title: "Open UX papercuts",
        dataSource: { kind: "memory", namespace: "ux_papercuts" },
        render: {
          kind: "scorecard",
          aggregation: "last",
          field: "openCount",
          format: "number",
          label: "Open UX papercuts",
          better: "down",
          target: "80",
          showProgress: true,
          sparklineField: "openCount",
          showChange: "both",
          changeCaption: "vs start of range",
        },
      }),
    ).toBeNull();
  });

  it("accepts a count scorecard without a field", () => {
    expect(
      validatePanelContent("scorecard", {
        dataSource: { kind: "runs", limit: 100 },
        render: { kind: "scorecard", aggregation: "count" },
      }),
    ).toBeNull();
  });

  it("rejects a scorecard without a data source", () => {
    expect(
      validatePanelContent("scorecard", {
        render: { kind: "scorecard", aggregation: "count" },
      }),
    ).toMatch(/dataSource must be an object/);
  });

  it("rejects a scorecard with the wrong render kind", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "number", aggregation: "count" },
    });
    expect(error).toMatch(/render\.kind must be "scorecard"/);
  });

  it("rejects unknown aggregations", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "median" },
    });
    expect(error).toMatch(/render\.aggregation must be one of/);
  });

  it("requires a field when aggregation is not count", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "last" },
    });
    expect(error).toMatch(/render\.field is required/);
  });

  it("rejects invalid better values", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "count", better: "sideways" },
    });
    expect(error).toMatch(/render\.better must be one of/);
  });

  it("rejects invalid showChange values", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "count", showChange: "chart" },
    });
    expect(error).toMatch(/render\.showChange must be one of/);
  });

  it("rejects a non-string target", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "count", target: 42 },
    });
    expect(error).toMatch(/render\.target must be a string/);
  });

  it("rejects a non-boolean showProgress", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "count", showProgress: "yes" },
    });
    expect(error).toMatch(/render\.showProgress must be a boolean/);
  });

  it("rejects a non-string prefix", () => {
    const error = validatePanelContent("scorecard", {
      dataSource: { kind: "memory", namespace: "x" },
      render: { kind: "scorecard", aggregation: "count", prefix: 42 },
    });
    expect(error).toMatch(/render\.prefix must be a string/);
  });
});
