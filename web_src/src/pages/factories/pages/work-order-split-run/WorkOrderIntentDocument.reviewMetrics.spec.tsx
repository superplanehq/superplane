import { screen, waitFor, within } from "@testing-library/react";
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

  it("reveals review checks on hover and pins them on click", async () => {
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
    expect(confidence).toHaveAccessibleName("Confidence 2/3");
    expect(
      within(confidence).getByTestId("split-run-intent-composer-confidence-meter").querySelectorAll(".sp-meter-bar"),
    ).toHaveLength(3);
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer-confidence-copy")).not.toBeInTheDocument();

    await user.hover(confidence);
    const metrics = await screen.findByTestId("split-run-intent-review-metrics");
    expect(metrics).toBeVisible();
    expect(metrics).toHaveTextContent("Clarity");
    expect(metrics).toHaveTextContent("Clear");
    expect(metrics).toHaveTextContent("Complexity");
    expect(metrics).toHaveTextContent("Moderate");
    expect(metrics).toHaveTextContent("Verifiability");
    expect(metrics).toHaveTextContent("Provable");
    expect(metrics).toHaveTextContent("The gap is specific and verifiable.");
    expect(metrics).toHaveTextContent("The work crosses a few layers.");
    expect(metrics).toHaveTextContent("Existing tests cover the change.");
    expect(metrics).not.toHaveTextContent("/3");
    expect(screen.queryByTestId("split-run-intent-composer-confidence-copy")).not.toBeInTheDocument();
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(confidence).toHaveAttribute("aria-pressed", "false");

    await user.unhover(confidence);
    await waitFor(() => {
      expect(confidence).toHaveAttribute("aria-expanded", "false");
    });
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "true");
    expect(confidence).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("split-run-intent-review-metrics")).toBeVisible();
    await user.unhover(confidence);
    expect(screen.getByTestId("split-run-intent-review-metrics")).toBeVisible();

    await user.click(confidence);
    expect(confidence).toHaveAttribute("aria-pressed", "false");
  });

  it("shows the summary card instead of the review drawer at 3/3", async () => {
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
    expect(confidence).toHaveAccessibleName("Confidence 3/3");

    await user.hover(confidence);
    expect(await screen.findByTestId("split-run-intent-composer-confidence-copy")).toHaveTextContent(
      "The gap is specific and verifiable.",
    );
    expect(screen.queryByTestId("split-run-intent-review-metrics")).not.toBeInTheDocument();
  });
});
