import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

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

function renderDraftFooter(onStart: () => void, selectedModel = DRAFT_START_MODEL_AUTO, onChange = vi.fn()) {
  const footer = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer;
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <SplitRunReview
        footer={footer}
        onStart={onStart}
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
    </QueryClientProvider>,
  );
}

describe("SplitRunReview draft model select", () => {
  it("shows Auto on the draft footer", () => {
    renderDraftFooter(vi.fn());

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("split-run-draft-model")).toHaveTextContent("Auto");
    expect(within(note).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
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

    await user.click(screen.getByRole("combobox", { name: "Model" }));
    await user.click(await screen.findByRole("option", { name: "claude-opus-4-6" }));
    expect(onChange).toHaveBeenCalledWith("claude-opus-4-6");
  });
});
