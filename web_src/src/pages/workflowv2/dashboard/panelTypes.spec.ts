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

  it("creates a non-empty default columns list for table", () => {
    const tpl = templateForPanelType("table") as { render: { columns: unknown[] } };
    expect(tpl.render.columns.length).toBeGreaterThan(0);
  });

  it("uses runs as the default data source for number panels", () => {
    const tpl = templateForPanelType("number") as { dataSource: { kind: string }; render: { aggregation: string } };
    expect(tpl.dataSource.kind).toBe("runs");
    expect(tpl.render.aggregation).toBe("count");
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

  it("requires a non-empty columns array on table panels", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "executions" },
      render: { kind: "table", columns: [] },
    });
    expect(error).toMatch(/render\.columns must be a non-empty array/);
  });

  it("requires a known chart type", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "executions" },
      render: { kind: "chart", type: "bogus", xField: "x", series: [{}] },
    });
    expect(error).toMatch(/render\.type must be one of/);
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

  it("memory data sources require namespace", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory" },
      render: { kind: "table", columns: [{ field: "x" }] },
    });
    expect(error).toMatch(/dataSource\.namespace must be a non-empty string/);
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
