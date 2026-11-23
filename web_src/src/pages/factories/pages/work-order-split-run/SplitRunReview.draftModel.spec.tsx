import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { MobileDraftStartModelSelect } from "../../mobile/MobileDraftStartModelSelect";
import type { DraftStartSelection } from "./draftStartModel";
import { SplitRunReview } from "./SplitRunReview";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

let catalog = [{ id: "openai/gpt-6.1-sol", name: "GPT 6.1 Sol" }];
vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: { models: catalog },
    isLoading: false,
  }),
}));

function Selector({ disabled = false }: { disabled?: boolean }) {
  const [selection, onChange] = useState<DraftStartSelection>({});
  return (
    <DraftStartModelSelect
      {...selection}
      defaults={{
        values: { model: "openai/gpt-6.1-sol", thinkingLevel: "high" },
        isLoading: false,
        isError: false,
        retry: vi.fn(),
      }}
      onChange={onChange}
      disabled={disabled}
      appearance="labeled"
    />
  );
}

async function choose(user: ReturnType<typeof userEvent.setup>, field: string, name: string) {
  await user.hover(screen.getByTestId(field));
  fireEvent.click(await screen.findByRole("menuitem", { name }));
}

describe("SplitRunReview draft model select", () => {
  beforeEach(() => {
    catalog = [{ id: "openai/gpt-6.1-sol", name: "GPT 6.1 Sol" }];
  });
  it("shows saved defaults next to Start and preserves independent choices", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <TooltipProvider>
          <SplitRunReview
            footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer}
            onStart={vi.fn()}
            modelSelect={<Selector />}
          />
        </TooltipProvider>
      </QueryClientProvider>,
    );
    const trigger = screen.getByTestId("split-run-draft-model");
    expect(trigger).toHaveAccessibleName("Model: GPT 6.1 Sol (Default) · Thinking: High (Default)");
    await user.click(trigger);
    expect(screen.getByTestId("split-run-draft-model-list")).toHaveTextContent("ModelGPT 6.1 Sol");
    expect(screen.getByTestId("split-run-draft-thinking")).toHaveTextContent("ThinkingHigh");
    expect(screen.queryByRole("menuitem", { name: "Use defaults" })).not.toBeInTheDocument();
    await choose(user, "split-run-draft-model-list", "GPT 6.1 Sol");
    expect(trigger).toHaveAccessibleName(/GPT 6.1 Sol.*Thinking: High \(Default\)/);
    expect(screen.getByRole("menuitem", { name: "Use defaults" })).toBeInTheDocument();
    await choose(user, "split-run-draft-thinking", "Low");
    expect(trigger).toHaveAccessibleName(/Thinking: Low$/);
    await choose(user, "split-run-draft-model-list", "Default (GPT 6.1 Sol)");
    expect(trigger).toHaveAccessibleName(/Default.*Thinking: Low$/);
    await user.click(screen.getByRole("menuitem", { name: "Use defaults" }));
    expect(trigger).toHaveAccessibleName(/High \(Default\)/);
    await user.click(trigger);
    expect(screen.queryByRole("menuitem", { name: "Use defaults" })).not.toBeInTheDocument();
  });

  it.each(["desktop", "mobile"])("keeps same-name choices readable and selectable on %s", async (surface) => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const model = "openai/gpt-6.1-sol";
    catalog = [
      { id: `hosted::openrouter::${model}`, name: model },
      { id: `byok::openrouter::${model}`, name: model },
      { id: `hosted::openai::gpt-6.1-sol`, name: model },
      { id: model, name: model },
      { id: "hosted::openrouter::vendor/unique", name: "hosted::openrouter::vendor/unique" },
    ];
    function Choices() {
      const [selection, setSelection] = useState<DraftStartSelection>({});
      const props = {
        ...selection,
        lineName: "ship",
        defaults: {
          values: { model: catalog[0].id, thinkingLevel: "high" },
          isLoading: false,
          isError: false,
          retry: vi.fn(),
        },
        onChange: (next: DraftStartSelection) => {
          setSelection(next);
          onChange(next);
        },
      };
      return surface === "mobile" ? <MobileDraftStartModelSelect {...props} /> : <DraftStartModelSelect {...props} />;
    }
    render(<Choices />);
    const trigger = screen.getByTestId("split-run-draft-model");
    await user.click(trigger);
    if (surface === "mobile") {
      expect(screen.getByText(`Model — ${model} (Hosted · OpenRouter)`)).toBeInTheDocument();
      expect(screen.getByText("Thinking — High")).toBeInTheDocument();
    }
    expect(screen.queryByRole("menuitem", { name: "Use defaults" })).not.toBeInTheDocument();
    for (const [index, qualifier] of ["Hosted · OpenRouter", "API key", "Hosted · OpenAI", "Line runner"].entries()) {
      const label = `${model} (${qualifier})`;
      if (surface === "desktop") await choose(user, "split-run-draft-model-list", label);
      else await user.click(screen.getByRole("menuitem", { name: label }));
      expect(onChange).toHaveBeenLastCalledWith({ model: catalog[index].id, thinkingLevel: undefined });
      expect(trigger).toHaveAccessibleName(`Model: ${label} · Thinking: High (Default)`);
      if (surface === "mobile") await user.click(trigger);
      expect(screen.getByRole("menuitem", { name: "Use defaults" })).toBeInTheDocument();
    }
    expect(screen.getByRole("menuitem", { name: "vendor/unique" })).toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "Use defaults" }));
    expect(onChange).toHaveBeenLastCalledWith({});
    await user.click(trigger);
    expect(screen.queryByRole("menuitem", { name: "Use defaults" })).not.toBeInTheDocument();
  });

  it("disables the selector when Start is disabled", () => {
    render(<Selector disabled />);
    expect(screen.getByTestId("split-run-draft-model")).toBeDisabled();
  });

  it.each(["desktop", "mobile"])("refreshes catalog labels and saved defaults on %s", async (surface) => {
    const model = catalog[0].id;
    const props = {
      lineName: "ship",
      defaults: {
        values: { model, thinkingLevel: "high" },
        isLoading: false,
        isError: false,
        retry: vi.fn(),
      },
      onChange: vi.fn(),
    };
    const Component = surface === "mobile" ? MobileDraftStartModelSelect : DraftStartModelSelect;
    const { rerender } = render(<Component {...props} />);
    catalog = [{ id: model, name: "Updated model" }];
    rerender(<Component {...props} defaults={{ ...props.defaults, values: { model, thinkingLevel: "low" } }} />);
    expect(screen.getByTestId("split-run-draft-model")).toHaveAccessibleName(
      "Model: Updated model (Default) · Thinking: Low (Default)",
    );
    const user = userEvent.setup();
    await user.click(screen.getByTestId("split-run-draft-model"));
    if (surface === "desktop") await user.hover(screen.getByTestId("split-run-draft-model-list"));
    expect(await screen.findByRole("menuitem", { name: "Default (Updated model)" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Updated model" })).toBeInTheDocument();
  });

  describe.each(["desktop", "mobile"])("%s unresolved defaults", (surface) => {
    it.each([
      { model: "openai/gpt-6.1-sol", thinkingLevel: undefined },
      { model: undefined, thinkingLevel: "low" },
      { model: "openai/gpt-6.1-sol", thinkingLevel: "low" },
    ])("shows explicit choices with unavailable inherited fields: %j", (selection) => {
      const props = {
        ...selection,
        defaults: { values: undefined, isLoading: false, isError: true, retry: vi.fn() },
        onChange: vi.fn(),
      };
      render(
        surface === "mobile" ? (
          <MobileDraftStartModelSelect {...props} />
        ) : (
          <DraftStartModelSelect {...props} appearance="labeled" />
        ),
      );
      const trigger = screen.getByTestId("split-run-draft-model");
      const modelLabel = selection.model ? "GPT 6.1 Sol" : "Implementation defaults unavailable";
      const thinkingLabel = selection.thinkingLevel ? "Low" : "Implementation defaults unavailable";
      expect(trigger).toHaveTextContent(`${modelLabel}· ${thinkingLabel}`);
      expect(trigger).toHaveAccessibleName(new RegExp(`Model: ${modelLabel}.*Thinking: ${thinkingLabel}`));
    });
  });

  it.each([false, true])(
    "shows unresolved defaults and retries without creating an override when failed=%s",
    async (isError) => {
      const retry = vi.fn();
      const onChange = vi.fn();
      render(
        <DraftStartModelSelect
          defaults={{ values: undefined, isLoading: !isError, isError, retry }}
          onChange={onChange}
          appearance="labeled"
        />,
      );
      const trigger = screen.getByTestId("split-run-draft-model");
      expect(trigger).toHaveAccessibleName(
        isError ? "Implementation defaults unavailable" : "Loading Implementation defaults",
      );
      await userEvent.setup().click(trigger);
      await userEvent.setup().click(screen.getByRole("menuitem", { name: "Retry Implementation defaults" }));
      expect(retry).toHaveBeenCalledTimes(1);
      expect(onChange).not.toHaveBeenCalled();
    },
  );
});
