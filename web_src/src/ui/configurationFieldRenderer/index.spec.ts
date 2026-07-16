import React from "react";
import { render, screen } from "@testing-library/react";
import type { ConfigurationField } from "@/api-client";
import { describe, expect, it, vi } from "vitest";
import { ConfigurationFieldRenderer } from "./index";
import { buildTemplateParametersAutocompleteObject } from "./templateParametersAutocomplete";

describe("ConfigurationFieldRenderer run title copy", () => {
  const runTitleField: ConfigurationField = {
    name: "customName",
    type: "string",
    label: "Run title",
    description: "Give each run a dynamic title using expressions.",
    togglable: true,
    placeholder: "{{ root().data.context }}",
  };

  it("explains disabled trigger run title customization", () => {
    render(
      React.createElement(ConfigurationFieldRenderer, {
        allowExpressions: true,
        field: runTitleField,
        value: null,
        onChange: vi.fn(),
        autocompleteExampleObj: { __root: { data: { context: "ci/build" } } },
      }),
    );

    expect(screen.getByText("Customize run title")).toBeInTheDocument();
    expect(
      screen.getByText(
        "This trigger starts a run when an event arrives. By default, SuperPlane names the run from the event payload.",
      ),
    ).toBeInTheDocument();
  });

  it("explains enabled trigger run title customization", () => {
    render(
      React.createElement(ConfigurationFieldRenderer, {
        allowExpressions: true,
        field: runTitleField,
        value: "{{ root().data.context }}",
        onChange: vi.fn(),
        autocompleteExampleObj: { __root: { data: { context: "ci/build" } } },
      }),
    );

    expect(
      screen.getByText(
        "Set the title for runs started by this trigger. Use root().data to reference fields from the trigger event.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Preview title" })).toBeInTheDocument();
  });
});

describe("ConfigurationFieldRenderer preserveEditLayout", () => {
  it("renders nested list fields with standard disabled inputs", () => {
    render(
      React.createElement(ConfigurationFieldRenderer, {
        field: {
          name: "matchList",
          type: "list",
          typeOptions: {
            list: {
              itemDefinition: {
                type: "object",
                schema: [
                  { name: "left", type: "string", label: "Left" },
                  { name: "right", type: "expression", label: "Right" },
                ],
              },
            },
          },
        },
        value: [{ left: "updated_at", right: "int(now().Unix())" }],
        onChange: vi.fn(),
        readOnly: true,
        preserveEditLayout: true,
        allowExpressions: false,
      }),
    );

    expect(screen.getByDisplayValue("updated_at")).toBeDisabled();
    expect(screen.getByDisplayValue("int(now().Unix())")).toBeDisabled();
  });

  it("does not mutate values when readOnly", () => {
    const onChange = vi.fn();
    render(
      React.createElement(ConfigurationFieldRenderer, {
        field: { name: "timezone", type: "timezone" },
        value: undefined,
        onChange,
        readOnly: true,
        preserveEditLayout: true,
      }),
    );

    expect(onChange).not.toHaveBeenCalled();
  });
});

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
