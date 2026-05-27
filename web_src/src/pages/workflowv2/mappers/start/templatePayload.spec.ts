import { describe, expect, it } from "vitest";

import {
  coerceParameterValue,
  initialParameterValue,
  parameterDefaultValue,
  parameterDisplayLabel,
  payloadForTemplateRun,
  payloadRecordForParameters,
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

describe("payloadRecordForParameters", () => {
  it("returns objects as-is", () => {
    expect(payloadRecordForParameters({ a: 1 })).toEqual({ a: 1 });
  });

  it("parses JSON strings and returns empty object on failure", () => {
    expect(payloadRecordForParameters('{"a":1}')).toEqual({ a: 1 });
    expect(payloadRecordForParameters("not json")).toEqual({});
  });
});

describe("parameterDisplayLabel", () => {
  it("uses title when set", () => {
    expect(parameterDisplayLabel({ name: "msg", title: "Message", type: "string" })).toBe("Message");
  });

  it("falls back to name when title is missing or blank", () => {
    expect(parameterDisplayLabel({ name: "msg", type: "string" })).toBe("msg");
    expect(parameterDisplayLabel({ name: "msg", title: "  ", type: "string" })).toBe("msg");
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
  it("uses configured parameter defaults", () => {
    expect(initialParameterValue({ name: "count", type: "number", defaultNumber: 1 })).toBe(1);
    expect(initialParameterValue({ name: "redundancy", type: "string", defaultString: "dual" })).toBe("dual");
  });

  it("uses false when a boolean default is explicitly set to false", () => {
    expect(initialParameterValue({ name: "flag", type: "boolean", defaultBoolean: false })).toBe(false);
  });
});
