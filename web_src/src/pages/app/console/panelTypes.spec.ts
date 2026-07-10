import { describe, expect, it } from "vitest";

import {
  CREATABLE_PANEL_TYPES,
  isPanelType,
  normalizeTablePanelContent,
  PANEL_TYPE_META,
  PANEL_TYPES,
  templateForPanelType,
  validatePanelContent,
} from "./panelTypes";

describe("PANEL_TYPES", () => {
  it("includes the seven supported types", () => {
    expect(PANEL_TYPES).toEqual(["markdown", "html", "node", "nodes", "table", "chart", "number"]);
  });

  it("isPanelType narrows to the union", () => {
    expect(isPanelType("markdown")).toBe(true);
    expect(isPanelType("html")).toBe(true);
    expect(isPanelType("node")).toBe(true);
    expect(isPanelType("nodes")).toBe(true);
    expect(isPanelType("timeline")).toBe(false);
    expect(isPanelType(42)).toBe(false);
  });
});

describe("CREATABLE_PANEL_TYPES", () => {
  it("hides the legacy `node` type from the Add Panel picker", () => {
    expect(CREATABLE_PANEL_TYPES).not.toContain("node");
  });

  it("still offers every other panel type", () => {
    for (const type of PANEL_TYPES) {
      if (type === "node") continue;
      expect(CREATABLE_PANEL_TYPES).toContain(type);
    }
  });

  it("uses the merged label/description for the nodes type", () => {
    expect(PANEL_TYPE_META.nodes.label).toBe("Nodes");
    expect(PANEL_TYPE_META.nodes.description.toLowerCase()).toContain("one or more");
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

describe("validatePanelContent — markdown and node", () => {
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

  it("accepts an optional string label on node panels", () => {
    expect(validatePanelContent("node", { node: "deploy-prod", label: "Ship" })).toBeNull();
    expect(validatePanelContent("node", { node: "deploy-prod", label: 42 })).toMatch(/content\.label must be a string/);
  });
});

describe("validatePanelContent — html", () => {
  it("accepts a valid html body", () => {
    expect(validatePanelContent("html", { body: "<p>Hello</p>" })).toBeNull();
  });

  it("rejects html body that is not a string", () => {
    expect(validatePanelContent("html", { body: 42 })).toMatch(/content\.body must be a string/);
  });

  it("rejects html title that is not a string", () => {
    expect(validatePanelContent("html", { title: 42 })).toMatch(/content\.title must be a string/);
  });

  it("validates html variables through the shared validator", () => {
    expect(
      validatePanelContent("html", {
        body: "<p>{{ x.field }}</p>",
        variables: [{ name: "1bad", source: { kind: "memory", namespace: "n" } }],
      }),
    ).toMatch(/content\.variables\[0\]\.name must be a valid identifier/);
  });

  it("accepts well-formed html content with memory variables", () => {
    expect(
      validatePanelContent("html", {
        title: "Latest {{ rec.name }}",
        body: '<div class="p-2"><strong>{{ rec.status }}</strong></div>',
        variables: [{ name: "rec", source: { kind: "memory", namespace: "deploys" } }],
      }),
    ).toBeNull();
  });
});

describe("validatePanelContent — table", () => {
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

  it("accepts row-style rules with a known tone", () => {
    expect(
      validatePanelContent("table", {
        dataSource: { kind: "memory", namespace: "env" },
        render: {
          kind: "table",
          columns: [],
          rowStyles: [
            { field: "status", op: "eq", value: "error", tone: "red-soft" },
            { field: "status", op: "eq", value: "deploying", tone: "orange-soft" },
          ],
        },
      }),
    ).toBeNull();
  });

  it("rejects row-style rules with an unknown tone", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [],
        rowStyles: [{ field: "status", op: "eq", value: "error", tone: "magenta" }],
      },
    });
    expect(error).toMatch(/render\.rowStyles\[0\]\.tone must be one of/);
  });

  it("rejects row-style rules with an unsupported op", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [],
        rowStyles: [{ field: "status", op: "regex", value: "err.*", tone: "red" }],
      },
    });
    expect(error).toMatch(/render\.rowStyles\[0\]\.op is not supported/);
  });

  it("rejects row-style rules with an empty field", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [],
        rowStyles: [{ field: "", op: "eq", value: "error", tone: "red" }],
      },
    });
    expect(error).toMatch(/render\.rowStyles\[0\]\.field must be a non-empty string/);
  });

  it("requires progressTarget when format is progress", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [{ field: "done", format: "progress" }],
      },
    });
    expect(error).toMatch(/render\.columns\[0\]\.progressTarget/);
  });

  it("rejects an unknown progressLabel value", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [{ field: "done", format: "progress", progressTarget: "total", progressLabel: "fraction" }],
      },
    });
    expect(error).toMatch(/render\.columns\[0\]\.progressLabel must be one of/);
  });

  it("accepts a well-formed progress column", () => {
    expect(
      validatePanelContent("table", {
        dataSource: { kind: "memory", namespace: "env" },
        render: {
          kind: "table",
          columns: [{ field: "done", format: "progress", progressTarget: "total", progressLabel: "number" }],
        },
      }),
    ).toBeNull();
  });
});

