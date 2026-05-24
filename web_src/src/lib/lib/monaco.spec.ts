import { describe, expect, it } from "vitest";
import { coerceMonacoValue } from "./monaco";

describe("coerceMonacoValue", () => {
  it("returns strings unchanged", () => {
    expect(coerceMonacoValue("hello")).toBe("hello");
    expect(coerceMonacoValue("")).toBe("");
  });

  it("converts null/undefined to an empty string", () => {
    expect(coerceMonacoValue(null)).toBe("");
    expect(coerceMonacoValue(undefined)).toBe("");
  });

  it("stringifies primitives", () => {
    expect(coerceMonacoValue(42)).toBe("42");
    expect(coerceMonacoValue(true)).toBe("true");
    expect(coerceMonacoValue(false)).toBe("false");
    expect(coerceMonacoValue(0)).toBe("0");
  });

  it("JSON-stringifies plain objects so Monaco never receives a non-string", () => {
    expect(coerceMonacoValue({ foo: "bar" })).toBe(JSON.stringify({ foo: "bar" }, null, 2));
    expect(coerceMonacoValue({})).toBe("{}");
    expect(coerceMonacoValue(["a", "b"])).toBe(JSON.stringify(["a", "b"], null, 2));
  });

  it("falls back to an empty string when serialization fails (e.g. circular refs)", () => {
    const circular: Record<string, unknown> = {};
    circular.self = circular;
    expect(coerceMonacoValue(circular)).toBe("");
  });
});
