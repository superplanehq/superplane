import { StrictMode } from "react";
import { act, fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME } from "../../lib/confidenceScore";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { ANALYSIS_REPLY_THINKING_STATES, ANALYSIS_THINKING_STATES } from "./analysisLiveWorkState";
import {
  analysisChat,
  HIGH_CONFIDENCE,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  notifyIntentResize,
  renderIntentDocument,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import { resetStreamMemoryForTests } from "./useStreamOnUpdate";
import type { SplitRunSource } from "./splitRunSource";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Ada Lovelace", initials: "AL" } : null,
    isLoading: false,
  }),
}));

const GITHUB_SOURCE: SplitRunSource = {
  kind: "intake",
  name: "GitHub issues",
  iconSrc: "/github.svg",
  iconAlt: "GitHub",
  ticket: { label: "acme/payments-service#842", href: "https://github.com/acme/payments-service/issues/842" },
};

describe("WorkOrderIntentDocument", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
  });

  it("shows the original request and the generated summary", () => {
    renderIntentDocument(<WorkOrderIntentDocument {...INTENT_DOC} artifacts={[INTENT]} confidence={HIGH_CONFIDENCE} />);

    expect(screen.getByTestId("split-run-intent-session")).toHaveTextContent("Show a clearer empty state");
    expect(screen.queryByText("Original request")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-description")).toHaveTextContent(
      "Imported from GitHub: billing empty state is unclear.",
    );
    expect(screen.getByRole("heading", { name: "Clearer empty state" })).toBeInTheDocument();
    const summary = screen.getByTestId("split-run-intent-summary");
    expect(summary.parentElement).not.toHaveAttribute("data-streaming");
    expect(summary).toHaveTextContent("A person can add a payment method from the empty billing page.");
    expect(within(summary).getByRole("heading", { name: "Goal" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Done when" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Out of scope" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Key architecture decisions" })).toBeInTheDocument();
    expect(within(summary).getAllByRole("listitem")).toHaveLength(4);
    const result = screen.getByTestId("split-run-intent-result");
    const chip = within(result).getByTestId("split-run-intent-confidence-chip");
    expect(chip).toHaveAccessibleName(`${CONFIDENCE_CHECK_NAME} 4/5`);
    expect(chip).toHaveTextContent("4/5");
    expect(screen.queryByTestId("split-run-intent-confidence-copy")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-intent-request")).queryByTestId("split-run-overview-checks"),
    ).toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan")).not.toBeInTheDocument();
  });

  it("reveals the confidence why when the chip is opened", async () => {
    const user = userEvent.setup();
    renderIntentDocument(<WorkOrderIntentDocument {...INTENT_DOC} artifacts={[INTENT]} confidence={HIGH_CONFIDENCE} />);

    await user.click(screen.getByTestId("split-run-intent-confidence-chip"));
    expect(await screen.findByTestId("split-run-intent-confidence-copy")).toHaveTextContent(
      "This issue is a good fit for an agent on this factory line.",
    );
  });

  it("puts source context on the left after Start and hides confidence", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
        contextSidebar={<aside data-testid="split-run-overview-sidebar">Source</aside>}
      />,
    );

    const request = screen.getByTestId("split-run-intent-request");
    const result = screen.getByTestId("split-run-intent-result");
    expect(within(request).getByTestId("split-run-overview-sidebar")).toHaveTextContent("Source");
    expect(screen.queryByTestId("split-run-intent-confidence-chip")).not.toBeInTheDocument();
    expect(within(result).queryByTestId("split-run-overview-checks")).toBeNull();
    expect(screen.queryByTestId("split-run-intent-chat")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(within(result).getByTestId("split-run-intent-summary")).toBeInTheDocument();
  });

  it("keeps a decision note on the plan pane", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
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
    renderIntentDocument(<WorkOrderIntentDocument {...INTENT_DOC} artifacts={[INTENT]} />);

    expect(screen.getByTestId("split-run-intent-summary")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show full plan" }));
    expect(screen.getByTestId("split-run-intent-plan")).toHaveTextContent("The empty state tells the user");
    expect(screen.getByTestId("split-run-intent-summary")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-plan-panel")).toBeInTheDocument();
  });

  it("starts the request pane at two fifths width and lets the reader drag the split", () => {
    renderIntentDocument(<WorkOrderIntentDocument {...INTENT_DOC} artifacts={[INTENT]} />);

    const request = screen.getByTestId("split-run-intent-request");
    const handle = screen.getByTestId("split-run-intent-resize-handle");
    expect(request.style.getPropertyValue("--intent-left")).toBe("40%");
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

  it("shows the request as the first chat message on the right", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        source={GITHUB_SOURCE}
        analysis={analysisChat({ composer: "Need the existing empty-state component." })}
      />,
    );

    const chat = within(screen.getByTestId("split-run-intent-request")).getByTestId("split-run-intent-chat");
    expect(chat).toBeInTheDocument();
    expect(within(chat).queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(within(chat).getByTestId("split-run-description")).toHaveClass("justify-end");
    expect(within(chat).getByTestId("split-run-source")).toHaveTextContent("acme/payments-service#842");
    expect(within(chat).queryByText(CREATE_WITH_AGENT_COPY.request)).not.toBeInTheDocument();
    expect(within(chat).queryByText(CREATE_WITH_AGENT_COPY.you)).not.toBeInTheDocument();
    expect(within(chat).getByTestId("split-run-description").querySelector(".sp-user-note")).not.toBeNull();
    expect(screen.getByTestId("split-run-intent-composer").closest(".sp-user-note")).not.toBeNull();
    expect(within(chat).getByTestId("split-run-description")).toHaveTextContent(
      "Imported from GitHub: billing empty state is unclear.",
    );
    expect(within(chat).getByTestId("split-run-intent-thinking")).toHaveTextContent(ANALYSIS_THINKING_STATES[0]);
    expect(within(chat).queryByTestId("split-run-phase-planning")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-summary")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show full plan" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-plan")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-decision")).toHaveClass("min-h-[5.5rem]");
    expect(screen.getByTestId("split-run-intent-composer").closest("form")).toHaveClass("min-h-[5.5rem]");
    expect(within(screen.getByTestId("split-run-intent-result")).queryByTestId("split-run-intent-chat")).toBeNull();
    expect(screen.getByTestId("split-run-intent-composer")).toHaveValue("Need the existing empty-state component.");
  });

  it("streams the summary when the spec updates after open", () => {
    const { rerender } = renderIntentDocument(
      <StrictMode>
        <WorkOrderIntentDocument
          {...INTENT_DOC}
          streamKey="order-stream"
          artifacts={[{ ...INTENT, data: { ...INTENT.data, body: "# Old title\n\nOld summary only.\n" } }]}
        />
      </StrictMode>,
    );

    expect(screen.getByTestId("split-run-intent-summary").parentElement).not.toHaveAttribute("data-streaming");

    rerender(
      <TooltipProvider>
        <StrictMode>
          <WorkOrderIntentDocument {...INTENT_DOC} streamKey="order-stream" artifacts={[INTENT]} />
        </StrictMode>
      </TooltipProvider>,
    );

    expect(screen.getByTestId("split-run-intent-summary").parentElement).toHaveAttribute("data-streaming");
    expect(screen.getByRole("heading", { name: "Goal" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Out of scope" })).not.toBeInTheDocument();
  });

  it("streams the first spec that arrives while the card is open", () => {
    const { rerender } = renderIntentDocument(
      <StrictMode>
        <WorkOrderIntentDocument {...INTENT_DOC} streamKey="order-first-spec" artifacts={[]} isAnalyzing />
      </StrictMode>,
    );

    expect(screen.getByTestId("split-run-intent-summary").parentElement).not.toHaveAttribute("data-streaming");

    rerender(
      <TooltipProvider>
        <StrictMode>
          <WorkOrderIntentDocument {...INTENT_DOC} streamKey="order-first-spec" artifacts={[INTENT]} isAnalyzing />
        </StrictMode>
      </TooltipProvider>,
    );

    expect(screen.getByTestId("split-run-intent-summary").parentElement).toHaveAttribute("data-streaming");
    expect(screen.queryByRole("heading", { name: "Out of scope" })).not.toBeInTheDocument();
  });

  it("does not stream when artifacts load after the card opens", () => {
    const { rerender } = renderIntentDocument(
      <StrictMode>
        <WorkOrderIntentDocument {...INTENT_DOC} streamKey="order-open" streamReady={false} artifacts={[]} />
      </StrictMode>,
    );

    rerender(
      <TooltipProvider>
        <StrictMode>
          <WorkOrderIntentDocument {...INTENT_DOC} streamKey="order-open" streamReady artifacts={[INTENT]} />
        </StrictMode>
      </TooltipProvider>,
    );

    expect(screen.getByTestId("split-run-intent-summary").parentElement).not.toHaveAttribute("data-streaming");
    expect(screen.getByRole("heading", { name: "Out of scope" })).toBeInTheDocument();
  });

  it("keeps the stored transcript outside the current run activity", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [
              { id: "user-1", kind: "text", role: "user", text: "Use the current empty-state component." },
              { id: "agent-1", kind: "text", role: "agent", text: "I updated the plan with that constraint." },
              { id: "survey-1", kind: "text", role: "user", origin: "survey", text: "What is the priority? High" },
            ],
          },
        })}
      />,
    );

    const transcript = screen.getByTestId("split-run-intent-transcript");
    expect(within(transcript).getByText("Use the current empty-state component.")).toBeInTheDocument();
    expect(within(transcript).getByText("I updated the plan with that constraint.")).toBeInTheDocument();
    expect(within(transcript).getByText("What is the priority?")).toBeInTheDocument();
    expect(within(transcript).queryByText(CREATE_WITH_AGENT_COPY.youSurvey)).not.toBeInTheDocument();
    expect(screen.getAllByText("Use the current empty-state component.")).toHaveLength(1);
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-planning")).not.toBeInTheDocument();
  });

  it("does not restream the last agent line after a survey answer", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          canSend: false,
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [
              {
                id: "agent-1",
                kind: "text",
                role: "agent",
                text: "Because the premise is false, I scored this low (confidence 2) and asked one clarifying question: close it as already fixed with a test, upgrade the plain-text 404 into a real page, or provide steps where the 500 still occurs. Spec, score, and survey are published. Files written: /tmp/intake-analysis.json and /tmp/intent.md.",
              },
              {
                id: "survey-1",
                kind: "text",
                role: "user",
                origin: "survey",
                text: "The delete route already returns 404 for a missing puppy. Add a test and close the ticket.",
              },
            ],
          },
        })}
      />,
    );

    const transcript = screen.getByTestId("split-run-intent-transcript");
    expect(within(transcript).getByText(/Because the premise is false/)).toBeInTheDocument();
    expect(transcript.querySelector(".sp-stream-text")).toBeNull();
    expect(screen.getAllByText(/Because the premise is false/)).toHaveLength(1);
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent(ANALYSIS_REPLY_THINKING_STATES[0]);
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.machineStarting)).not.toBeInTheDocument();
  });

  it("does not send a multi-question survey when Next is clicked", async () => {
    const user = userEvent.setup();
    const onSubmitSurvey = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          onSubmitSurvey,
          view: {
            machineStatus: "waiting",
            survey: {
              id: "survey-1",
              questions: [
                { prompt: "What is the priority?", options: ["High", "Low"] },
                { prompt: "What is the scope?", options: ["One file", "The service"] },
              ],
            },
          },
        })}
      />,
    );

    expect(
      within(screen.getByTestId("split-run-intent-chat-log")).getByTestId("create-with-agent-survey"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.nextQuestion }));

    expect(onSubmitSurvey).not.toHaveBeenCalled();
    expect(screen.getByText("What is the scope?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.sendAnswers })).toBeDisabled();
  });

  it("streams the analysis agent in the left chat", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: { machineStatus: "running", canvasId: "canvas-1", canvasRunId: "run-1", executionId: "exec-1" },
        })}
      />,
    );

    expect(
      within(screen.getByTestId("split-run-intent-chat")).getByTestId("split-run-intent-thinking"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-planning")).not.toBeInTheDocument();
  });

  it("collapses a long request and expands it on Show more", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        description={"Imported from GitHub.\n\n" + "Need a payment method.\n".repeat(40)}
        artifacts={[INTENT]}
      />,
    );

    const content = screen.getByTestId("work-order-description-markdown").parentElement;
    Object.defineProperty(content!, "scrollHeight", { configurable: true, get: () => 640 });
    act(() => notifyIntentResize());

    expect(screen.getByRole("button", { name: /show more/i })).toBeInTheDocument();
    expect(content).toHaveStyle({ maxHeight: "220px" });

    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(screen.getByRole("button", { name: /show less/i })).toBeInTheDocument();
    expect(content).not.toHaveStyle({ maxHeight: "220px" });
  });

  it("focuses the plan pane when the plan-updated banner is clicked", async () => {
    const user = userEvent.setup();
    const scrollIntoView = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 4 }],
          },
        })}
      />,
    );

    const result = screen.getByTestId("split-run-intent-result");
    result.scrollIntoView = scrollIntoView;
    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.planUpdated }));
    expect(scrollIntoView).toHaveBeenCalled();
  });
});
