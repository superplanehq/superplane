import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import {
  analysisChat,
  HIGH_CLARITY,
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

describe("WorkOrderIntentDocument score summaries", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
  });

  it("shows the Clarity summary in the status card", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
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
    const drawer = within(card).getByTestId("split-run-intent-summary-drawer");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(drawer).toHaveAttribute("data-kind", "clarity");
    expect(drawer.className).toContain("transition-[grid-template-rows]");
    expect(within(card).getByTestId("split-run-intent-summary-copy").closest("[data-slot=frame-panel]")).not.toBeNull();
    expect(within(card).getByTestId("split-run-intent-summary-copy")).toHaveTextContent(HIGH_CLARITY.summary);
    expect(within(card).getByTestId("split-run-intent-composer-score")).toHaveAccessibleName("Clarity 4/5");
    expect(within(card).getByTestId("split-run-intent-composer-score")).toHaveAttribute("aria-expanded", "true");
    expect(within(card).getByTestId("split-run-intent-composer-confidence")).toHaveAccessibleName("Confidence 4/5");
    expect(within(card).getByTestId("split-run-intent-composer-confidence")).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByTestId("split-run-intent-composer-card")).not.toContainElement(
      screen.getByTestId("split-run-intent-summary-copy"),
    );
  });

  it("switches the shared drawer between Clarity and Confidence", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 3, summary: "The change crosses billing and the API." }}
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
    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    const drawer = screen.getByTestId("split-run-intent-summary-drawer");
    expect(confidence).toHaveAccessibleName("Confidence 3/5");
    expect(drawer).toHaveAttribute("data-kind", "clarity");

    await user.click(confidence);
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(drawer).toHaveAttribute("data-kind", "confidence");
    expect(screen.getByTestId("split-run-intent-summary-copy")).toHaveTextContent(
      "The change crosses billing and the API.",
    );
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(clarity).toHaveAttribute("aria-expanded", "false");
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      openSummary: "confidence",
      planOpen: false,
    });

    await user.click(clarity);
    expect(drawer).toHaveAttribute("data-kind", "clarity");
    expect(screen.getByTestId("split-run-intent-summary-copy")).toHaveTextContent(HIGH_CLARITY.summary);
  });

  it("falls back to the Confidence summary when Clarity has none", async () => {
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
          },
        })}
      />,
    );

    const drawer = screen.getByTestId("split-run-intent-summary-drawer");
    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(drawer).toHaveAttribute("data-kind", "confidence");
    expect(screen.getByTestId("split-run-intent-composer-score")).not.toHaveAttribute("aria-expanded");
    expect(confidence).toHaveAttribute("aria-expanded", "true");

    await user.click(confidence);
    expect(drawer).toHaveAttribute("data-state", "closed");
    expect(confidence).toHaveAttribute("aria-expanded", "false");
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      openSummary: null,
      planOpen: false,
    });

    await user.click(confidence);
    expect(drawer).toHaveAttribute("data-kind", "confidence");
    expect(drawer).toHaveAttribute("data-state", "open");
  });

  it("toggles the Clarity summary from the Clarity button", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
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
    const drawer = screen.getByTestId("split-run-intent-summary-drawer");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("split-run-intent-summary-copy")).toBeInTheDocument();
    await user.click(clarity);
    expect(clarity).toHaveAttribute("aria-expanded", "false");
    expect(drawer).toHaveAttribute("data-state", "closed");
    expect(drawer).toHaveAttribute("aria-hidden", "true");
    await user.click(clarity);
    expect(clarity).toHaveAttribute("aria-expanded", "true");
    expect(drawer).toHaveAttribute("data-state", "open");
    expect(drawer).not.toHaveAttribute("aria-hidden");
    expect(JSON.parse(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY) || "{}")).toEqual({
      openSummary: "clarity",
      planOpen: false,
    });
  });

  it("restores Clarity and plan pane from localStorage", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: null, planOpen: true }));
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
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

    expect(screen.getByTestId("split-run-intent-summary-drawer")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("split-run-intent-composer-score")).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("split-run-intent-plan-status")).toHaveTextContent(CREATE_WITH_AGENT_COPY.planReady);
  });

  it("does not open the plan pane from storage before a spec exists", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: "clarity", planOpen: true }));
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
