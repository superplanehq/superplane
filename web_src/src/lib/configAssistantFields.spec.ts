import { describe, expect, it } from "vitest";
import {
  buildConfigAssistantFieldContext,
  isConfigAssistantSupportedField,
  parseAssistantBooleanValue,
  parseAssistantMultiSelectValue,
  parseAssistantNumberValue,
} from "./configAssistantFields";

describe("isConfigAssistantSupportedField", () => {
  it("allows expression when not sensitive", () => {
    expect(isConfigAssistantSupportedField({ type: "expression", sensitive: false })).toBe(true);
  });

  it("rejects sensitive regardless of type", () => {
    expect(isConfigAssistantSupportedField({ type: "expression", sensitive: true })).toBe(false);
    expect(isConfigAssistantSupportedField({ type: "string", sensitive: true })).toBe(false);
  });

  it("rejects unknown types", () => {
    expect(isConfigAssistantSupportedField({ type: "secret-key", sensitive: false })).toBe(false);
    expect(isConfigAssistantSupportedField({ type: undefined, sensitive: false })).toBe(false);
  });

  it("allows listed types", () => {
    expect(isConfigAssistantSupportedField({ type: "string" })).toBe(true);
    expect(isConfigAssistantSupportedField({ type: "url" })).toBe(true);
  });

  it("rejects select, multi-select, number, and boolean", () => {
    expect(isConfigAssistantSupportedField({ type: "select" })).toBe(false);
    expect(isConfigAssistantSupportedField({ type: "multi-select" })).toBe(false);
    expect(isConfigAssistantSupportedField({ type: "number" })).toBe(false);
    expect(isConfigAssistantSupportedField({ type: "boolean" })).toBe(false);
  });

  it("allows single integration-resource, not multi", () => {
    expect(
      isConfigAssistantSupportedField({
        type: "integration-resource",
        typeOptions: { resource: { type: "repo", multi: false } },
      }),
    ).toBe(true);
    expect(
      isConfigAssistantSupportedField({
        type: "integration-resource",
        typeOptions: { resource: { type: "label", multi: true } },
      }),
    ).toBe(false);
  });
});

describe("parseAssistantMultiSelectValue", () => {
  it("parses JSON array", () => {
    expect(parseAssistantMultiSelectValue('["a","b"]')).toEqual(["a", "b"]);
  });

  it("parses comma-separated", () => {
    expect(parseAssistantMultiSelectValue("a, b")).toEqual(["a", "b"]);
  });
});

describe("parseAssistantNumberValue", () => {
  it("validates range", () => {
    expect(parseAssistantNumberValue("5", 0, 10)).toEqual({ ok: true, value: 5 });
    expect(parseAssistantNumberValue("11", 0, 10).ok).toBe(false);
  });
});

describe("parseAssistantBooleanValue", () => {
  it("parses common tokens", () => {
    expect(parseAssistantBooleanValue("true")).toBe(true);
    expect(parseAssistantBooleanValue("no")).toBe(false);
    expect(parseAssistantBooleanValue("maybe")).toBe(null);
  });
});

describe("buildConfigAssistantFieldContext", () => {
  it("includes typeOptions when present", () => {
    const ctx = buildConfigAssistantFieldContext({
      name: "x",
      type: "select",
      typeOptions: { select: { options: [{ label: "A", value: "a" }] } },
    });
    expect(ctx.typeOptions).toBeDefined();
  });
});
