import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  analysisChat,
  HIGH_CONFIDENCE,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  renderIntentDocument,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
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
  messages: [{ id: "plan-1", kind: "plan" as const, role: "plan" as const, score: 3 }],
};

function reviewMetric(key: "clarity" | "complexity" | "verifiability", score: number, summary: string) {
  const name = key.charAt(0).toUpperCase() + key.slice(1);
  return {
    id: `m-${key}`,
    key,
    name,
    score,
    maxScore: 3,
    level: score >= 3 ? ("positive" as const) : ("caution" as const),
    summary,
  };
}

describe("WorkOrderIntentDocument review metrics", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
  });

  it("shows review checks for a 2/3 score until the user hides them", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{
          ...HIGH_CONFIDENCE,
          score: 2,
          maxScore: 3,
          summary: "The change crosses billing and the API.",
        }}
        reviewMetrics={[
          reviewMetric("clarity", 3, "The gap is specific and verifiable."),
          reviewMetric("complexity", 2, "The work crosses a few layers."),
          reviewMetric("verifiability", 3, "Existing tests cover the change."),
        ]}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(confidence).toHaveAccessibleName(
      "Confidence checks: Clarity clear; Complexity moderate; Verifiability provable. Hide checks",
    );
    expect(confidence).not.toHaveTextContent("2/3");
    const bars = within(confidence)
      .getByTestId("split-run-intent-composer-confidence-meter")
      .querySelectorAll(".sp-meter-bar");
    expect(bars).toHaveLength(3);
    expect(bars[0]).toHaveClass("bg-emerald-500");
    expect(bars[1]).toHaveClass("bg-amber-500");
    expect(bars[2]).toHaveClass("bg-emerald-500");
    const metrics = screen.getByTestId("split-run-intent-review-metrics");
    expect(metrics).toBeVisible();
    expect(screen.getByTestId("split-run-intent-status-card")).toContainElement(metrics);
    expect(screen.queryByTestId("split-run-intent-composer-confidence-copy")).not.toBeInTheDocument();

    expect(metrics).toHaveAccessibleName("Confidence checks");
    expect(metrics).toHaveTextContent("Complexity needs review.");
    expect(metrics).toHaveTextContent("Clarity");
    expect(metrics).toHaveTextContent("Complexity");
    expect(metrics).toHaveTextContent("Verifiability");
    expect(metrics).not.toHaveTextContent("Moderate");
    expect(within(metrics).getByRole("img", { name: "Moderate" })).toBeInTheDocument();
    expect(metrics).toHaveTextContent("The gap is specific and verifiable.");
    expect(metrics).toHaveTextContent("The work crosses a few layers.");
    expect(metrics).toHaveTextContent("Existing tests cover the change.");
    expect(screen.queryByTestId("split-run-intent-composer-confidence-copy")).not.toBeInTheDocument();
    expect(confidence).toHaveAttribute("aria-expanded", "true");

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-expanded", "false");
    expect(confidence).toHaveTextContent("Show checks");
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("split-run-intent-review-metrics")).toBeVisible();
  });

  it("keeps checks available while a question needs an answer", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{ ...HIGH_CONFIDENCE, score: 2, maxScore: 3 }}
        reviewMetrics={[
          reviewMetric("clarity", 2, "The task needs a choice."),
          reviewMetric("complexity", 3, "The work is limited."),
          reviewMetric("verifiability", 3, "The result can be checked."),
        ]}
        analysis={analysisChat({
          view: {
            ...WAITING_WITH_PLAN,
            messages: [
              ...WAITING_WITH_PLAN.messages,
              { id: "agent-1", kind: "text" as const, role: "agent" as const, text: "Choose companies." },
            ],
            survey: { id: "survey-1", questions: [{ prompt: "Which companies?", options: ["Northstar"] }] },
          },
        })}
      />,
    );

    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(confidence).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();
    expect(screen.getByText("Which companies?")).toBeVisible();

    await user.click(confidence);
    expect(screen.getByTestId("split-run-intent-review-metrics")).toBeVisible();
  });

  it("shows all review checks at 3/3", async () => {
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        confidence={{
          ...HIGH_CONFIDENCE,
          score: 3,
          maxScore: 3,
          summary: "The gap is specific and verifiable.",
        }}
        reviewMetrics={[
          reviewMetric("clarity", 3, "The gap is specific and verifiable."),
          reviewMetric("complexity", 3, "The change stays in one layer."),
          reviewMetric("verifiability", 3, "Existing tests cover the change."),
        ]}
        analysis={analysisChat({ view: WAITING_WITH_PLAN })}
      />,
    );

    const confidence = screen.getByTestId("split-run-intent-composer-confidence");
    expect(confidence).toHaveAccessibleName(
      "Confidence checks: Clarity clear; Complexity easy; Verifiability provable. Show checks",
    );
    expect(confidence).not.toHaveTextContent("3/3");

    expect(confidence).toHaveTextContent("Show checks");
    expect(confidence).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();

    await user.click(confidence);
    const metrics = screen.getByTestId("split-run-intent-review-metrics");
    expect(metrics).toHaveAccessibleName("Confidence checks");
    expect(metrics).not.toHaveTextContent("The lowest check sets the score.");
    expect(metrics).toHaveTextContent("Clarity");
    expect(metrics).toHaveTextContent("Complexity");
    expect(metrics).toHaveTextContent("Verifiability");
    expect(metrics).not.toHaveTextContent("Easy");
    expect(within(metrics).getByRole("img", { name: "Easy" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer-confidence-copy")).not.toBeInTheDocument();
  });
});
