import { describe, expect, it } from "vitest";
import type { ConfigurationField } from "@/api-client";
import {
  areRequiredConfigurationFieldsFilled,
  filterVisibleConfiguration,
  isFieldRequired,
  isFieldVisible,
  parseDefaultValues,
  validateFieldForSubmission,
} from "@/lib/components";

function buildField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "field",
    type: "string",
    ...overrides,
  };
}

describe("components visibility helpers", () => {
  it("evaluates field visibility with exact and wildcard matches", () => {
    const field = buildField({
      visibilityConditions: [
        { field: "provider", values: ["github"] },
        { field: "token", values: ["*"] },
      ],
    });

    expect(isFieldVisible(field, { provider: "github", token: "secret" })).toBe(true);
    expect(isFieldVisible(field, { provider: "github", token: "" })).toBe(false);
  });

  it("treats boolean true/false as the string values 'true'/'false'", () => {
    // Boolean config fields gate dependents via values: ["true"] (e.g. memory
    // components list mode). The helper stringifies the actual value before
    // comparing, so JSON booleans satisfy these conditions.
    const field = buildField({
      visibilityConditions: [{ field: "iterateList", values: ["true"] }],
    });

    expect(isFieldVisible(field, { iterateList: true })).toBe(true);
    expect(isFieldVisible(field, { iterateList: false })).toBe(false);
    expect(isFieldVisible(field, { iterateList: "true" })).toBe(true);
    expect(isFieldVisible(field, {})).toBe(false);
  });

  it("filters hidden nested fields from objects and lists", () => {
    const fields: ConfigurationField[] = [
      buildField({ name: "provider" }),
      buildField({
        name: "config",
        type: "object",
        typeOptions: {
          object: {
            schema: [
              buildField({ name: "visibleChild" }),
              buildField({
                name: "hiddenChild",
                visibilityConditions: [{ field: "provider", values: ["github"] }],
              }),
            ],
          },
        },
      }),
      buildField({
        name: "items",
        type: "list",
        typeOptions: {
          list: {
            itemDefinition: {
              schema: [
                buildField({ name: "always" }),
                buildField({
                  name: "gated",
                  visibilityConditions: [{ field: "kind", values: ["enabled"] }],
                }),
              ],
            },
          },
        },
      }),
    ];

    expect(
      filterVisibleConfiguration(
        {
          provider: "gitlab",
          config: {
            visibleChild: "yes",
            hiddenChild: "no",
          },
          items: [
            { always: "a", kind: "enabled", gated: "keep" },
            { always: "b", kind: "disabled", gated: "drop" },
          ],
        },
        fields,
      ),
    ).toEqual({
      provider: "gitlab",
      config: {
        visibleChild: "yes",
      },
      items: [{ always: "a", gated: "keep" }, { always: "b" }],
    });
  });

  it("evaluates required conditions", () => {
    const alwaysRequired = buildField({ required: true });
    const conditionallyRequired = buildField({
      requiredConditions: [{ field: "provider", values: ["github", "gitlab"] }],
    });

    expect(isFieldRequired(alwaysRequired, {})).toBe(true);
    expect(isFieldRequired(conditionallyRequired, { provider: "github" })).toBe(true);
    expect(isFieldRequired(conditionallyRequired, { provider: "slack" })).toBe(false);
  });
});

describe("areRequiredConfigurationFieldsFilled", () => {
  it("is false when a required visible field is empty", () => {
    const fields = [buildField({ name: "apiKey", required: true })];

    expect(areRequiredConfigurationFieldsFilled(fields, { apiKey: "" })).toBe(false);
    expect(areRequiredConfigurationFieldsFilled(fields, {})).toBe(false);
  });

  it("is true once the required field has a value", () => {
    const fields = [buildField({ name: "apiKey", required: true })];

    expect(areRequiredConfigurationFieldsFilled(fields, { apiKey: "sk-123" })).toBe(true);
  });

  it("only requires conditionally-required fields once their condition matches", () => {
    const fields = [
      buildField({ name: "provider" }),
      buildField({
        name: "token",
        requiredConditions: [{ field: "provider", values: ["github"] }],
      }),
    ];

    expect(areRequiredConfigurationFieldsFilled(fields, { provider: "slack" })).toBe(true);
    expect(areRequiredConfigurationFieldsFilled(fields, { provider: "github" })).toBe(false);
    expect(areRequiredConfigurationFieldsFilled(fields, { provider: "github", token: "abc" })).toBe(true);
  });

  it("ignores a required field that is hidden", () => {
    const fields = [
      buildField({ name: "adminKey", required: true, visibilityConditions: [{ field: "advanced", values: ["true"] }] }),
    ];

    expect(areRequiredConfigurationFieldsFilled(fields, { advanced: "false" })).toBe(true);
    expect(areRequiredConfigurationFieldsFilled(fields, { advanced: "true" })).toBe(false);
  });

  it("treats an empty array as missing for a required multi-select field", () => {
    const fields = [buildField({ name: "scopes", type: "multi-select", required: true })];

    expect(areRequiredConfigurationFieldsFilled(fields, { scopes: [] })).toBe(false);
    expect(areRequiredConfigurationFieldsFilled(fields, { scopes: ["repo"] })).toBe(true);
  });
});

describe("components value parsing and validation", () => {
  it("validates cron and number submission values", () => {
    expect(validateFieldForSubmission(buildField({ type: "cron" }), "bad")).toEqual(["Cron expression too short"]);
    expect(validateFieldForSubmission(buildField({ type: "cron" }), "0 9 31 * *")).toEqual([]);
    expect(validateFieldForSubmission(buildField({ type: "cron" }), "0 24 * * *")).toEqual(["Invalid hour value"]);
    expect(
      validateFieldForSubmission(
        buildField({
          type: "number",
          typeOptions: { number: { min: 2, max: 4 } },
        }),
        1,
      ),
    ).toEqual(["Value must be at least 2"]);
    expect(
      validateFieldForSubmission(
        buildField({
          type: "number",
          typeOptions: { number: { min: 2, max: 4 } },
        }),
        5,
      ),
    ).toEqual(["Value must not exceed 4"]);
  });

  it("parses default values according to field type", () => {
    expect(
      parseDefaultValues([
        buildField({ name: "count", type: "number", defaultValue: "3" }),
        buildField({ name: "enabled", type: "boolean", defaultValue: "true" }),
        buildField({ name: "items", type: "multi-select", defaultValue: '["a","b"]' }),
        buildField({ name: "single", type: "multi-select", defaultValue: "a" }),
        buildField({ name: "config", type: "object", defaultValue: '{"ok":true}' }),
        buildField({ name: "timezone", type: "timezone", defaultValue: "current" }),
        buildField({ name: "raw", type: "string", defaultValue: "value" }),
      ]),
    ).toEqual({
      count: 3,
      enabled: true,
      items: ["a", "b"],
      single: ["a"],
      config: { ok: true },
      timezone: (-new Date().getTimezoneOffset() / 60).toString(),
      raw: "value",
    });
  });
});