describe("normalizeTablePanelContent — data source limits", () => {
  it("preserves an explicit numeric limit for runs and executions", () => {
    const runs = normalizeTablePanelContent({
      dataSource: { kind: "runs", limit: 250 },
      render: { kind: "table", columns: [] },
    });
    expect(runs.dataSource).toEqual({ kind: "runs", limit: 250 });

    const executions = normalizeTablePanelContent({
      dataSource: { kind: "executions", node: "deploy", limit: 75 },
      render: { kind: "table", columns: [] },
    });
    expect(executions.dataSource).toEqual({ kind: "executions", node: "deploy", limit: 75 });
  });

  it("leaves limit undefined when not provided, so blank means 'load all'", () => {
    const runs = normalizeTablePanelContent({
      dataSource: { kind: "runs" },
      render: { kind: "table", columns: [] },
    });
    expect(runs.dataSource).toEqual({ kind: "runs", limit: undefined });

    const executions = normalizeTablePanelContent({
      dataSource: { kind: "executions" },
      render: { kind: "table", columns: [] },
    });
    expect(executions.dataSource).toEqual({ kind: "executions", node: undefined, limit: undefined });
  });

  it("drops a non-numeric limit rather than coercing it to a default", () => {
    const runs = normalizeTablePanelContent({
      dataSource: { kind: "runs", limit: "many" },
      render: { kind: "table", columns: [] },
    });
    expect(runs.dataSource).toEqual({ kind: "runs", limit: undefined });
  });
});

describe("normalizeTablePanelContent — rowStyles round-trip", () => {
  it("preserves avatar column options", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "checks" },
      render: {
        kind: "table",
        columns: [
          {
            field: "author",
            label: "Author",
            format: "avatar",
            avatarCommitterField: "committer",
          },
        ],
      },
    });
    expect(normalized.render.columns[0]).toEqual({
      field: "author",
      label: "Author",
      format: "avatar",
      avatarCommitterField: "committer",
    });
  });

  it("preserves progress column options and drops unknown progressLabel values", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [
          { field: "done", format: "progress", progressTarget: "total", progressLabel: "number" },
          { field: "score", format: "progress", progressTarget: "100", progressLabel: "fraction" },
        ],
      },
    });
    expect(normalized.render.columns[0]).toEqual({
      field: "done",
      format: "progress",
      progressTarget: "total",
      progressLabel: "number",
    });
    expect(normalized.render.columns[1]).toEqual({
      field: "score",
      format: "progress",
      progressTarget: "100",
      progressLabel: undefined,
    });
  });

  it("preserves valid rowStyles entries verbatim", () => {
    const normalized = normalizeTablePanelContent({
      title: "Envs",
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [{ field: "status" }],
        rowStyles: [
          { field: "status", op: "eq", value: "error", tone: "red-soft" },
          { field: "status", op: "eq", value: "deploying", tone: "orange-soft" },
        ],
      },
    });
    expect(normalized.render.rowStyles).toEqual([
      { field: "status", op: "eq", value: "error", tone: "red-soft" },
      { field: "status", op: "eq", value: "deploying", tone: "orange-soft" },
    ]);
  });

  it("drops entries with invalid field/op/tone and returns undefined when nothing remains", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "env" },
      render: {
        kind: "table",
        columns: [],
        rowStyles: [
          { field: "", op: "eq", value: "x", tone: "red" },
          { field: "status", op: "regex", value: "x", tone: "red" },
          { field: "status", op: "eq", value: "x", tone: "magenta" },
        ],
      },
    });
    expect(normalized.render.rowStyles).toBeUndefined();
  });

  it("returns undefined when rowStyles is missing entirely", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: { kind: "memory", namespace: "env" },
      render: { kind: "table", columns: [] },
    });
    expect(normalized.render.rowStyles).toBeUndefined();
  });
});

