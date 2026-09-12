import type { ReactElement } from "react";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME, confidenceSuitabilitySummary } from "../../lib/confidenceScore";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
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

### Goal

A person can add a payment method from the empty billing page.

The agent reads this as copy and an action on the current empty view. It does not read it as a new billing flow.

### Done when

- The empty view names the next action.
- The action opens add-payment-method.

### Out of scope

- The page after a card exists.

### Key architecture decisions

- Reuse the current empty view. Do not add a new page.

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
    expect(summary).toHaveTextContent("A person can add a payment method from the empty billing page.");
    expect(within(summary).getByRole("heading", { name: "Goal" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Done when" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Out of scope" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Key architecture decisions" })).toBeInTheDocument();
    expect(within(summary).getAllByRole("listitem")).toHaveLength(4);
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

  it("puts source context and confidence on the left after Start", () => {
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
        contextSidebar={<aside data-testid="split-run-overview-sidebar">Source</aside>}
      />,
    );

    const request = screen.getByTestId("split-run-intent-request");
    const result = screen.getByTestId("split-run-intent-result");
    expect(within(request).getByTestId("split-run-overview-sidebar")).toHaveTextContent("Source");
    expect(within(request).getByTestId("split-run-overview-checks")).toHaveTextContent(CONFIDENCE_CHECK_NAME);
    expect(within(result).queryByTestId("split-run-overview-checks")).toBeNull();
    expect(screen.queryByTestId("split-run-intent-chat")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(within(result).getByTestId("split-run-intent-summary")).toBeInTheDocument();
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

    expect(screen.getByTestId("split-run-intent-summary")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show full plan" }));
    expect(screen.getByTestId("split-run-intent-plan")).toHaveTextContent("The empty state tells the user");
    expect(screen.getByTestId("split-run-intent-summary")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-plan-panel")).toBeInTheDocument();
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

  it("shows the request as the first chat message on the left", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        analysis={{
          organizationId: "org-1",
          view: {
            repository: "acme/payments",
            machineStatus: "starting",
            canvasId: "",
            canvasRunId: "",
            executionId: "",
            messages: [],
            composer: "",
            created: [],
            right: { kind: "empty" },
            endConfirmOpen: false,
            selectableModelKey: "",
            refining: false,
          },
          composer: "Need the existing empty-state component.",
          canSend: true,
          onComposerChange: vi.fn(),
          onSend: vi.fn(),
          onSubmitSurvey: vi.fn(),
        }}
      />,
    );

    const chat = within(screen.getByTestId("split-run-intent-request")).getByTestId("split-run-intent-chat");
    expect(chat).toBeInTheDocument();
    expect(within(chat).getByText("You")).toBeInTheDocument();
    expect(within(chat).getByTestId("split-run-description")).toHaveTextContent(
      "Imported from GitHub: billing empty state is unclear.",
    );
    expect(within(chat).getByText("The agent is writing the plan.")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-intent-result")).queryByTestId("split-run-intent-chat")).toBeNull();
    expect(screen.getByTestId("split-run-intent-composer")).toHaveValue("Need the existing empty-state component.");
  });

  it("keeps the stored transcript outside the current run activity", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        analysis={{
          organizationId: "org-1",
          view: {
            repository: "acme/payments",
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [
              { id: "user-1", kind: "text", role: "user", text: "Use the current empty-state component." },
              { id: "agent-1", kind: "text", role: "agent", text: "I updated the plan with that constraint." },
              {
                id: "survey-1",
                kind: "text",
                role: "user",
                origin: "survey",
                text: "What is the priority? High",
              },
            ],
            composer: "",
            created: [],
            right: { kind: "empty" },
            endConfirmOpen: false,
            selectableModelKey: "",
            refining: false,
          },
          composer: "",
          canSend: true,
          onComposerChange: vi.fn(),
          onSend: vi.fn(),
          onSubmitSurvey: vi.fn(),
        }}
      />,
    );

    const transcript = screen.getByTestId("split-run-intent-transcript");
    expect(within(transcript).getByText("Use the current empty-state component.")).toBeInTheDocument();
    expect(within(transcript).getByText("I updated the plan with that constraint.")).toBeInTheDocument();
    expect(within(transcript).getByText(CREATE_WITH_AGENT_COPY.youSurvey)).toBeInTheDocument();
    expect(screen.getAllByText("Use the current empty-state component.")).toHaveLength(1);
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-planning")).not.toBeInTheDocument();
  });

  it("does not send a multi-question survey when Next is clicked", async () => {
    const user = userEvent.setup();
    const onSubmitSurvey = vi.fn();
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        analysis={{
          organizationId: "org-1",
          view: {
            repository: "acme/payments",
            machineStatus: "waiting",
            canvasId: "",
            canvasRunId: "",
            executionId: "",
            messages: [],
            composer: "",
            created: [],
            right: { kind: "empty" },
            endConfirmOpen: false,
            selectableModelKey: "",
            refining: false,
            survey: {
              id: "survey-1",
              questions: [
                { prompt: "What is the priority?", options: ["High", "Low"] },
                { prompt: "What is the scope?", options: ["One file", "The service"] },
              ],
            },
          },
          composer: "",
          canSend: true,
          onComposerChange: vi.fn(),
          onSend: vi.fn(),
          onSubmitSurvey,
        }}
      />,
    );

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));

    expect(onSubmitSurvey).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers })).toBeDisabled();
  });

  it("streams the analysis agent in the left chat", () => {
    renderDocument(
      <WorkOrderIntentDocument
        title="Show a clearer empty state"
        description="Imported from GitHub: billing empty state is unclear."
        artifacts={[INTENT]}
        analysis={{
          organizationId: "org-1",
          view: {
            repository: "acme/payments",
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [],
            composer: "",
            created: [],
            right: { kind: "empty" },
            endConfirmOpen: false,
            selectableModelKey: "",
            refining: false,
          },
          composer: "",
          canSend: true,
          onComposerChange: vi.fn(),
          onSend: vi.fn(),
          onSubmitSurvey: vi.fn(),
        }}
      />,
    );

    expect(
      within(screen.getByTestId("split-run-intent-chat")).getByTestId("split-run-phase-planning"),
    ).toBeInTheDocument();
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
