import { describe, expect, it } from "vitest";

import { normalizeTablePanelContent, validatePanelContent } from "./panelTypes";

describe("runs data source — status and trigger filters", () => {
  it("accepts valid status and trigger filter arrays", () => {
    expect(
      validatePanelContent("table", {
        dataSource: {
          kind: "runs",
          limit: 100,
          statuses: ["running", "failed"],
          triggers: ["deploy", "9f2c1e5a-1234-4b0c-9c0d-abcdefabcdef"],
        },
        render: { kind: "table", columns: [{ field: "status" }] },
      }),
    ).toBeNull();
  });

  it("treats missing / empty filters as valid (empty means all)", () => {
    expect(
      validatePanelContent("table", {
        dataSource: { kind: "runs", statuses: [], triggers: [] },
        render: { kind: "table", columns: [{ field: "status" }] },
      }),
    ).toBeNull();
  });

  it("rejects an unknown status value", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "runs", statuses: ["running", "flaky"] },
      render: { kind: "table", columns: [{ field: "status" }] },
    });
    expect(error).toMatch(/statuses\[1\]/);
    expect(error).toMatch(/running, passed, failed, cancelled/);
  });

  it("rejects a non-string trigger entry", () => {
    const error = validatePanelContent("table", {
      dataSource: { kind: "runs", triggers: ["deploy", 42] },
      render: { kind: "table", columns: [{ field: "status" }] },
    });
    expect(error).toMatch(/triggers\[1\]/);
  });

  it("preserves and dedupes statuses / triggers through normalize", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: {
        kind: "runs",
        statuses: ["failed", "failed", "passed", "running"],
        triggers: [" deploy ", "deploy", "release"],
      },
      render: { kind: "table", columns: [] },
    });
    expect(normalized.dataSource).toEqual({
      kind: "runs",
      limit: undefined,
      statuses: ["failed", "passed", "running"],
      triggers: ["deploy", "release"],
    });
  });

  it("drops filter fields when normalize would leave them empty", () => {
    const normalized = normalizeTablePanelContent({
      dataSource: {
        kind: "runs",
        statuses: ["unknown-status"],
        triggers: ["", "   "],
      },
      render: { kind: "table", columns: [] },
    });
    expect(normalized.dataSource).toEqual({ kind: "runs", limit: undefined });
  });
});
