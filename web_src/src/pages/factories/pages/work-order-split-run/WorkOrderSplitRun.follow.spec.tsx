import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...props} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup Follow", () => {
  it("starts Follow on while a phase is running", () => {
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    expect(screen.getByRole("switch", { name: "Follow" })).toBeChecked();
    const follow = screen.getByTestId("split-run-follow");
    expect(follow.className).toMatch(/ml-auto/);
    expect(
      screen.getByRole("tab", { name: "Automations" }).compareDocumentPosition(follow) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
  });

  it("hides Follow on the Description tab", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    await user.click(screen.getByRole("tab", { name: "Description" }));

    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
  });

  it("starts Follow off when the run is finished", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });
    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(screen.getByRole("switch", { name: "Follow" })).not.toBeChecked();
  });

  it("turns Follow off when the user clicks the switch", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    await user.click(screen.getByRole("switch", { name: "Follow" }));

    expect(screen.getByRole("switch", { name: "Follow" })).not.toBeChecked();
  });

  it("explains what Follow does in a tooltip", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: SPLIT_RUN_RUNNING });

    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    const follow = screen.getByTestId("split-run-follow");
    await user.hover(follow);

    const tooltip = await screen.findByRole("tooltip");
    expect(tooltip).toHaveTextContent("Auto-scroll the log to the newest output.");
    expect(follow).toHaveAttribute("aria-describedby", tooltip.id);
    expect(screen.getByRole("switch", { name: "Follow" })).toBeInTheDocument();
  });
});
