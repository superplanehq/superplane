import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import {
  analysisChat,
  HIGH_CONFIDENCE,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  renderIntentDocument,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import { REFINE_LAYOUT_STORAGE_KEY } from "./refineLayoutPreference";
import { resetStreamMemoryForTests } from "./useStreamOnUpdate";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Ada Lovelace", initials: "AL" } : null,
    isLoading: false,
  }),
}));

describe("WorkOrderIntentDocument composer", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
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

  it("keeps draft actions on the chips before a score exists", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        resultFooter={<div data-testid="split-run-review">Ready</div>}
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

    const card = screen.getByTestId("split-run-intent-status-card");
    const chips = screen.getByTestId("split-run-intent-composer-chips");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(card).toContainElement(chips);
    expect(screen.queryByTestId("split-run-intent-confidence-copy")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence-drawer")).not.toBeInTheDocument();
    expect(within(chips).getByRole("button", { name: CREATE_WITH_AGENT_COPY.clarity })).toBeInTheDocument();
    expect(within(chips).getByTestId("split-run-review")).toHaveTextContent("Ready");
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
    const analyzing = within(chips).getByTestId("split-run-intent-plan-analyzing");
    expect(analyzing.querySelector(".t-matrix")).not.toBeNull();
    expect(analyzing).not.toHaveTextContent("Analyzing");
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

  it("explains the Clarity matrix on hover while the agent works", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
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

    await user.hover(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.clarity }));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(CREATE_WITH_AGENT_COPY.clarityAnalyzing);
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
    const cardMatrix = within(chips).getByTestId("split-run-intent-plan-analyzing");
    expect(cardMatrix.querySelector(".t-matrix")).not.toBeNull();
    expect(cardMatrix).not.toHaveTextContent("Analyzing");
    const planMatrix = within(chips).getByTestId("split-run-intent-plan-chip-analyzing");
    expect(planMatrix.querySelector(".t-matrix")).not.toBeNull();
    expect(planMatrix).not.toHaveTextContent("Analyzing");
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
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: false, planOpen: true }));
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
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ clarityExpanded: true, planOpen: true }));
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
