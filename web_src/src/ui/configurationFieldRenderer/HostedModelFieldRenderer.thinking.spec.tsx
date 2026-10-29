import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";
import { HostedModelFieldRenderer } from "./HostedModelFieldRenderer";
import {
  createField,
  expectClosedThinkingWord,
  installPointerCapturePolyfills,
  mockSelectableModels,
  renderField,
  selectableModel,
  selectFlyoutOption,
} from "./HostedModelFieldRenderer.testHelpers";

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isPending: false }),
}));

describe("HostedModelFieldRenderer thinking levels", () => {
  beforeAll(installPointerCapturePolyfills);

  beforeEach(() => {
    mockSelectableModels([selectableModel("byok", "anthropic", "claude-sonnet-4-6")]);
  });

  it("shows the thinking word beside the closed model name", () => {
    const cases = [
      [{}, "Default"],
      [{ thinkingLevel: "low" }, "Low"],
      [{ thinkingLevel: "medium" }, "Medium"],
      [{ thinkingLevel: "high" }, "High"],
    ] as const;

    for (const [extra, word] of cases) {
      const view = renderField(
        <HostedModelFieldRenderer
          field={createField()}
          value="claude-sonnet-4-6"
          onChange={vi.fn()}
          allValues={{ model: "claude-sonnet-4-6", ...extra }}
          organizationId="org-1"
        />,
      );

      expectClosedThinkingWord("sonnet 4-6", word);
      view.unmount();
    }
  });

  it("writes thinkingLevel next to the model", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "claude-sonnet-4-6" }}
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("sonnet 4-6");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "High");
    expect(onValuesChange).toHaveBeenCalledWith({ model: "claude-sonnet-4-6", thinkingLevel: "high" });
  });

  it("offers Codex effort levels for OpenAI models", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([selectableModel("byok", "openai", "gpt-5.4", ["none", "low", "medium", "high", "xhigh"])]);

    renderField(
      <HostedModelFieldRenderer
        field={createField("openai")}
        value="gpt-5.4"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "gpt-5.4" }}
        organizationId="org-1"
      />,
    );

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "Extra high");
    expect(onValuesChange).toHaveBeenCalledWith({ model: "gpt-5.4", thinkingLevel: "xhigh" });
  });

  it("uses a flat model-only menu and preserves Codex effort when the model changes", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([
      selectableModel("byok", "openai", "gpt-5.4", ["low", "medium", "high", "xhigh"]),
      selectableModel("byok", "openai", "gpt-5.3-codex", ["low", "medium", "high", "xhigh"]),
    ]);
    renderField(
      <HostedModelFieldRenderer
        field={createField("openai")}
        value="gpt-5.4"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "gpt-5.4", thinkingLevel: "xhigh" }}
        organizationId="org-1"
        modelOnly
      />,
    );
    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.queryByTestId("field-model-hosted-thinking")).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "openai/gpt-5.3-codex" }));
    expect(onValuesChange).toHaveBeenCalledWith({ model: "gpt-5.3-codex", thinkingLevel: "xhigh" });
  });

  it("offers only the thinking levels of the selected model", async () => {
    const user = userEvent.setup();
    mockSelectableModels([
      selectableModel("byok", "anthropic", "claude-opus-5-5", ["low", "medium", "high", "xhigh", "max"]),
    ]);

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-opus-5-5"
        onChange={vi.fn()}
        onValuesChange={vi.fn()}
        allValues={{ model: "claude-opus-5-5" }}
        organizationId="org-1"
      />,
    );

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-thinking"));
    expect(await screen.findByRole("menuitem", { name: "Max" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Extra high" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Minimal" })).not.toBeInTheDocument();
  });

  it("hides Thinking for a model without thinking levels", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([
      selectableModel("byok", "anthropic", "claude-haiku-4-5", []),
      selectableModel("byok", "anthropic", "claude-sonnet-4-6"),
    ]);

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-haiku-4-5"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "claude-haiku-4-5", thinkingLevel: "high" }}
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("field-model-hosted-model")).not.toHaveTextContent("High");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    expect(screen.queryByTestId("field-model-hosted-thinking")).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "sonnet 4-6" }));
    expect(onValuesChange).toHaveBeenCalledWith({ model: "claude-sonnet-4-6", thinkingLevel: "high" });
  });

  it("clears a thinking level that the new model does not support", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();
    mockSelectableModels([
      selectableModel("byok", "anthropic", "claude-opus-5-5", ["low", "medium", "high", "xhigh", "max"]),
      selectableModel("byok", "anthropic", "claude-haiku-4-5", []),
    ]);

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-opus-5-5"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "claude-opus-5-5", thinkingLevel: "max" }}
        organizationId="org-1"
      />,
    );

    expectClosedThinkingWord("opus 5-5", "Max");
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-model-list", "haiku 4-5");
    expect(onValuesChange).toHaveBeenCalledWith({ model: "claude-haiku-4-5", thinkingLevel: undefined });
  });

  it("clears thinkingLevel when Default is selected", async () => {
    const user = userEvent.setup();
    const onValuesChange = vi.fn();

    renderField(
      <HostedModelFieldRenderer
        field={createField()}
        value="claude-sonnet-4-6"
        onChange={vi.fn()}
        onValuesChange={onValuesChange}
        allValues={{ model: "claude-sonnet-4-6", thinkingLevel: "high" }}
        organizationId="org-1"
      />,
    );

    await user.click(screen.getByTestId("field-model-hosted-model"));
    await selectFlyoutOption(user, "field-model-hosted-thinking", "Default");
    expect(onValuesChange).toHaveBeenCalledWith({ model: "claude-sonnet-4-6", thinkingLevel: undefined });
  });
});
