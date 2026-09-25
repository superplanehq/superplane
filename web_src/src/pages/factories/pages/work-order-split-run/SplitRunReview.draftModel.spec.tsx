import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunReview } from "./SplitRunReview";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO, draftStartModelPayload } from "./draftStartModel";
import { DRAFT_START_THINKING_AUTO } from "@/lib/thinkingLevel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: [{ id: "claude-opus-4-6", name: "claude-opus-4-6" }],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => true,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

async function selectFlyoutOption(user: ReturnType<typeof userEvent.setup>, parentTestId: string, optionName: string) {
  await user.hover(screen.getByTestId(parentTestId));
  fireEvent.click(await screen.findByRole("menuitem", { name: optionName }));
}

function renderModelSelect({
  appearance = "labeled",
  model = "claude-opus-4-6",
  thinkingLevel = "medium",
  onChange = vi.fn(),
}: {
  appearance?: "icon" | "labeled" | "ghost";
  model?: string;
  thinkingLevel?: string;
  onChange?: ReturnType<typeof vi.fn>;
} = {}) {
  return render(
    <DraftStartModelSelect
      organizationId="org-1"
      factoryId="factory-1"
      lineName="ship"
      model={model}
      thinkingLevel={thinkingLevel}
      onChange={onChange as (next: { model: string; thinkingLevel: string }) => void}
      appearance={appearance}
    />,
  );
}

function renderDraftFooter(
  onStart: () => void,
  selectedModel = DRAFT_START_MODEL_AUTO,
  onChange = vi.fn(),
  startDisabled = false,
) {
  const footer = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer;
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TooltipProvider>
        <SplitRunReview
          footer={footer}
          onStart={onStart}
          startDisabled={startDisabled}
          modelSelect={
            <DraftStartModelSelect
              organizationId="org-1"
              factoryId="factory-1"
              lineName="ship"
              model={selectedModel}
              thinkingLevel={DRAFT_START_THINKING_AUTO}
              onChange={onChange}
            />
          }
        />
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

describe("SplitRunReview draft model select", () => {
  it("shows a Start plus model chevron on the draft footer", () => {
    renderDraftFooter(vi.fn());

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("button", { name: "Model: Auto" })).toBeInTheDocument();
    expect(within(note).getByTestId("split-run-draft-model")).not.toHaveTextContent("Auto");
    expect(within(note).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Archive" })).not.toBeInTheDocument();
  });

  it("keeps the model select off a waiting footer", () => {
    render(<SplitRunReview footer={splitRunFixtureForWorkOrder(OPEN_WORK_ORDER).footer} onStart={vi.fn()} />);

    expect(screen.queryByTestId("split-run-draft-model")).not.toBeInTheDocument();
  });

  it("starts with Auto without a model id", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderDraftFooter(onStart);

    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(draftStartModelPayload(DRAFT_START_MODEL_AUTO)).toBeUndefined();
  });

  it("lists a runner model the user can pick", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, onChange);

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    expect(screen.getByTestId("split-run-draft-model-list")).toHaveTextContent("Auto");
    expect(screen.getByTestId("split-run-draft-thinking")).toHaveTextContent("Auto");
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-opus-4-6");
    expect(onChange).toHaveBeenCalledWith({ model: "claude-opus-4-6", thinkingLevel: DRAFT_START_THINKING_AUTO });
  });

  it("lists thinking levels the user can pick", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, onChange);

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    await selectFlyoutOption(user, "split-run-draft-thinking", "High");
    expect(onChange).toHaveBeenCalledWith({ model: DRAFT_START_MODEL_AUTO, thinkingLevel: "high" });
    expect(screen.getByTestId("split-run-draft-model-list")).toBeInTheDocument();
  });

  it("keeps the menu open so the user can pick model and thinking", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, onChange);

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    await selectFlyoutOption(user, "split-run-draft-thinking", "High");
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-opus-4-6");
    expect(onChange).toHaveBeenNthCalledWith(1, { model: DRAFT_START_MODEL_AUTO, thinkingLevel: "high" });
    expect(onChange).toHaveBeenNthCalledWith(2, { model: "claude-opus-4-6", thinkingLevel: DRAFT_START_THINKING_AUTO });
  });

  it("disables the model chevron when Start is disabled", () => {
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, vi.fn(), true);

    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Model: Auto" })).toBeDisabled();
  });

  it("shows Medium after the model name on a closed labeled trigger", () => {
    renderModelSelect({ appearance: "labeled", thinkingLevel: "medium" });

    const trigger = screen.getByRole("button", { name: "Model: claude-opus-4-6, Medium" });
    const name = within(trigger).getByText("claude-opus-4-6");
    const level = within(trigger).getByText("Medium");
    expect(name.compareDocumentPosition(level) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(level).toHaveClass("shrink-0", "text-muted-foreground");
    expect(name).toHaveClass("truncate");
  });

  it("shows Medium after the model name on a closed ghost trigger", () => {
    renderModelSelect({ appearance: "ghost", thinkingLevel: "medium" });

    const trigger = screen.getByRole("button", { name: "Model: claude-opus-4-6, Medium" });
    const name = within(trigger).getByText("claude-opus-4-6");
    const level = within(trigger).getByText("Medium");
    expect(name.compareDocumentPosition(level) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(level).toHaveClass("shrink-0", "text-muted-foreground");
  });

  it("hides Auto and Default on a closed trigger", () => {
    const { rerender } = renderModelSelect({
      appearance: "labeled",
      thinkingLevel: "auto",
      model: DRAFT_START_MODEL_AUTO,
    });

    expect(screen.getByRole("button", { name: "Model: Auto" }).textContent?.replace(/\s+/g, " ").trim()).toBe("Auto");

    rerender(
      <DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="default" onChange={vi.fn()} appearance="ghost" />,
    );
    const defaultTrigger = screen.getByRole("button", { name: "Model: claude-opus-4-6" });
    expect(defaultTrigger).not.toHaveTextContent("Default");
    expect(defaultTrigger).not.toHaveTextContent("Medium");

    rerender(
      <DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="" onChange={vi.fn()} appearance="labeled" />,
    );
    expect(screen.getByRole("button", { name: "Model: claude-opus-4-6" })).not.toHaveTextContent("Default");
  });

  it("keeps the icon trigger free of the model name and thinking level", () => {
    renderModelSelect({ appearance: "icon", thinkingLevel: "medium" });

    const trigger = screen.getByRole("button", { name: "Model: claude-opus-4-6" });
    expect(trigger).not.toHaveTextContent("claude-opus-4-6");
    expect(trigger).not.toHaveTextContent("Medium");
  });

  it("still reports the current thinking level on the open Thinking row", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderModelSelect({ appearance: "labeled", thinkingLevel: "medium", onChange });

    await user.click(screen.getByRole("button", { name: "Model: claude-opus-4-6, Medium" }));
    expect(screen.getByTestId("split-run-draft-thinking")).toHaveTextContent("Thinking");
    expect(screen.getByTestId("split-run-draft-thinking")).toHaveTextContent("Medium");
    await selectFlyoutOption(user, "split-run-draft-thinking", "Low");
    expect(onChange).toHaveBeenCalledWith({ model: "claude-opus-4-6", thinkingLevel: "low" });
  });
});
