import { describe, expect, it } from "vitest";

import { validateMarkdownVariables } from "./markdownVariables";

describe("validateMarkdownVariables", () => {
  it("accepts an undefined/null variables array", () => {
    expect(validateMarkdownVariables(undefined)).toBeNull();
    expect(validateMarkdownVariables(null)).toBeNull();
  });

  it("accepts an empty array", () => {
    expect(validateMarkdownVariables([])).toBeNull();
  });

  it("rejects a non-array variables value", () => {
    expect(validateMarkdownVariables({})).toMatch(/array/);
    expect(validateMarkdownVariables("nope")).toMatch(/array/);
  });

  it("requires a valid identifier name", () => {
    expect(validateMarkdownVariables([{ name: "1bad", source: { kind: "run", select: "latest" } }])).toMatch(
      /valid identifier/,
    );
    expect(validateMarkdownVariables([{ name: "bad-name", source: { kind: "run", select: "latest" } }])).toMatch(
      /valid identifier/,
    );
    expect(validateMarkdownVariables([{ name: "validName_1", source: { kind: "run", select: "latest" } }])).toBeNull();
  });

  it("rejects duplicate names", () => {
    const error = validateMarkdownVariables([
      { name: "x", source: { kind: "run", select: "latest" } },
      { name: "x", source: { kind: "run", select: "latest_passed" } },
    ]);
    expect(error).toMatch(/duplicated/);
  });

  it("rejects unknown source kinds", () => {
    expect(validateMarkdownVariables([{ name: "bad", source: { kind: "executions" } as unknown as never }])).toMatch(
      /memory.*run|run.*memory/,
    );
  });

  it("validates memory source shape", () => {
    expect(
      validateMarkdownVariables([{ name: "ok", source: { kind: "memory", namespace: "recipes", direction: "asc" } }]),
    ).toBeNull();
    expect(validateMarkdownVariables([{ name: "bad", source: { kind: "memory", namespace: "" } }])).toMatch(
      /namespace/,
    );
    expect(
      validateMarkdownVariables([
        { name: "bad", source: { kind: "memory", namespace: "n", direction: "sideways" as unknown as never } },
      ]),
    ).toMatch(/direction/);
  });

  it("validates memory matches", () => {
    expect(
      validateMarkdownVariables([
        {
          name: "ok",
          source: { kind: "memory", namespace: "n", matches: [{ field: "status", value: "x" }] },
        },
      ]),
    ).toBeNull();
    expect(
      validateMarkdownVariables([
        {
          name: "bad",
          source: { kind: "memory", namespace: "n", matches: [{ field: "", value: "x" }] as unknown as never },
        },
      ]),
    ).toMatch(/field/);
  });

  it("validates run select values", () => {
    expect(validateMarkdownVariables([{ name: "ok", source: { kind: "run", select: "latest_failed" } }])).toBeNull();
    expect(
      validateMarkdownVariables([{ name: "bad", source: { kind: "run", select: "first" as unknown as never } }]),
    ).toMatch(/select/);
  });
});
