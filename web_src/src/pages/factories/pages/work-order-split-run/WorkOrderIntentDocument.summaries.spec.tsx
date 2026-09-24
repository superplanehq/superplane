import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { DRAFT_READINESS_NOTES } from "../../lib/draftReadiness";
import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import { SplitRunReview } from "./SplitRunReview";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
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

const WAITING_WITH_PLAN = {
  machineStatus: "waiting" as const,
  canvasId: "canvas-1",
  canvasRunId: "run-1",
  executionId: "exec-1",
  messages: [{ id: "plan-1", kind: "plan" as const, role: "plan" as const, score: 4 }],
};

describe("WorkOrderIntentDocument score evidence", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
  });

  it("leads with the verdict and keeps both scores as evidence", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    const verdict = within(card).getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "ready");
    expect(verdict).toHaveTextContent(DRAFT_READINESS_NOTES.ready.headline);
    expect(verdict).not.toHaveTextContent(DRAFT_READINESS_NOTES.ready.text);
    const clarity = within(card).getByTestId("split-run-intent-composer-score");
    const confidence = within(card).getByTestId("split-run-intent-composer-confidence");
    expect(clarity).toHaveAccessibleName("Clarity 4/5");
    expect(clarity).toHaveAttribute("aria-expanded", "false");
    expect(confidence).toHaveAccessibleName("Confidence 4/5");
    expect(
      within(clarity).getByTestId("split-run-intent-composer-score-meter").querySelectorAll("[data-filled='true']"),
    ).toHaveLength(4);
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-summary-drawer")).not.toBeInTheDocument();
  });

  it("warns in the verdict when Clarity is low and explains what to do", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={{ ...HIGH_CLARITY, score: 2 }}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({
          view: {
            ...WAITING_WITH_PLAN,
            messages: [{ id: "plan-1", kind: "plan", role: "plan", score: 2 }],
          },
        })}
      />,
    );

    const verdict = screen.getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "blocked");
    expect(verdict).toHaveTextContent(DRAFT_READINESS_NOTES.unclear.headline);
    expect(verdict).toHaveTextContent(DRAFT_READINESS_NOTES.unclear.text);
  });

  it("cautions in the verdict when Confidence is low", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 2 }}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const verdict = screen.getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "caution");
    expect(verdict).toHaveTextContent(DRAFT_READINESS_NOTES.agentFit.headline);
    expect(verdict).toHaveTextContent(DRAFT_READINESS_NOTES.agentFit.text);
  });

  it("fills Start when the verdict is ready and orders the controls Plan, model, Start", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={HIGH_CONFIDENCE}
        resultFooter={<SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} compact />}
        analysis={analysisChat({
          view: WAITING_WITH_PLAN,
          modelSelect: <button type="button">Model: Auto</button>,
        })}
      />,
    );

    const card = screen.getByTestId("split-run-intent-status-card");
    const verdict = within(card).getByTestId("split-run-intent-verdict");
    expect(within(verdict).queryByRole("button")).not.toBeInTheDocument();
    const settings = within(card).getByTestId("split-run-intent-settings");
    const plan = within(settings).getByRole("button", { name: CREATE_WITH_AGENT_COPY.plan });
    const model = within(settings).getByRole("button", { name: "Model: Auto" });
    const start = within(settings).getByRole("button", { name: "Start" });
    expect(plan.compareDocumentPosition(model) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(model.compareDocumentPosition(start) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(start).toHaveClass("bg-primary");
    expect(within(card).getByTestId("split-run-draft-action-group")).not.toHaveClass("border");
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
  });

  it("quiets Start to an outline when the verdict warns, but keeps it enabled", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 2 }}
        resultFooter={<SplitRunReview footer={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer} compact />}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    expect(screen.getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "caution");
    const start = screen.getByRole("button", { name: "Start" });
    expect(start).not.toHaveClass("bg-primary");
    expect(start).toHaveClass("border");
    expect(start).toBeEnabled();
    expect(screen.queryByTestId("split-run-intent-settings")).toBeInTheDocument();
  });

  it("peeks a score summary on hover and pins it on click", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        confidence={{ ...HIGH_CONFIDENCE, score: 3, summary: "The change crosses billing and the API." }}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(confidence).toHaveAccessibleName("Confidence 3/5");

    await user.hover(confidence);
    expect(await screen.findByTestId("split-run-intent-composer-confidence-copy")).toHaveTextContent(
      "The change crosses billing and the API.",
    );
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(confidence).toHaveAttribute("aria-pressed", "false");

    await user.unhover(confidence);
    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "true");
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    await user.unhover(confidence);
    expect(screen.getByTestId("split-run-intent-composer-confidence-copy")).toBeInTheDocument();

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "false");

    await user.hover(screen.getByTestId("split-run-intent-composer-score"));
    expect(await screen.findByTestId("split-run-intent-composer-score-copy")).toHaveTextContent(HIGH_CLARITY.summary);
    expect(window.localStorage.getItem(REFINE_LAYOUT_STORAGE_KEY)).toBeNull();
  });

  it("shows a dash for a score the analysis has not published", () => {
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

    const clarity = screen.getByTestId("split-run-intent-composer-score");
    expect(clarity).toHaveTextContent("–");
    expect(clarity).not.toHaveAttribute("aria-expanded");
    expect(screen.getByTestId("split-run-intent-composer-confidence")).toHaveAccessibleName("Confidence 4/5");
    expect(screen.getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "ready");
  });

  it("restores the plan pane from localStorage and ignores legacy summary keys", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ openSummary: "clarity", planOpen: true }));
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        clarity={HIGH_CLARITY}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "open");
    expect(screen.queryByTestId("split-run-intent-plan-status")).not.toBeInTheDocument();
  });

  it("does not open the plan pane from storage before a spec exists", () => {
    window.localStorage.setItem(REFINE_LAYOUT_STORAGE_KEY, JSON.stringify({ planOpen: true }));
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        confidence={HIGH_CONFIDENCE}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.plan })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(false);
  });
});
