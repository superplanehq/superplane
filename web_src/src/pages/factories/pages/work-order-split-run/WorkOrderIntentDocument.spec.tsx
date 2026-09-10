import type { ReactElement } from "react";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME, confidenceSuitabilitySummary } from "../../lib/confidenceScore";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";

function renderDocument(ui: ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
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
    expect(screen.getByTestId("split-run-overview-checks")).toHaveTextContent(CONFIDENCE_CHECK_NAME);
    expect(screen.getByTestId("split-run-intent-confidence-copy")).toHaveTextContent(
      "This issue is a good fit for an agent on this factory line.",
    );
    expect(screen.queryByTestId("split-run-intent-plan")).not.toBeInTheDocument();
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
});
