import { describe, expect, it } from "vitest";

import {
  buildParameterFormPayload,
  coerceParameterValue,
  initialParameterValue,
  isValidSelectParameterValue,
  parameterDefaultValue,
  parameterDisplayLabel,
  parameterInputPlaceholder,
  parseJsonEventPayload,
  parameterPlaceholder,
  startRunModalTitle,
  payloadForTemplateRun,
  payloadRecordForParameters,
  selectOptionValues,
  validateSubmittedParameterValue,
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

  it("does not duplicate when title matches name ignoring case", () => {
    expect(parameterDisplayLabel({ name: "Name", title: "Name", type: "string" })).toBe("Name");
    expect(parameterDisplayLabel({ name: "name", title: "Name", type: "string" })).toBe("Name");
  });
});

describe("startRunModalTitle", () => {
  it("prefers the node name over the template name", () => {
    expect(startRunModalTitle("Start OpenClaw server", "run")).toBe("Start OpenClaw server");
  });

  it("falls back to the template name when the node is unnamed", () => {
    expect(startRunModalTitle("", "Hello World")).toBe("Hello World");
    expect(startRunModalTitle(undefined, "Hello World")).toBe("Hello World");
  });

  it("falls back to Run when both names are empty", () => {
    expect(startRunModalTitle("", "")).toBe("Run");
  });
});

describe("parameterPlaceholder", () => {
  it("returns trimmed placeholder when set", () => {
    expect(parameterPlaceholder({ name: "agent", placeholder: "  Name of the openclaw agent  ", type: "string" })).toBe(
      "Name of the openclaw agent",
    );
  });

  it("returns empty string when placeholder is missing or blank", () => {
    expect(parameterPlaceholder({ name: "agent", type: "string" })).toBe("");
    expect(parameterPlaceholder({ name: "agent", placeholder: "  ", type: "string" })).toBe("");
  });
});

describe("parameterInputPlaceholder", () => {
  it("omits placeholders that repeat the field label", () => {
    expect(
      parameterInputPlaceholder({ name: "name", title: "Name", placeholder: "Name", type: "string" }, "Name"),
    ).toBe(undefined);
  });

  it("keeps distinct placeholders", () => {
    expect(
      parameterInputPlaceholder({ name: "agent", placeholder: "Name of the openclaw agent", type: "string" }, "Agent"),
    ).toBe("Name of the openclaw agent");
  });
});

describe("coerceParameterValue", () => {
  it("coerces by parameter type", () => {
    expect(coerceParameterValue({ name: "n", type: "number" }, "42")).toBe(42);
    expect(coerceParameterValue({ name: "b", type: "boolean" }, "true")).toBe(true);
    expect(coerceParameterValue({ name: "s", type: "string" }, 1)).toBe("1");
    expect(coerceParameterValue({ name: "p", type: "select" }, "openai")).toBe("openai");
  });

  it("treats text values as strings and preserves newlines", () => {
    expect(coerceParameterValue({ name: "prompt", type: "text" }, "line 1\nline 2")).toBe("line 1\nline 2");
    expect(coerceParameterValue({ name: "prompt", type: "text" }, null)).toBe("");
  });
});

describe("selectOptionValues", () => {
  it("returns option values for select parameters", () => {
    expect(
      selectOptionValues({
        name: "provider",
        type: "select",
        options: [
          { label: "OpenAI", value: "openai" },
          { label: "Anthropic", value: "anthropic" },
        ],
      }),
    ).toEqual(["openai", "anthropic"]);
  });
});

describe("isValidSelectParameterValue", () => {
  const param = {
    name: "provider",
    type: "select" as const,
    options: [
      { label: "OpenAI", value: "openai" },
      { label: "Anthropic", value: "anthropic" },
    ],
  };

  it("accepts configured option values", () => {
    expect(isValidSelectParameterValue(param, "openai")).toBe(true);
    expect(isValidSelectParameterValue(param, "anthropic")).toBe(true);
  });

  it("rejects values outside configured options", () => {
    expect(isValidSelectParameterValue(param, "other")).toBe(false);
  });
});

