import { describe, expect, it } from "vitest";
import type { ConfigurationField } from "@/api-client";
import {
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

  it("accepts day and month names in either case", () => {
    // Cron names are case-insensitive for the backend validator
    // (pkg/configuration/validation.go allows a-z as well as A-Z) and for the
    // next-run preview in lib/cron.ts, which upper-cases before lookup. The
    // submission validator has to agree, or a lowercase schedule the server
    // would happily accept is blocked in the form.
    const cronField = buildField({ type: "cron" });

    for (const expression of [
      "0 0 * * mon",
      "0 0 * * MON",
      "0 0 * * mon-fri",
      "0 0 * * sun,sat",
      "0 0 * jan *",
      "0 0 * Feb *",
    ]) {
      expect(validateFieldForSubmission(cronField, expression)).toEqual([]);
    }
  });

  it("still rejects characters that are not valid in a cron expression", () => {
    const cronField = buildField({ type: "cron" });

    expect(validateFieldForSubmission(cronField, "0 0 * * mon;fri")).toEqual([
      "Invalid characters. Use only: numbers, *, ,, -, / and day names",
    ]);
    expect(validateFieldForSubmission(cronField, "0 0 * * @mon")).toEqual([
      "Invalid characters. Use only: numbers, *, ,, -, / and day names",
    ]);
    expect(validateFieldForSubmission(cronField, "0 0 * * mon_fri")).toEqual([
      "Invalid characters. Use only: numbers, *, ,, -, / and day names",
    ]);
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
