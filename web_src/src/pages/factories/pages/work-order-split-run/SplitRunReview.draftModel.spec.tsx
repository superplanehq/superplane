import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunReview } from "./SplitRunReview";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO, draftStartModelPayload } from "./draftStartModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: {
      models: [
        { id: "claude-haiku-4-5", name: "claude-haiku-4-5", thinkingLevels: [] },
        { id: "claude-opus-4-6", name: "claude-opus-4-6", thinkingLevels: ["low", "medium", "high", "max"] },
        { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6", thinkingLevels: ["low", "medium", "high"] },
      ],
    },
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

function StatefulDraftModelSelect({
  onChange,
}: {
  onChange: (next: { model: string; thinkingLevel: string }) => void;
}) {
  const [model, setModel] = useState(DRAFT_START_MODEL_AUTO);
  const [thinkingLevel, setThinkingLevel] = useState("");
  return (
    <DraftStartModelSelect
      organizationId="org-1"
      factoryId="factory-1"
      lineName="ship"
      model={model}
      thinkingLevel={thinkingLevel}
      onChange={(next) => {
        setModel(next.model);
        setThinkingLevel(next.thinkingLevel);
        onChange(next);
      }}
    />
  );
}

function renderDraftFooter(
  onStart: () => void,
  selectedModel = DRAFT_START_MODEL_AUTO,
  onChange = vi.fn(),
  startDisabled = false,
  thinkingLevel = "",
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
              thinkingLevel={thinkingLevel}
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
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <SplitRunReview footer={splitRunFixtureForWorkOrder(OPEN_WORK_ORDER).footer} onStart={vi.fn()} />
      </QueryClientProvider>,
    );

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

  it("lists a runner model the user can pick and hides Thinking for Auto", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, onChange);

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    expect(screen.getByTestId("split-run-draft-model-list")).toHaveTextContent("Auto");
    expect(screen.queryByTestId("split-run-draft-thinking")).not.toBeInTheDocument();
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-opus-4-6");
    expect(onChange).toHaveBeenCalledWith({ model: "claude-opus-4-6", thinkingLevel: "" });
  });

  it("lists the thinking levels of the selected model", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), "claude-opus-4-6", onChange);

    await user.click(screen.getByRole("button", { name: "Model: claude-opus-4-6" }));
    await user.hover(screen.getByTestId("split-run-draft-thinking"));
    expect(await screen.findByRole("menuitem", { name: "Low" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Medium" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "High" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Max" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Minimal" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Default" })).not.toBeInTheDocument();
    await selectFlyoutOption(user, "split-run-draft-thinking", "Max");
    expect(onChange).toHaveBeenCalledWith({ model: "claude-opus-4-6", thinkingLevel: "max" });
  });

  it("hides Thinking for a model without thinking levels", async () => {
    const user = userEvent.setup();
    renderDraftFooter(vi.fn(), "claude-haiku-4-5", vi.fn(), false, "high");

    await user.click(screen.getByRole("button", { name: "Model: claude-haiku-4-5" }));
    expect(screen.queryByTestId("split-run-draft-thinking")).not.toBeInTheDocument();
  });

  it("keeps the picked thinking level only while the model supports it", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <TooltipProvider>
          <StatefulDraftModelSelect onChange={onChange} />
        </TooltipProvider>
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-opus-4-6");
    await selectFlyoutOption(user, "split-run-draft-thinking", "High");
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-sonnet-4-6");
    await selectFlyoutOption(user, "split-run-draft-model-list", "claude-haiku-4-5");
    expect(onChange).toHaveBeenNthCalledWith(1, { model: "claude-opus-4-6", thinkingLevel: "" });
    expect(onChange).toHaveBeenNthCalledWith(2, { model: "claude-opus-4-6", thinkingLevel: "high" });
    expect(onChange).toHaveBeenNthCalledWith(3, { model: "claude-sonnet-4-6", thinkingLevel: "high" });
    expect(onChange).toHaveBeenNthCalledWith(4, { model: "claude-haiku-4-5", thinkingLevel: "" });
  });

  it("clears a stored thinking level that the model does not support", () => {
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), "claude-sonnet-4-6", onChange, false, "max");

    expect(onChange).toHaveBeenCalledWith({ model: "claude-sonnet-4-6", thinkingLevel: "" });
  });

  it("keeps a stored thinking level that the model supports", () => {
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), "claude-sonnet-4-6", onChange, false, "high");

    expect(onChange).not.toHaveBeenCalled();
  });

  it("disables the model chevron when Start is disabled", () => {
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, vi.fn(), true);

    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Model: Auto" })).toBeDisabled();
  });

  it("shows Medium after the model name on the closed ghost box", () => {
    render(
      <DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="medium" onChange={vi.fn()} appearance="ghost" />,
    );

    expect(screen.getByRole("button", { name: "Model: claude-opus-4-6 Medium" })).toHaveTextContent(
      "claude-opus-4-6 Medium",
    );
  });

  it("shows Medium after the model name on the closed labeled box", () => {
    render(
      <DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="medium" onChange={vi.fn()} appearance="labeled" />,
    );

    expect(screen.getByRole("button", { name: "Model: claude-opus-4-6 Medium" })).toHaveTextContent(
      "claude-opus-4-6 Medium",
    );
  });

  it("leaves Auto as the model name without a thinking word", () => {
    render(
      <DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="auto" onChange={vi.fn()} appearance="ghost" />,
    );

    const button = screen.getByRole("button", { name: "Model: claude-opus-4-6" });
    expect(button).toHaveTextContent("claude-opus-4-6");
    expect(button).not.toHaveTextContent("Medium");
    expect(button).not.toHaveTextContent("Default");
  });

  it("keeps the icon control free of visible thinking text", () => {
    render(<DraftStartModelSelect model="claude-opus-4-6" thinkingLevel="medium" onChange={vi.fn()} />);

    const button = screen.getByRole("button", { name: "Model: claude-opus-4-6 Medium" });
    expect(button).not.toHaveTextContent("Medium");
    expect(button).not.toHaveTextContent("claude-opus-4-6");
  });
});
