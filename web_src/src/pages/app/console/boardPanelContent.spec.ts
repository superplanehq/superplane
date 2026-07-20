import { describe, expect, it } from "vitest";

import { normalizeBoardPanelContent, templateForBoardPanel, validateBoardContent } from "./boardPanelContent";

const validBoardBody = () => ({
  title: "Pipeline",
  dataSource: { kind: "memory", namespace: "factory_tasks" },
  render: {
    kind: "board",
    groupBy: "status",
    lanes: [{ value: "Todo" }, { value: "Done", color: "green" }],
    card: { titleField: "title" },
  },
});

describe("templateForBoardPanel", () => {
  it("returns a body that passes the FE validator", () => {
    const template = templateForBoardPanel("Pipeline");
    expect(validateBoardContent(template)).toBeNull();
    expect(template.title).toBe("Pipeline");
  });

  it("seeds a memory data source with an empty namespace so the editor discovers rows", () => {
    const template = templateForBoardPanel();
    expect(template.dataSource).toEqual({ kind: "memory", namespace: "" });
  });
});

describe("validateBoardContent — happy path", () => {
  it("accepts a fully-specified board body", () => {
    const body = {
      ...validBoardBody(),
      render: {
        ...validBoardBody().render,
        otherLane: true,
        card: {
          titleField: "title",
          fields: [
            { field: "pr_url", format: "link", label: "PR" },
            { field: "updatedAt", format: "relative" },
          ],
        },
        where: [{ field: "archived", op: "not_exists" }],
        sort: { field: "updatedAt", order: "desc" },
        rowActions: [{ kind: "trigger", node: "start", hook: "run" }],
        emptyMessage: "Nothing to show yet",
      },
    };
    expect(validateBoardContent(body)).toBeNull();
  });
});

describe("validateBoardContent — rejections", () => {
  it("requires content to be an object", () => {
    expect(validateBoardContent(null)).toMatch(/content must be an object/);
  });

  it("requires a non-empty groupBy", () => {
    const body = { ...validBoardBody(), render: { ...validBoardBody().render, groupBy: "" } };
    expect(validateBoardContent(body)).toMatch(/render\.groupBy must be a non-empty string/);
  });

  it("requires at least one lane", () => {
    const body = { ...validBoardBody(), render: { ...validBoardBody().render, lanes: [] } };
    expect(validateBoardContent(body)).toMatch(/render\.lanes must be a non-empty array/);
  });

  it("rejects blank lane values", () => {
    const body = { ...validBoardBody(), render: { ...validBoardBody().render, lanes: [{ value: "  " }] } };
    expect(validateBoardContent(body)).toMatch(/render\.lanes\[0\]\.value/);
  });

  it("rejects unknown lane colors", () => {
    const body = {
      ...validBoardBody(),
      render: { ...validBoardBody().render, lanes: [{ value: "Todo", color: "fuchsia" }] },
    };
    expect(validateBoardContent(body)).toMatch(/render\.lanes\[0\]\.color must be one of/);
  });

  it("requires a card with a non-empty titleField", () => {
    const body = { ...validBoardBody(), render: { ...validBoardBody().render, card: { titleField: "" } } };
    expect(validateBoardContent(body)).toMatch(/render\.card\.titleField must be a non-empty string/);
  });

  it("rejects non-boolean otherLane", () => {
    const body = { ...validBoardBody(), render: { ...validBoardBody().render, otherLane: "yes" } };
    expect(validateBoardContent(body)).toMatch(/render\.otherLane must be a boolean/);
  });

  it("rejects a card field with a blank field name", () => {
    const body = {
      ...validBoardBody(),
      render: {
        ...validBoardBody().render,
        card: { titleField: "title", fields: [{ field: "" }] },
      },
    };
    expect(validateBoardContent(body)).toMatch(/render\.card\.fields\[0\]\.field/);
  });

  it("rejects an unrecognized where op", () => {
    const body = {
      ...validBoardBody(),
      render: { ...validBoardBody().render, where: [{ field: "s", op: "matches" }] },
    };
    expect(validateBoardContent(body)).toMatch(/render\.where\[0\]\.op/);
  });

  it("rejects a rowAction that is not a trigger", () => {
    const body = {
      ...validBoardBody(),
      render: { ...validBoardBody().render, rowActions: [{ kind: "http" }] },
    };
    expect(validateBoardContent(body)).toMatch(/render\.rowActions\[0\] must be a trigger action/);
  });
});

describe("normalizeBoardPanelContent", () => {
  it("drops unknown fields on lanes and card fields", () => {
    const normalized = normalizeBoardPanelContent({
      title: "P",
      dataSource: { kind: "memory", namespace: "tasks" },
      render: {
        kind: "board",
        groupBy: "status",
        lanes: [
          { value: "Todo", nonsense: true, color: "gray" },
          { value: 42 },
          { value: "Done", color: "not-a-color" },
        ],
        card: {
          titleField: "title",
          fields: [{ field: "pr_url", format: "link", extras: 1 }, { field: "" }, "not-an-object"],
        },
      },
    });
    expect(normalized.render.lanes).toEqual([
      { value: "Todo", label: undefined, color: "gray" },
      { value: "Done", label: undefined, color: undefined },
    ]);
    expect(normalized.render.card.fields).toEqual([
      { field: "pr_url", format: "link", label: undefined, show: undefined, href: undefined },
    ]);
  });

  it("drops empty where / rowActions so persisted YAML stays clean", () => {
    const normalized = normalizeBoardPanelContent({
      dataSource: { kind: "memory", namespace: "tasks" },
      render: {
        kind: "board",
        groupBy: "status",
        lanes: [{ value: "Todo" }],
        card: { titleField: "title" },
        where: [{ field: "", op: "eq" }],
        rowActions: [{ kind: "http" }],
      },
    });
    expect(normalized.render.where).toBeUndefined();
    expect(normalized.render.rowActions).toBeUndefined();
  });

  it("normalizes sort with a valid order and clears blank field", () => {
    const normalized = normalizeBoardPanelContent({
      dataSource: { kind: "memory", namespace: "tasks" },
      render: {
        kind: "board",
        groupBy: "status",
        lanes: [{ value: "Todo" }],
        card: { titleField: "title" },
        sort: { field: "  ", order: "asc" },
      },
    });
    expect(normalized.render.sort).toBeUndefined();
  });
});
