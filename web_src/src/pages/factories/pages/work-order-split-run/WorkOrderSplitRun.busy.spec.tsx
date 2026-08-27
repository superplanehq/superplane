import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type * as ApiClient from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_ID } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING } from "./splitRunMocks";

const { cancelRunMock } = vi.hoisted(() => ({
  cancelRunMock: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesCancelRun: (...args: unknown[]) => cancelRunMock(...args),
  };
});

function renderRunningPopup() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId={FACTORIES_ORGANIZATION_ID}
              factoryId={PRIMARY_FACTORY_ID}
              orderId="wo-running"
              fixture={SPLIT_RUN_RUNNING}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup action busy state", () => {
  beforeEach(() => {
    cancelRunMock.mockReset().mockReturnValue(new Promise(() => {}));
  });

  it("shares the busy state between automation Stop and header Reject", async () => {
    const user = userEvent.setup();
    renderRunningPopup();

    await user.click(screen.getByRole("button", { name: "Stop" }));
    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByRole("button", { name: "Stop" })).toBeDisabled();
    expect(
      within(screen.getByTestId("split-run-header-actions")).getByRole("button", { name: "Reject" }),
    ).toBeDisabled();
    expect(
      within(screen.getByTestId("split-run-header-actions")).getByRole("button", { name: "Approve" }),
    ).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Stop" }));
    expect(cancelRunMock).toHaveBeenCalledTimes(1);
  });
});
