import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { SplitRunReview } from "./SplitRunReview";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO, draftStartModelPayload } from "./draftStartModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: [{ id: "claude-opus-4-6", name: "claude-opus-4-6" }],
    isLoading: false,
  }),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

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
              value={selectedModel}
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
    expect(within(note).getByRole("button", { name: "Build" })).toBeInTheDocument();
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

    await user.click(screen.getByRole("button", { name: "Build" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(draftStartModelPayload(DRAFT_START_MODEL_AUTO)).toBeUndefined();
  });

  it("lists a runner model the user can pick", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, onChange);

    await user.click(screen.getByRole("button", { name: "Model: Auto" }));
    expect(await screen.findByRole("menuitemradio", { name: "Auto" })).toHaveAttribute("aria-checked", "true");
    await user.click(screen.getByRole("menuitemradio", { name: "claude-opus-4-6" }));
    expect(onChange).toHaveBeenCalledWith("claude-opus-4-6");
  });

  it("disables the model chevron when Start is disabled", () => {
    renderDraftFooter(vi.fn(), DRAFT_START_MODEL_AUTO, vi.fn(), true);

    expect(screen.getByRole("button", { name: "Build" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Model: Auto" })).toBeDisabled();
  });
});
