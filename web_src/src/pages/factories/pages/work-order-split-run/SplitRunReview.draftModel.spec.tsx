import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "bun:test";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { MobileDraftStartModelSelect } from "../../mobile/MobileDraftStartModelSelect";
import type { DraftStartSelection } from "./draftStartModel";
import { SplitRunReview } from "./SplitRunReview";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: { models: [{ id: "openai/gpt-6.1-sol", name: "GPT 6.1 Sol" }] },
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
    expect(trigger).toHaveAccessibleName(/openai\/gpt-6.1-sol.*Implementation default.*High.*Implementation default/);
    await user.click(trigger);
    await choose(user, "split-run-draft-model-list", "GPT 6.1 Sol (openai/gpt-6.1-sol)");
    expect(trigger).toHaveAccessibleName(/GPT 6.1 Sol.*Thinking: High \(Implementation default\)/);
    await choose(user, "split-run-draft-thinking", "Low");
    expect(trigger).toHaveAccessibleName(/Thinking: Low$/);
    await choose(user, "split-run-draft-model-list", "Implementation default");
    expect(trigger).toHaveAccessibleName(/Implementation default.*Thinking: Low$/);
    await user.click(screen.getByRole("menuitem", { name: "Use Implementation defaults" }));
    expect(trigger).toHaveAccessibleName(/High \(Implementation default\)/);
  });

  it("disables the selector when Start is disabled", () => {
    render(<Selector disabled />);
    expect(screen.getByTestId("split-run-draft-model")).toBeDisabled();
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
