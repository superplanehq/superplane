import { describe, expect, it } from "vitest";
import { buildTemplateParametersAutocompleteObject } from "./templateParametersAutocomplete";

describe("buildTemplateParametersAutocompleteObject", () => {
  it("returns null when parameters are missing", () => {
    expect(buildTemplateParametersAutocompleteObject({})).toBeNull();
  });

  it("builds defaults and typed fallbacks for template parameters", () => {
    const out = buildTemplateParametersAutocompleteObject({
      parameters: [
        { name: "message", type: "string", defaultString: "Hello" },
        { name: "count", type: "number" },
        { name: "enabled", type: "boolean" },
      ],
    });

    expect(out).toEqual({
      message: "Hello",
      count: 0,
      enabled: false,
    });
  });

  it("builds select defaults from defaultString or first option", () => {
    const out = buildTemplateParametersAutocompleteObject({
      parameters: [
        {
          name: "provider",
          type: "select",
          defaultString: "anthropic",
          options: [
            { label: "OpenAI", value: "openai" },
            { label: "Anthropic", value: "anthropic" },
          ],
        },
        {
          name: "region",
          type: "select",
          options: [
            { label: "US", value: "us" },
            { label: "EU", value: "eu" },
          ],
        },
      ],
    });

    expect(out).toEqual({
      provider: "anthropic",
      region: "us",
    });
  });

  it("ignores invalid items and keeps empty string defaults", () => {
    const out = buildTemplateParametersAutocompleteObject({
      parameters: [
        { name: "message", type: "string", defaultString: "" },
        { name: "", type: "number", defaultNumber: 1 },
        { bad: "item" },
      ],
    });

    expect(out).toEqual({
      message: "",
    });
  });
});
