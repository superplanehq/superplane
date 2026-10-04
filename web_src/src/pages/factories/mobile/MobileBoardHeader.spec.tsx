import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrdersPage: () => ({ data: [], isLoading: false }),
  useWorkOrderArtifacts: () => ({ data: [], isLoading: false }),
  useSendWorkOrderToBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

import { REFUND_FACTORY_LINES, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import { HostedCreditHeaderKicker } from "../HostedCreditHeaderKicker";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { MobileBoardHeader } from "./MobileBoardHeader";

const PHONE_HEADER_WIDTH_PX = 390;

function PhoneBoardHeader() {
  const state = useWorkOrderListState("factory-1");
  const expiresAt = new Date(Date.now() + 13 * 24 * 60 * 60 * 1000);

  return (
    <div data-testid="phone-board-header-frame" style={{ width: PHONE_HEADER_WIDTH_PX }}>
      <MobileBoardHeader
        state={state}
        searchRef={createRef()}
        creditKicker={
          <HostedCreditHeaderKicker
            compact
            spendingHref="/org/workspaces/RF/settings/organization/billing"
            welcomeCreditExpiresAt={expiresAt.toISOString()}
            remainingCreditCents={4124}
          />
        }
        sourceOptions={[]}
        assigneeOptions={[]}
        showPullRequestMerge={false}
        colorView="vivid"
        onColorViewChange={vi.fn()}
        intakes={[]}
        prFeedbackHandlers={[]}
        onOpenIntake={vi.fn()}
        onOpenPRFeedback={vi.fn()}
        lines={REFUND_FACTORY_LINES}
        lineId={REFUND_LINE_PLAN_ID}
        onSelectLine={vi.fn()}
        organizationId="org-1"
        factoryId="factory-1"
        factoryKey="RF"
        canManageClosedStatus={false}
      />
    </div>
  );
}

function creditSlot(row: HTMLElement, chip: HTMLElement): HTMLElement {
  const slot = Array.from(row.children).find((child) => child.contains(chip));
  if (!(slot instanceof HTMLElement)) {
    throw new Error("credit chip is not in the phone header row");
  }
  return slot;
}

describe("MobileBoardHeader credit chip", () => {
  it("keeps the trial balance in a phone-width row beside the line switcher and controls", () => {
    render(
      <MemoryRouter>
        <PhoneBoardHeader />
      </MemoryRouter>,
    );

    const row = screen.getByTestId("mobile-board-header");
    const chip = screen.getByTestId("hosted-credit-header-kicker");
    const slot = creditSlot(row, chip);
    const switcher = screen.getByTestId("mobile-board-line-switcher");
    const filter = screen.getByTestId("work-orders-filter-trigger");
    const search = screen.getByTestId("mobile-board-search-toggle");
    const view = screen.getByTestId("lines-board-view-menu");
    const menu = screen.getByTestId("mobile-board-menu");

    const frame = screen.getByTestId("phone-board-header-frame");
    expect(frame).toContainElement(row);
    expect(frame.getAttribute("style")).toBe(`width: ${PHONE_HEADER_WIDTH_PX}px;`);
    expect(chip).toHaveTextContent("Trial");
    expect(chip).toHaveTextContent("$41.24");
    expect(chip).not.toHaveTextContent("Subscribe");
    expect(switcher.compareDocumentPosition(chip) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(chip.compareDocumentPosition(filter) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(row).toContainElement(search);
    expect(row).toContainElement(view);
    expect(row).toContainElement(menu);

    expect(slot).toHaveClass("min-w-0", "flex-1", "overflow-hidden");
    expect(slot.className).not.toContain("overflow-x-auto");
    expect(chip).toHaveClass("min-w-0", "max-w-full", "overflow-hidden");
    expect(chip).not.toHaveClass("shrink-0");
    expect(chip.querySelector(".truncate")).not.toBeNull();
    expect(filter.parentElement).toHaveClass("shrink-0");
  });
});
