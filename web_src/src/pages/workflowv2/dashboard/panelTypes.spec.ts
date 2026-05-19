import { describe, expect, it } from "vitest";

import { isPanelType, PANEL_TYPES, templateForPanelType, validatePanelContent } from "./panelTypes";

describe("PANEL_TYPES", () => {
  it("includes the five supported types", () => {
    expect(PANEL_TYPES).toEqual(["markdown", "node", "table", "chart", "number"]);
  });

  it("isPanelType narrows to the union", () => {
    expect(isPanelType("markdown")).toBe(true);
    expect(isPanelType("node")).toBe(true);
    expect(isPanelType("timeline")).toBe(false);
    expect(isPanelType(42)).toBe(false);
  });
});

describe("templateForPanelType", () => {
  it("produces a valid template for each kind", () => {
    for (const type of PANEL_TYPES) {
      const tpl = templateForPanelType(type, "Test title");
      expect(validatePanelContent(type, tpl), `${type} template should validate`).toBeNull();
    }
  });

  it("seeds the title when provided", () => {
    expect(templateForPanelType("markdown", "Hello").title).toBe("Hello");
  });

  it("defaults table panels to memory with empty columns for discovery", () => {
    const tpl = templateForPanelType("table") as {
      dataSource: { kind: string };
      render: { columns: unknown[] };
    };
    expect(tpl.dataSource.kind).toBe("memory");
    expect(tpl.render.columns).toEqual([]);
  });

  it("uses runs as the default data source for number panels", () => {
    const tpl = templateForPanelType("number") as { dataSource: { kind: string }; render: { aggregation: string } };
    expect(tpl.dataSource.kind).toBe("runs");
    expect(tpl.render.aggregation).toBe("count");
  });

  it("defaults chart panels to count rows when no series field is set", () => {
    const tpl = templateForPanelType("chart") as { render: { series: Array<{ field?: string; label?: string }> } };
    expect(tpl.render.series).toEqual([{ label: "Count" }]);
  });
});

describe("validatePanelContent", () => {
  it("accepts a valid markdown body", () => {
    expect(validatePanelContent("markdown", { body: "# Hi" })).toBeNull();
  });

  it("rejects markdown body that is not a string", () => {
    expect(validatePanelContent("markdown", { body: 42 })).toMatch(/content\.body must be a string/);
  });

  it("requires the node id field on node panels to be a string", () => {
    expect(validatePanelContent("node", { node: 42 })).toMatch(/content\.node must be a string/);
    expect(validatePanelContent("node", { node: "" })).toBeNull();
    expect(validatePanelContent("node", { node: "deploy-prod" })).toBeNull();
  });

  it("requires a data source on table panels", () => {
    expect(validatePanelContent("table", {})).toMatch(/dataSource must be an object/);
  });

  it("allows an empty columns array on table panels", () => {
    expect(
      validatePanelContent("table", {
        dataSource: { kind: "memory", namespace: "env" },
        render: { kind: "table", columns: [] },
      }),
    ).toBeNull();
  });

  it("allows draft table panels with an empty memory namespace", () => {
    expect(
      validatePanelContent("table", {
        dataSource: { kind: "memory", namespace: "" },
        render: { kind: "table", columns: [] },
      }),
    ).toBeNull();
  });

  it("rejects table columns without a field", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: { kind: "table", columns: [{ label: "Missing field" }] },
    });
    expect(error).toMatch(/render\.columns\[0\]\.field/);
  });

  it("rejects unsupported table filter operators", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: { kind: "table", columns: [], where: [{ field: "status", op: "regex" }] },
    });
    expect(error).toMatch(/render\.where\[0\]\.op is not supported/);
  });

  it("rejects trigger row actions without a node", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: { kind: "table", columns: [], rowActions: [{ kind: "trigger", label: "Run" }] },
    });
    expect(error).toMatch(/render\.rowActions\[0\]\.node/);
  });

  it("requires a known chart type", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "executions" },
      render: { kind: "chart", type: "bogus", xField: "x", series: [{}] },
    });
    expect(error).toMatch(/render\.type must be one of/);
  });

  it("rejects non-numeric data source limits", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "executions", limit: "many" },
      render: { kind: "chart", type: "bar", xField: "x", series: [{}] },
    });
    expect(error).toMatch(/dataSource\.limit must be a number/);
  });

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
});
