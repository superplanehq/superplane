import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry, type WorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderLineStep } from "./WorkOrderLineStep";

const factory: FactoriesFactory = { id: "f", name: "Refunds", key: "RF" };

function entry(overrides: Partial<FactoriesWorkOrder> = {}): WorkOrderListEntry {
  return buildWorkOrderListEntry(
    {
      id: "wo-1",
      title: "Order title",
      state: "STATE_OPEN",
      createdAt: "2024-06-01T00:00:00Z",
      updatedAt: "2024-06-02T00:00:00Z",
      executions: [],
      ...overrides,
    },
    factory,
  );
}

const running = entry({
  executions: [{ id: "e1", step: "verify", state: "STATE_STARTED", line: { id: "line-a", name: "hotfix" } }],
});

const finished = entry({
  executions: [
    {
      id: "e1",
      step: "plan",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      line: { id: "line-a", name: "review" },
    },
  ],
});

describe("WorkOrderLineStep", () => {
  it("shows the shared caption and spins only while running", () => {
    const { container: runningContainer } = render(<WorkOrderLineStep entry={running} />);
    expect(screen.getByText("hotfix · verify")).toBeInTheDocument();
    expect(runningContainer.querySelector(".animate-spin")).not.toBeNull();

    const { container: finishedContainer } = render(<WorkOrderLineStep entry={finished} />);
    expect(screen.getByText("review · plan")).toBeInTheDocument();
    expect(finishedContainer.querySelector(".animate-spin")).toBeNull();
  });

  it("renders nothing before the first run unless a fallback is given", () => {
    const { container } = render(<WorkOrderLineStep entry={entry()} />);
    expect(container).toBeEmptyDOMElement();

    render(<WorkOrderLineStep entry={entry()} fallback="—" />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  // The table hides this column below `md`, so the caller's display class
  // has to win over the component's own `inline-flex`.
  it("lets the caller override the display class", () => {
    const { container } = render(<WorkOrderLineStep entry={running} className="hidden md:inline-flex" />);
    const caption = container.firstElementChild;

    expect(caption).toHaveClass("hidden", "md:inline-flex");
    expect(caption).not.toHaveClass("inline-flex");
  });
});
