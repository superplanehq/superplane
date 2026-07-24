import { describe, expect, it } from "vitest";

import { coerceRunParameterValues, getSyncedRunParameterValues, normalizeRunParameterDefinitions } from "./runParameters";

describe("coerceRunParameterValues", () => {
  it("returns an empty object for missing or invalid input", () => {
    expect(coerceRunParameterValues(undefined)).toEqual({});
    expect(coerceRunParameterValues(null)).toEqual({});
    expect(coerceRunParameterValues([])).toEqual({});
    expect(coerceRunParameterValues("value")).toEqual({});
  });

  it("returns object values unchanged", () => {
    expect(coerceRunParameterValues({ message: "hello" })).toEqual({ message: "hello" });
  });
});

describe("getSyncedRunParameterValues", () => {
  it("clears values when no parameter definitions exist", () => {
    expect(getSyncedRunParameterValues({ custom: true }, [])).toEqual({});
    expect(getSyncedRunParameterValues({}, [])).toBeNull();
  });

  it("removes keys that are not defined on the target trigger", () => {
    const definitions = normalizeRunParameterDefinitions([{ type: "string", name: "message" }]);

    expect(getSyncedRunParameterValues({ message: "hello", obsolete: true }, definitions)).toEqual({
      message: "hello",
    });
    expect(getSyncedRunParameterValues({ message: "hello" }, definitions)).toBeNull();
  });
});

describe("normalizeRunParameterDefinitions", () => {
  it("returns an empty list for missing or invalid input", () => {
    expect(normalizeRunParameterDefinitions(undefined)).toEqual([]);
    expect(normalizeRunParameterDefinitions(null)).toEqual([]);
    expect(normalizeRunParameterDefinitions({})).toEqual([]);
    expect(normalizeRunParameterDefinitions([])).toEqual([]);
  });

  it("maps onRun parameter definitions to configuration fields", () => {
    expect(
      normalizeRunParameterDefinitions([
        {
          type: "string",
          name: "message",
          label: "Message",
          description: "The message to send",
          required: true,
          default: "hello",
        },
        {
          type: "number",
          name: "count",
          required: false,
        },
      ]),
    ).toEqual([
      {
        name: "message",
        label: "Message",
        type: "string",
        description: "The message to send",
        required: true,
        defaultValue: "hello",
        typeOptions: undefined,
      },
      {
        name: "count",
        label: "count",
        type: "number",
        description: undefined,
        required: false,
        defaultValue: undefined,
        typeOptions: undefined,
      },
    ]);
  });

  it("prefers label over name for display", () => {
    expect(
      normalizeRunParameterDefinitions([
        {
          type: "boolean",
          name: "is_active",
          label: "Is active",
        },
      ]),
    ).toEqual([
      {
        name: "is_active",
        label: "Is active",
        type: "boolean",
        description: undefined,
        required: false,
        defaultValue: undefined,
        typeOptions: undefined,
      },
    ]);
  });

  it("skips entries without a name", () => {
    expect(
      normalizeRunParameterDefinitions([
        { type: "string", name: "valid" },
        { type: "string", name: "  " },
        { type: "string" },
      ]),
    ).toEqual([
      {
        name: "valid",
        label: "valid",
        type: "string",
        description: undefined,
        required: false,
        defaultValue: undefined,
        typeOptions: undefined,
      },
    ]);
  });
});
