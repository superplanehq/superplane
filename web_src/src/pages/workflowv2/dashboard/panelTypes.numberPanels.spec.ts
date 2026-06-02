import { describe, expect, it } from "vitest";

import { validatePanelContent } from "./panelTypes";

describe("validatePanelContent — number and data sources", () => {
  it("requires a known aggregation on number panels", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "executions" },
      render: { kind: "number", aggregation: "median" },
    });
    expect(error).toMatch(/render\.aggregation must be one of/);
  });

  it("requires field when aggregation is not count", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "executions" },
      render: { kind: "number", aggregation: "sum" },
    });
    expect(error).toMatch(/render\.field is required/);
  });

  it("memory data sources require namespace to be a string", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory" },
      render: { kind: "table", columns: [{ field: "x" }] },
    });
    expect(error).toMatch(/dataSource\.namespace must be a string/);
  });

  it("accepts runs data sources", () => {
    expect(
      validatePanelContent("number", {
        dataSource: { kind: "runs" },
        render: { kind: "number", aggregation: "count" },
      }),
    ).toBeNull();
  });

  it("accepts number panels with prefix and suffix symbols", () => {
    expect(
      validatePanelContent("number", {
        dataSource: { kind: "runs" },
        render: { kind: "number", aggregation: "count", prefix: "R$", suffix: " MWh" },
      }),
    ).toBeNull();
  });

  it("rejects non-string prefix on number panels", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "runs" },
      render: { kind: "number", aggregation: "count", prefix: 42 },
    });
    expect(error).toMatch(/render\.prefix must be a string/);
  });

  it("accepts composite memory number panels with heterogeneous sources", () => {
    expect(
      validatePanelContent("number", {
        dataSource: {
          kind: "memory",
          combine: "sum",
          sources: [
            { namespace: "a", aggregation: "sum", field: "cost" },
            { namespace: "b", aggregation: "count" },
          ],
        },
        render: { kind: "number" },
      }),
    ).toBeNull();
  });

  it("rejects composite memory panels that also set render.aggregation", () => {
    const error = validatePanelContent("number", {
      dataSource: {
        kind: "memory",
        combine: "sum",
        sources: [{ namespace: "a", aggregation: "sum", field: "cost" }],
      },
      render: { kind: "number", aggregation: "sum" },
    });
    expect(error).toMatch(/render\.aggregation must not be set/);
  });

  it("rejects composite memory panels that also set render.field", () => {
    const error = validatePanelContent("number", {
      dataSource: {
        kind: "memory",
        combine: "sum",
        sources: [{ namespace: "a", aggregation: "sum", field: "cost" }],
      },
      render: { kind: "number", field: "cost" },
    });
    expect(error).toMatch(/render\.field must not be set/);
  });

  it("rejects composite memory panels with an unknown combine operator", () => {
    const error = validatePanelContent("number", {
      dataSource: {
        kind: "memory",
        combine: "median",
        sources: [{ namespace: "a", aggregation: "sum", field: "cost" }],
      },
      render: { kind: "number" },
    });
    expect(error).toMatch(/dataSource\.combine must be one of/);
  });

  it("rejects composite memory sources missing a field for non-count aggregation", () => {
    const error = validatePanelContent("number", {
      dataSource: {
        kind: "memory",
        combine: "sum",
        sources: [{ namespace: "a", aggregation: "sum" }],
      },
      render: { kind: "number" },
    });
    expect(error).toMatch(/dataSource\.sources\[0\]\.field is required/);
  });

  it("rejects composite memory panels with empty sources", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "memory", combine: "sum", sources: [] },
      render: { kind: "number" },
    });
    expect(error).toMatch(/dataSource\.sources must be a non-empty array/);
  });

  it("rejects composite memory panels when sources is null", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "memory", combine: "sum", sources: null },
      render: { kind: "number" },
    });
    expect(error).toMatch(/dataSource\.sources must be an array/);
  });

  it("rejects composite memory panels when sources is not an array", () => {
    const error = validatePanelContent("number", {
      dataSource: { kind: "memory", combine: "sum", sources: "oops" },
      render: { kind: "number", aggregation: "count" },
    });
    expect(error).toMatch(/dataSource\.sources must be an array/);
  });
});

describe("validatePanelContent — multi-number panels", () => {
  it("accepts a valid multi-number panel with mixed data sources", () => {
    expect(
      validatePanelContent("number", {
        title: "Pipeline KPIs",
        metrics: [
          { dataSource: { kind: "runs" }, render: { kind: "number", aggregation: "count", label: "Total runs" } },
          {
            dataSource: { kind: "memory", namespace: "costs" },
            render: { kind: "number", aggregation: "sum", field: "cost", label: "Total cost", prefix: "R$" },
          },
        ],
      }),
    ).toBeNull();
  });

  it("rejects multi-number panels with an empty metrics array", () => {
    const error = validatePanelContent("number", { metrics: [] });
    expect(error).toMatch(/metrics must be a non-empty array/);
  });

  it("rejects multi-number metrics with an unknown aggregation", () => {
    const error = validatePanelContent("number", {
      metrics: [{ dataSource: { kind: "runs" }, render: { kind: "number", aggregation: "median" } }],
    });
    expect(error).toMatch(/metrics\[0\]\.render\.aggregation must be one of/);
  });

  it("rejects multi-number metrics missing field for non-count aggregation", () => {
    const error = validatePanelContent("number", {
      metrics: [
        {
          dataSource: { kind: "memory", namespace: "costs" },
          render: { kind: "number", aggregation: "sum" },
        },
      ],
    });
    expect(error).toMatch(/metrics\[0\]\.render\.field is required/);
  });

  it("rejects multi-number metrics that use a composite data source", () => {
    const error = validatePanelContent("number", {
      metrics: [
        {
          dataSource: {
            kind: "memory",
            combine: "sum",
            sources: [{ namespace: "a", aggregation: "count" }],
          },
          render: { kind: "number", aggregation: "count" },
        },
      ],
    });
    expect(error).toMatch(/metrics\[0\]\.dataSource must be a single-source/);
  });

  it("rejects multi-number metrics with a bad render kind", () => {
    const error = validatePanelContent("number", {
      metrics: [{ dataSource: { kind: "runs" }, render: { kind: "table" } }],
    });
    expect(error).toMatch(/metrics\[0\]\.render\.kind must be "number"/);
  });

  it("rejects multi-number metrics with non-string prefix", () => {
    const error = validatePanelContent("number", {
      metrics: [{ dataSource: { kind: "runs" }, render: { kind: "number", aggregation: "count", prefix: 5 } }],
    });
    expect(error).toMatch(/metrics\[0\]\.render\.prefix must be a string/);
  });
});
