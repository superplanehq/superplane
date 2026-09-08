import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
} from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const duplicateMutate = vi.fn();

vi.mock("@/hooks/useFactoryData", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    useDuplicateWorkOrder: () => ({ mutateAsync: duplicateMutate, isPending: false }),
  };
});

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPopup(canCreate = true, onCreated = vi.fn()) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId={FACTORIES_ORGANIZATION_ID}
              factoryId={PRIMARY_FACTORY_ID}
              orderId={DRAFT_WORK_ORDER.id}
              fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
              canCreate={canCreate}
              onCreated={onCreated}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onCreated };
}

describe("WorkOrderSplitRunPopup duplicate", () => {
  beforeEach(() => {
    duplicateMutate.mockReset();
    duplicateMutate.mockResolvedValue({ id: "wo-copy", number: "201" });
  });

  it("duplicates the task and opens the copy", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderPopup();

    await user.click(screen.getByTestId("work-order-actions-button"));
    await user.click(screen.getByTestId("work-order-duplicate-button"));

    expect(duplicateMutate).toHaveBeenCalledWith(DRAFT_WORK_ORDER.id);
    expect(onCreated).toHaveBeenCalledWith("201");
  });

  it("disables Duplicate without create permission", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderPopup(false);

    await user.click(screen.getByTestId("work-order-actions-button"));
    expect(screen.getByTestId("work-order-duplicate-button")).toHaveAttribute("data-disabled");
    await user.click(screen.getByTestId("work-order-duplicate-button"));
    expect(duplicateMutate).not.toHaveBeenCalled();
    expect(onCreated).not.toHaveBeenCalled();
  });
});
