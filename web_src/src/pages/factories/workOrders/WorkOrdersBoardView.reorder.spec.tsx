import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrdersBoardView } from "./WorkOrdersBoardView";

const organizationId = "org-1";
const factoryKey = "RF";
const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: factoryKey };

function draftEntry(id: string, title: string) {
  return buildWorkOrderListEntry({ id, number: id, title, state: "STATE_DRAFT" } satisfies FactoriesWorkOrder, factory);
}

const entries = [draftEntry("wo-1", "First"), draftEntry("wo-2", "Second")];

const sharedProps = {
  organizationId,
  factoryKey,
  factoryLines: [],
  canDispatch: true,
  canAssign: true,
  dispatchingOrderIds: new Set<string>(),
  isAssigneesSaving: false,
  onDispatch: vi.fn().mockResolvedValue(undefined),
  onAssigneesSave: vi.fn().mockResolvedValue(undefined),
};

function cardListItem(title: string) {
  return screen.getByRole("link", { name: `Open ${title}` }).closest("li") as HTMLElement;
}

describe("WorkOrdersBoardView drag-and-drop", () => {
  it("does not make cards draggable without a manual ordering + onReorder handler", () => {
    render(
      <MemoryRouter>
        <WorkOrdersBoardView {...sharedProps} entries={entries} />
      </MemoryRouter>,
    );

    expect(cardListItem("First")).not.toHaveAttribute("aria-roledescription");
  });

  it("does not make cards draggable under a non-manual ordering, even with a handler", () => {
    render(
      <MemoryRouter>
        <WorkOrdersBoardView {...sharedProps} entries={entries} ordering="updated" onReorder={vi.fn()} />
      </MemoryRouter>,
    );

    expect(cardListItem("First")).not.toHaveAttribute("aria-roledescription");
  });

  it("makes cards draggable in manual ordering with a handler", () => {
    render(
      <MemoryRouter>
        <WorkOrdersBoardView {...sharedProps} entries={entries} ordering="manual" onReorder={vi.fn()} />
      </MemoryRouter>,
    );

    const card = cardListItem("First");
    expect(card).toHaveAttribute("aria-roledescription", "sortable");
    expect(card).toHaveAttribute("tabIndex", "0");
  });
});
