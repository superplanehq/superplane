import { StrictMode } from "react";
import { act, fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME } from "../../lib/confidenceScore";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
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
import { REFINE_LAYOUT_STORAGE_KEY } from "./refineLayoutPreference";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
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
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
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
    expect(within(summary).getByRole("heading", { name: "Problem" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Proposed outcome" })).toBeInTheDocument();
    expect(within(summary).getByRole("heading", { name: "Constraints" })).toBeInTheDocument();
    expect(within(summary).queryByRole("heading", { name: "Scope" })).not.toBeInTheDocument();
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

  it("shows the request as the first chat message on the right", async () => {
    const user = userEvent.setup();
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
    expect(screen.queryByTestId("split-run-intent-status-card")).not.toBeInTheDocument();
    const pendingChips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(pendingChips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.clarity })).toBeInTheDocument();
    expect(within(pendingChips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).toBeInTheDocument();
    expect(within(pendingChips).getByTestId("split-run-intent-plan-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(within(pendingChips).getByTestId("split-run-intent-plan-chip-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-composer-score")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence-copy")).not.toBeInTheDocument();
    await user.click(within(pendingChips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.clarity }));
    expect(screen.queryByTestId("split-run-intent-confidence-copy")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Build" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Model/ })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-composer-card")).toHaveAttribute("data-slot", "input-group");
    expect(screen.getByTestId("split-run-intent-chat-log").className).toContain("[scrollbar-gutter:stable]");
    expect(within(chat).getByTestId("split-run-description")).toHaveTextContent(
      "Imported from GitHub: billing empty state is unclear.",
    );
    expect(within(chat).getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting analysis…");
    expect(within(chat).queryByTestId("split-run-phase-planning")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByTestId("split-run-intent-resize-handle")).toHaveClass("lg:hidden");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    const columns = screen.getAllByTestId("split-run-intent-chat-column");
    expect(columns.length).toBeGreaterThanOrEqual(2);
    for (const column of columns) {
      expect(column).toHaveClass("mx-auto", "max-w-5xl", "px-4");
    }
    expect(columns[0]).toContainElement(screen.getByTestId("split-run-description"));
    expect(columns[columns.length - 1]).toContainElement(screen.getByTestId("split-run-intent-composer"));
    expect(screen.getByTestId("split-run-intent-composer").closest("[data-slot=input-group]")).not.toBeNull();
    expect(columns[columns.length - 1].className).toContain("[scrollbar-gutter:stable]");
    expect(screen.getByTestId("split-run-intent-composer").closest("form")).toHaveClass("min-h-[5.5rem]");
    expect(screen.getByTestId("split-run-intent-composer")).toHaveValue("Need the existing empty-state component.");
    expect(screen.getByTestId("split-run-intent-composer-kbd")).toHaveTextContent(ANALYSIS_PLANNING_COPY.sendShortcut);
    const send = screen.getByTestId("split-run-intent-composer-send");
    expect(send).toHaveAttribute("data-size", "icon-sm");
    expect(send).toHaveAttribute("aria-label", ANALYSIS_PLANNING_COPY.send);
    expect(send.children).toHaveLength(1);
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
    expect(screen.getByTestId("split-run-intent-summary")).toHaveTextContent("A");
    expect(screen.queryByRole("heading", { name: "Constraints" })).not.toBeInTheDocument();
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
    expect(screen.queryByRole("heading", { name: "Constraints" })).not.toBeInTheDocument();
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
    expect(screen.getByRole("heading", { name: "Constraints" })).toBeInTheDocument();
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
    expect(transcript.querySelector(".sp-stream-w")).toBeNull();
    expect(screen.getAllByText(/Because the premise is false/)).toHaveLength(1);
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting analysis…");
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
            messages: [{ id: "agent-1", kind: "text", role: "agent", text: "I need two details." }],
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

  it("waits for the final agent message before it shows a survey", () => {
    const survey = {
      id: "survey-1",
      questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
    };
    const { rerender } = renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            survey,
          },
        })}
      />,
    );

    expect(screen.queryByTestId("create-with-agent-survey")).not.toBeInTheDocument();

    rerender(
      <TooltipProvider>
        <WorkOrderIntentDocument
          {...INTENT_DOC}
          artifacts={[INTENT]}
          analysis={analysisChat({
            view: {
              machineStatus: "waiting",
              canvasId: "canvas-1",
              canvasRunId: "run-1",
              executionId: "exec-1",
              messages: [{ id: "agent-1", kind: "text", role: "agent", text: "I need one detail." }],
              survey,
            },
          })}
        />
      </TooltipProvider>,
    );

    const log = screen.getByTestId("split-run-intent-chat-log");
    const message = within(log).getByTestId("split-run-intent-transcript");
    expect(message).toHaveTextContent("I need one detail.");
    const question = within(log).getByText("What is the priority?");
    expect(message.compareDocumentPosition(question) & Node.DOCUMENT_POSITION_FOLLOWING).not.toBe(0);
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

  it("keeps the latest plan sticky and toggles the spec column", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
        resultFooter={<div data-testid="split-run-review">Ready</div>}
        analysis={analysisChat({
          view: {
            machineStatus: "waiting",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [
              { id: "agent-1", kind: "text", role: "agent", text: "I published the spec." },
              { id: "plan-1", kind: "plan", role: "plan", score: 4 },
            ],
          },
        })}
      />,
    );

    const strip = screen.getByTestId("split-run-intent-plan-updated");
    expect(
      within(screen.getByTestId("split-run-intent-transcript")).queryByTestId("split-run-intent-plan-updated"),
    ).toBeNull();
    const chips = screen.getByTestId("split-run-intent-composer-chips");
    const showPlan = within(chips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.plan });
    expect(within(chips).getByTestId("split-run-intent-composer-score")).toHaveAccessibleName("Clarity 4/5");
    expect(within(chips).getByTestId("split-run-intent-plan-status")).toHaveTextContent(
      CREATE_WITH_AGENT_COPY.planReady,
    );
    expect(showPlan).toHaveTextContent(CREATE_WITH_AGENT_COPY.plan);
    expect(showPlan).not.toHaveTextContent(CREATE_WITH_AGENT_COPY.showPlan);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    expect(within(strip).getByTestId("split-run-review")).toHaveTextContent("Ready");
    expect(screen.queryByTestId("split-run-intent-confidence-chip")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision")).not.toBeInTheDocument();

    await user.click(showPlan);
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      clarityExpanded: true,
      planOpen: true,
    });
    expect(screen.getByTestId("split-run-intent-result")).toBeInTheDocument();
    for (const column of screen.getAllByTestId("split-run-intent-chat-column")) {
      expect(column).toHaveClass("px-4");
    }
    expect(screen.getByTestId("split-run-intent-request").style.getPropertyValue("--intent-left")).toBe("50%");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(false);
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    const hidePlan = within(screen.getByTestId("split-run-intent-composer-chips")).getByRole("button", {
      name: CREATE_WITH_AGENT_COPY.plan,
    });
    expect(hidePlan).toHaveAttribute("aria-expanded", "true");
    expect(hidePlan).toHaveAttribute("aria-pressed", "true");
    expect(hidePlan).toHaveTextContent(CREATE_WITH_AGENT_COPY.plan);
    expect(within(hidePlan).queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-intent-result")).queryByRole("button", {
        name: CREATE_WITH_AGENT_COPY.plan,
      }),
    ).toBeNull();
    expect(screen.queryByTestId("split-run-intent-decision")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-intent-plan-updated")).getByTestId("split-run-review"),
    ).toHaveTextContent("Ready");

    await user.click(hidePlan);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(
      within(screen.getByTestId("split-run-intent-composer-chips")).getByRole("button", {
        name: CREATE_WITH_AGENT_COPY.plan,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(false);
  });

  it("hides the plan toggle until a spec exists", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).queryByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).not.toBeInTheDocument();
    expect(within(chips).getByTestId("split-run-intent-plan-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-composer-score")).not.toBeInTheDocument();
  });

  it("stops the Clarity matrix when analysis ends without a score", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        analysis={analysisChat({
          view: {
            machineStatus: "failed",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
          },
        })}
      />,
    );

    expect(screen.getByTestId("split-run-intent-composer-score")).toHaveTextContent("–");
    expect(screen.queryByTestId("split-run-intent-plan-analyzing")).not.toBeInTheDocument();
  });

  it("shows Ready then Updated when the spec changes", () => {
    const { rerender } = renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
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

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).getByTestId("split-run-intent-plan-status")).toHaveTextContent(
      CREATE_WITH_AGENT_COPY.planReady,
    );

    rerender(
      <TooltipProvider>
        <WorkOrderIntentDocument
          {...INTENT_DOC}
          artifacts={[{ ...INTENT, data: { ...INTENT.data, body: `${INTENT.data.body}\n\nMore scope.\n` } }]}
          confidence={HIGH_CONFIDENCE}
          analysis={analysisChat({
            view: {
              machineStatus: "waiting",
              canvasId: "canvas-1",
              canvasRunId: "run-1",
              executionId: "exec-1",
              messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 4 }],
            },
          })}
        />
      </TooltipProvider>,
    );

    const updated = within(screen.getByTestId("split-run-intent-composer-chips")).getByTestId(
      "split-run-intent-plan-status",
    );
    expect(updated).toHaveTextContent(CREATE_WITH_AGENT_COPY.planStatusUpdated);
    expect(updated.className).toContain("text-info-foreground");
    expect(updated.className).toContain("dark:text-warning");
  });

  it("replaces Clarity and the plan badge with the matrix while the agent works", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{ ...HIGH_CONFIDENCE, score: 2 }}
        analysis={analysisChat({
          view: {
            machineStatus: "running",
            canvasId: "canvas-1",
            canvasRunId: "run-1",
            executionId: "exec-1",
            messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 2 }],
          },
        })}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(within(chips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.clarity })).toBeInTheDocument();
    expect(within(chips).getByTestId("split-run-intent-plan-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(within(chips).getByTestId("split-run-intent-plan-chip-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(within(chips).queryByText(CREATE_WITH_AGENT_COPY.planReady)).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer-score")).not.toBeInTheDocument();
  });

  it("shows the Clarity summary in the status card", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
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

    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(
      within(card).getByTestId("split-run-intent-composer-chips").closest("[data-slot=frame-panel-header]"),
    ).not.toBeNull();
    expect(within(card).getByTestId("split-run-intent-confidence-drawer")).toHaveAttribute("data-state", "open");
    expect(within(card).getByTestId("split-run-intent-confidence-drawer").className).toContain(
      "transition-[grid-template-rows]",
    );
    expect(
      within(card).getByTestId("split-run-intent-confidence-copy").closest("[data-slot=frame-panel]"),
    ).not.toBeNull();
    expect(within(card).getByTestId("split-run-intent-confidence-copy")).toHaveTextContent(
      "This issue is a good fit for an agent on this factory line.",
    );
    expect(within(card).getByTestId("split-run-intent-composer-score")).toHaveAccessibleName("Clarity 4/5");
    expect(within(card).getByTestId("split-run-intent-composer-score")).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("split-run-intent-composer-card")).not.toContainElement(
      screen.getByTestId("split-run-intent-confidence-copy"),
    );
  });

  it("toggles the Clarity summary from the Clarity button", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
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

    const clarity = screen.getByTestId("split-run-intent-composer-score");
    const drawer = screen.getByTestId("split-run-intent-confidence-drawer");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("split-run-intent-confidence-copy")).toBeInTheDocument();
    await user.click(clarity);
    expect(clarity).toHaveAttribute("aria-expanded", "false");
    expect(drawer).toHaveAttribute("data-state", "closed");
    expect(drawer).toHaveAttribute("aria-hidden", "true");
    await user.click(clarity);
    expect(clarity).toHaveAttribute("aria-expanded", "true");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(drawer).not.toHaveAttribute("aria-hidden");
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      clarityExpanded: true,
      planOpen: false,
    });
  });

  it("restores Clarity and plan pane from localStorage", () => {
    window.localStorage.setItem(
      REFINE_LAYOUT_STORAGE_KEY,
      JSON.stringify({ clarityExpanded: false, planOpen: true }),
    );
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={HIGH_CONFIDENCE}
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

    expect(screen.getByTestId("split-run-intent-confidence-drawer")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("split-run-intent-composer-score")).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("split-run-intent-plan-status")).toHaveTextContent(CREATE_WITH_AGENT_COPY.planReady);
  });

  it("does not open the plan pane from storage before a spec exists", () => {
    window.localStorage.setItem(
      REFINE_LAYOUT_STORAGE_KEY,
      JSON.stringify({ clarityExpanded: true, planOpen: true }),
    );
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        confidence={HIGH_CONFIDENCE}
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

    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(false);
  });
});