describe("parameterDefaultValue", () => {
  it("treats null and empty string as unset", () => {
    expect(parameterDefaultValue({ name: "a", type: "string", defaultString: null })).toBeUndefined();
    expect(parameterDefaultValue({ name: "a", type: "string", defaultString: "" })).toBeUndefined();
    expect(parameterDefaultValue({ name: "b", type: "boolean", defaultBoolean: null })).toBeUndefined();
  });

  it("reads defaultString for text parameters", () => {
    expect(parameterDefaultValue({ name: "prompt", type: "text", defaultString: "hello" })).toBe("hello");
    expect(parameterDefaultValue({ name: "prompt", type: "text", defaultString: "" })).toBeUndefined();
  });
});

describe("initialParameterValue", () => {
  it("uses configured parameter defaults", () => {
    expect(initialParameterValue({ name: "count", type: "number", defaultNumber: 1 })).toBe(1);
    expect(initialParameterValue({ name: "redundancy", type: "string", defaultString: "dual" })).toBe("dual");
    expect(initialParameterValue({ name: "prompt", type: "text", defaultString: "hi\nthere" })).toBe("hi\nthere");
    expect(initialParameterValue({ name: "prompt", type: "text" })).toBe("");
    expect(
      initialParameterValue({
        name: "provider",
        type: "select",
        defaultString: "anthropic",
        options: [
          { label: "OpenAI", value: "openai" },
          { label: "Anthropic", value: "anthropic" },
        ],
      }),
    ).toBe("anthropic");
  });

  it("uses false when a boolean default is explicitly set to false", () => {
    expect(initialParameterValue({ name: "flag", type: "boolean", defaultBoolean: false })).toBe(false);
  });

  it("uses the first option value when select has no default", () => {
    expect(
      initialParameterValue({
        name: "provider",
        type: "select",
        options: [
          { label: "OpenAI", value: "openai" },
          { label: "Anthropic", value: "anthropic" },
        ],
      }),
    ).toBe("openai");
  });
});

describe("validateSubmittedParameterValue", () => {
  const selectParam = {
    name: "provider",
    type: "select" as const,
    options: [
      { label: "OpenAI", value: "openai" },
      { label: "Anthropic", value: "anthropic" },
    ],
  };

  it("returns null for valid values", () => {
    expect(validateSubmittedParameterValue(selectParam, "openai")).toBeNull();
    expect(validateSubmittedParameterValue({ name: "count", type: "number" }, 1)).toBeNull();
  });

  it("returns errors for invalid number and select values", () => {
    expect(validateSubmittedParameterValue({ name: "count", type: "number" }, Number.NaN)).toContain("valid number");
    expect(validateSubmittedParameterValue(selectParam, "other")).toContain("configured options");
  });
});

describe("buildParameterFormPayload", () => {
  it("builds and validates parameter values", () => {
    const result = buildParameterFormPayload([{ name: "message", type: "string" }], { message: "hello" });
    expect(result).toEqual({ payload: { message: "hello" } });
  });

  it("returns validation errors", () => {
    const result = buildParameterFormPayload(
      [
        {
          name: "provider",
          type: "select",
          options: [{ label: "OpenAI", value: "openai" }],
        },
      ],
      { provider: "invalid" },
    );
    expect(result).toEqual({ error: expect.stringContaining("configured options") });
  });
});

describe("parseJsonEventPayload", () => {
  it("parses JSON objects", () => {
    expect(parseJsonEventPayload('{"a":1}')).toEqual({ payload: { a: 1 } });
  });

  it("returns errors for invalid JSON", () => {
    expect(parseJsonEventPayload("not json")).toEqual({ error: "Invalid JSON format" });
    expect(parseJsonEventPayload("[]")).toEqual({ error: "Payload must be a JSON object" });
  });
});
