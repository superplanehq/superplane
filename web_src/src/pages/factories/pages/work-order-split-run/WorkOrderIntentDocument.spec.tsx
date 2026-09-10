import type { ReactElement } from "react";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME, confidenceSuitabilitySummary } from "../../lib/confidenceScore";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";

function renderDocument(ui: ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
}

let notifyResize: () => void;

class MockResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

const INTENT = {
  id: "art-intent",
  type: "TYPE_MARKDOWN" as const,
  data: {
    name: "intent.md",
    title: "intent.md",
    body: `# Clearer empty state

## Executive summary

The billing page empty state does not name the next action.

### Need
- The empty view only shows a title.

### Result
- The empty state tells the user how to add a payment method.

## Problem

The empty view only shows a title.

## Outcome

The empty state tells the user how to add a payment method.
`,
  },
};

describe("WorkOrderIntentDocument", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the original request and the generated summary", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        confidence={{
          id: "check-confidence",
          name: CONFIDENCE_CHECK_NAME,
          score: 4,
          maxScore: 5,
          level: "positive",
          summary: confidenceSuitabilitySummary("High"),
        }}
      />,
    );

    expect(screen.getByTestId("split-run-intent-session")).toHaveTextContent("Show a clearer empty state");
    expect(screen.queryByText("Original request")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-description")).toHaveTextContent(
      "Imported from GitHub: billing empty state is unclear.",
    );
    expect(screen.getByRole("heading", { name: "Clearer empty state" })).toBeInTheDocument();
    const summary = screen.getByTestId("split-run-intent-summary");
    expect(summary).toHaveTextContent("The billing page empty state does not name the next action.");
    expect(within(summary).getByRole("heading", { name: "Need" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Result" })).toBeInTheDocument();
    expect(within(summary).getAllByRole("listitem")).toHaveLength(2);
    const result = screen.getByTestId("split-run-intent-result");
    expect(within(result).getByTestId("split-run-overview-checks")).toHaveTextContent(CONFIDENCE_CHECK_NAME);
    expect(within(result).getByTestId("split-run-intent-confidence-copy")).toHaveTextContent(
      "This issue is a good fit for an agent on this factory line.",
    );
    expect(
      within(screen.getByTestId("split-run-intent-request")).queryByTestId("split-run-overview-checks"),
    ).toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan")).not.toBeInTheDocument();
  });

  it("keeps a decision note on the plan pane", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        resultFooter={<div data-testid="split-run-review">Ready</div>}
      />,
    );

    expect(within(screen.getByTestId("split-run-intent-result")).getByTestId("split-run-review")).toHaveTextContent(
      "Ready",
    );
    expect(within(screen.getByTestId("split-run-intent-request")).queryByTestId("split-run-review")).toBeNull();
  });

  it("switches to the full plan", async () => {
    const user = userEvent.setup();
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
      />,
    );

    await user.click(screen.getByRole("switch", { name: "Show the full plan" }));
    expect(screen.getByTestId("split-run-intent-plan")).toHaveTextContent("The empty state tells the user");
  });

  it("starts the request pane at one third width and lets the reader drag the split", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
      />,
    );

    const request = screen.getByTestId("split-run-intent-request");
    const handle = screen.getByTestId("split-run-intent-resize-handle");
    expect(request.style.getPropertyValue("--intent-left")).toBe("33%");
    expect(handle).toHaveAttribute("aria-label", "Resize the request and plan");

    const split = request.parentElement;
    expect(split).toBeTruthy();
    vi.spyOn(split as HTMLElement, "getBoundingClientRect").mockReturnValue({
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      bottom: 400,
      right: 1000,
      width: 1000,
      height: 400,
      toJSON: () => ({}),
    });

    fireEvent.pointerDown(handle, { clientX: 500, pointerId: 1 });
    fireEvent.pointerMove(window, { clientX: 620, pointerId: 1 });
    fireEvent.pointerUp(window, { clientX: 620, pointerId: 1 });
    expect(request.style.getPropertyValue("--intent-left")).toBe("62%");
  });

  it("collapses a long request and expands it on Show more", async () => {
    const user = userEvent.setup();
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description={"Imported from GitHub.\n\n" + "Need a payment method.\n".repeat(40)}
        artifacts={[INTENT]}
      />,
    );

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });
    act(() => notifyResize());

    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "220px" });

    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(screen.getByRole("button", { name: /show less/i })).toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });
});
