import React from "react";
import { render, screen } from "@testing-library/react";
import type { ConfigurationField } from "@/api-client";
import { describe, expect, it, vi } from "vitest";
import { ConfigurationFieldRenderer } from "./index";
import { buildTemplateParametersAutocompleteObject } from "./templateParametersAutocomplete";

const runTitleField: ConfigurationField = {
  name: "customName",
  type: "string",
  label: "Run title",
  description: "Give each run a dynamic title using expressions.",
  togglable: true,
  placeholder: "{{ root().data.context }}",
};

describe("ConfigurationFieldRenderer run title copy", () => {
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
