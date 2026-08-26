import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ConfigurationField } from "@/api-client";
import { describe, expect, it, vi } from "vitest";
import { TooltipProvider } from "@/ui/tooltip";
import { ConfigurationFieldRenderer } from "./index";
import { buildTemplateParametersAutocompleteObject } from "./templateParametersAutocomplete";
import { EXPRESSION_DOUBLE_BRACE_TIP } from "./expressionQuickTip";

const runTitleField: ConfigurationField = {
  name: "customName",
  type: "string",
  label: "Run title",
  description: "Give each run a dynamic title using expressions.",
  togglable: true,
  placeholder: "{{ root().data.context }}",
};

function renderConfigurationField(props: React.ComponentProps<typeof ConfigurationFieldRenderer>) {
  return render(
    React.createElement(TooltipProvider, {
      delayDuration: 0,
      children: React.createElement(ConfigurationFieldRenderer, props),
    }),
  );
}

describe("ConfigurationFieldRenderer run title copy", () => {
  it("explains disabled trigger run title customization in a label tooltip", async () => {
    const user = userEvent.setup();
    renderConfigurationField({
      allowExpressions: true,
      field: runTitleField,
      value: null,
      onChange: vi.fn(),
      autocompleteExampleObj: { __root: { data: { context: "ci/build" } } },
    });

    expect(screen.getByText("Customize run title")).toBeInTheDocument();
    expect(
      screen.queryByText(
        "This trigger starts a run when an event arrives. By default, SuperPlane names the run from the event payload.",
      ),
    ).not.toBeInTheDocument();

    await user.hover(screen.getByRole("button", { name: "About Customize run title" }));
    expect(
      await screen.findByText(
        "This trigger starts a run when an event arrives. By default, SuperPlane names the run from the event payload.",
      ),
    ).toBeInTheDocument();
  });

  it("explains enabled trigger run title customization in a label tooltip", async () => {
    const user = userEvent.setup();
    renderConfigurationField({
      allowExpressions: true,
      field: runTitleField,
      value: "{{ root().data.context }}",
      onChange: vi.fn(),
      autocompleteExampleObj: { __root: { data: { context: "ci/build" } } },
    });

    expect(screen.getByRole("button", { name: "Preview title" })).toBeInTheDocument();
    expect(
      screen.queryByText(
        "Set the title for runs started by this trigger. Use root().data to reference fields from the trigger event.",
      ),
    ).not.toBeInTheDocument();

    await user.hover(screen.getByRole("button", { name: "About Customize run title" }));
    expect(
      await screen.findByText(
        "Set the title for runs started by this trigger. Use root().data to reference fields from the trigger event.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(EXPRESSION_DOUBLE_BRACE_TIP)).toBeInTheDocument();
  });
});

describe("ConfigurationFieldRenderer tip and description layout", () => {
  it("keeps the description inline when expressions are not allowed", () => {
    const field: ConfigurationField = {
      name: "repo",
      type: "string",
      label: "Repository",
      description: "GitHub repository to watch.",
    };

    renderConfigurationField({
      allowExpressions: false,
      field,
      value: "acme/app",
      onChange: vi.fn(),
    });

    expect(screen.getByText("GitHub repository to watch.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "About Repository" })).not.toBeInTheDocument();
  });

  it("moves description into a tooltip for expression fields that also show a tip", async () => {
    const user = userEvent.setup();
    const field: ConfigurationField = {
      name: "branch",
      type: "string",
      label: "Branch",
      description: "Branch name or expression.",
    };

    renderConfigurationField({
      allowExpressions: true,
      field,
      value: "main",
      onChange: vi.fn(),
      autocompleteExampleObj: { __root: { data: {} } },
    });

    expect(screen.queryByText("Branch name or expression.")).not.toBeInTheDocument();
    expect(screen.queryByText(EXPRESSION_DOUBLE_BRACE_TIP)).not.toBeInTheDocument();

    await user.hover(screen.getByRole("button", { name: "About Branch" }));
    expect(await screen.findByText("Branch name or expression.")).toBeInTheDocument();
    expect(screen.getByText(EXPRESSION_DOUBLE_BRACE_TIP)).toBeInTheDocument();
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
