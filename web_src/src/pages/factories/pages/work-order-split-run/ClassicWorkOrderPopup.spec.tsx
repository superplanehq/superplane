import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { ClassicWorkOrderPopup } from "./ClassicWorkOrderPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";

function renderClassicPopup(workOrder = DRAFT_WORK_ORDER) {
  const fixture = splitRunFixtureForWorkOrder(workOrder);
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <ClassicWorkOrderPopup
              factoryKey="RF"
              orderNumber={workOrder.number}
              orderId={workOrder.id}
              fixture={fixture}
              canDispatch
              popupData={{
                artifacts: [],
                pullRequests: [],
                sourceDescription: fixture.descriptionText ?? "",
                useLive: false,
                artifactsLoading: false,
                pullRequestsLoading: false,
                pullRequestsError: null,
              }}
              sessionLookupError={null}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ClassicWorkOrderPopup", () => {
  it("restores the complete classic task view without refinement controls", () => {
    renderClassicPopup();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).not.toContain("w-[min(80rem");
    for (const className of SPLIT_RUN_POPUP_DIALOG_CLASSNAME.split(/\s+/)) {
      expect(dialog).not.toHaveClass(className);
    }

    expect(within(dialog).getByRole("tab", { name: "Task" })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Automations" })).toBeInTheDocument();
    expect(within(dialog).getByRole("region", { name: "Source" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Artifacts" })).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Pull requests" })).toBeInTheDocument();
    expect(within(dialog).getByText("This task is ready to start")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Archive" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
    expect(within(dialog).queryByText("Add context for this plan")).not.toBeInTheDocument();
  });

  it("shows the workspace-scoped task ID next to the title", () => {
    renderClassicPopup();

    const dialog = screen.getByTestId("work-order-split-run");
    const displayKey = within(dialog).getByTestId("popup-work-order-display-key");
    const title = within(dialog).getByTestId("popup-work-order-title");

    expect(displayKey).toHaveTextContent("RF-105");
    expect(displayKey).toHaveAccessibleName("Copy task ID RF-105");
    expect(title.compareDocumentPosition(displayKey) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("puts the pull request review note in the left context column", () => {
    renderClassicPopup(OPEN_WORK_ORDER);

    const dialog = screen.getByTestId("work-order-split-run");
    const tab = within(dialog).getByTestId("split-run-work-order-tab");
    const sidebar = within(tab).getByTestId("split-run-overview-sidebar");
    const note = within(sidebar).getByTestId("split-run-attention-note");

    expect(tab.firstElementChild).toBe(sidebar);
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
  });
});
