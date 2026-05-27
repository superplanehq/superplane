import { describe, expect, it } from "vitest";

import {
  coerceParameterValue,
  initialParameterValue,
  parameterDefaultValue,
  payloadForTemplateRun,
} from "./templatePayload";

describe("payloadForTemplateRun", () => {
  it("uses the template payload", () => {
    expect(
      payloadForTemplateRun({
        name: "t",
        payload: { a: 1 },
        parameters: [{ name: "a", type: "string", defaultString: "2" }],
      }),
    ).toEqual({ a: 1 });
  });

  it("returns empty object when payload is missing or invalid", () => {
    expect(payloadForTemplateRun({ name: "t", payload: undefined as unknown as Record<string, unknown> })).toEqual({});
  });
});

describe("coerceParameterValue", () => {
  it("coerces by parameter type", () => {
    expect(coerceParameterValue({ name: "n", type: "number" }, "42")).toBe(42);
    expect(coerceParameterValue({ name: "b", type: "boolean" }, "true")).toBe(true);
    expect(coerceParameterValue({ name: "s", type: "string" }, 1)).toBe("1");
  });
});

describe("parameterDefaultValue", () => {
  it("treats null and empty string as unset", () => {
    expect(parameterDefaultValue({ name: "a", type: "string", defaultString: null })).toBeUndefined();
    expect(parameterDefaultValue({ name: "a", type: "string", defaultString: "" })).toBeUndefined();
    expect(parameterDefaultValue({ name: "b", type: "boolean", defaultBoolean: null })).toBeUndefined();
  });
});

describe("initialParameterValue", () => {
  it("prefers payload values over parameter defaults", () => {
    expect(initialParameterValue({ name: "count", type: "number", defaultNumber: 1 }, { count: 5 })).toBe(5);
  });

  it("uses false when a boolean default is explicitly set to false", () => {
    expect(initialParameterValue({ name: "flag", type: "boolean", defaultBoolean: false }, {})).toBe(false);
  });
});
