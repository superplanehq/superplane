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
});