describe("validatePanelContent — chart", () => {
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

  it("accepts an optional seriesField on chart panels", () => {
    expect(
      validatePanelContent("chart", {
        dataSource: { kind: "memory", namespace: "costs" },
        render: {
          kind: "chart",
          type: "stacked-bar",
          xField: "date",
          seriesField: "service",
          series: [{ field: "cost_usd" }],
        },
      }),
    ).toBeNull();
  });

  it("rejects a non-string seriesField on chart panels", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "memory", namespace: "costs" },
      render: {
        kind: "chart",
        type: "stacked-bar",
        xField: "date",
        seriesField: 42,
        series: [{ field: "cost_usd" }],
      },
    });
    expect(error).toMatch(/render\.seriesField must be a string/);
  });
});

describe("validatePanelContent — nodes panels", () => {
  it("accepts a valid nodes panel with multiple entries", () => {
    expect(
      validatePanelContent("nodes", {
        title: "Key nodes",
        nodes: [
          { node: "deploy-prod", description: "Promotes the latest build" },
          { node: "rollback", label: "Rollback", showRun: true },
        ],
      }),
    ).toBeNull();
  });

  it("accepts a draft nodes panel with an empty nodes array", () => {
    expect(validatePanelContent("nodes", { nodes: [] })).toBeNull();
  });

  it("rejects nodes content where nodes is not an array", () => {
    expect(validatePanelContent("nodes", { nodes: {} })).toMatch(/content\.nodes must be an array/);
  });

  it("rejects nodes entries without a node reference", () => {
    expect(validatePanelContent("nodes", { nodes: [{ description: "missing" }] })).toMatch(
      /content\.nodes\[0\]\.node must be a non-empty string/,
    );
    expect(validatePanelContent("nodes", { nodes: [{ node: "" }] })).toMatch(
      /content\.nodes\[0\]\.node must be a non-empty string/,
    );
  });

  it("rejects nodes entries with a non-string description", () => {
    expect(validatePanelContent("nodes", { nodes: [{ node: "deploy", description: 42 }] })).toMatch(
      /content\.nodes\[0\]\.description must be a string/,
    );
  });

  it("rejects nodes entries with a non-boolean showRun", () => {
    expect(validatePanelContent("nodes", { nodes: [{ node: "deploy", showRun: "yes" }] })).toMatch(
      /content\.nodes\[0\]\.showRun must be a boolean/,
    );
  });
});

describe("validatePanelContent — chart series and legend", () => {
  it("accepts chart series with format, prefix, and suffix", () => {
    expect(
      validatePanelContent("chart", {
        dataSource: { kind: "executions" },
        render: {
          kind: "chart",
          type: "bar",
          xField: "service",
          series: [{ field: "cost", label: "Cost", format: "number", prefix: "$", suffix: " /mo" }],
        },
      }),
    ).toBeNull();
  });

  it("rejects chart series with a non-string prefix", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "executions" },
      render: {
        kind: "chart",
        type: "bar",
        xField: "service",
        series: [{ field: "cost", prefix: 42 }],
      },
    });
    expect(error).toMatch(/render\.series\[0\]\.prefix must be a string/);
  });

  it("accepts chart legend modes", () => {
    for (const legend of ["auto", "show", "hide"] as const) {
      expect(
        validatePanelContent("chart", {
          dataSource: { kind: "executions" },
          render: { kind: "chart", type: "bar", xField: "x", series: [{}], legend },
        }),
      ).toBeNull();
    }
  });

  it("rejects unknown legend modes", () => {
    const error = validatePanelContent("chart", {
      dataSource: { kind: "executions" },
      render: { kind: "chart", type: "bar", xField: "x", series: [{}], legend: "bogus" },
    });
    expect(error).toMatch(/render\.legend must be one of/);
  });
});
