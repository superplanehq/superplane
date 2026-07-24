import { describe, expect, it } from "vitest";

import { normalizeRunParameterDefinitions } from "./runParameters";

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
