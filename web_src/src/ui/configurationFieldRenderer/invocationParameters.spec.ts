import { describe, expect, it } from "vitest";

import { normalizeInvocationParameterDefinitions } from "./invocationParameters";

describe("normalizeInvocationParameterDefinitions", () => {
  it("returns an empty list for missing or invalid input", () => {
    expect(normalizeInvocationParameterDefinitions(undefined)).toEqual([]);
    expect(normalizeInvocationParameterDefinitions(null)).toEqual([]);
    expect(normalizeInvocationParameterDefinitions({})).toEqual([]);
    expect(normalizeInvocationParameterDefinitions([])).toEqual([]);
  });

  it("maps onInvoke parameter definitions to configuration fields", () => {
    expect(
      normalizeInvocationParameterDefinitions([
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
      normalizeInvocationParameterDefinitions([
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
      normalizeInvocationParameterDefinitions([
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
