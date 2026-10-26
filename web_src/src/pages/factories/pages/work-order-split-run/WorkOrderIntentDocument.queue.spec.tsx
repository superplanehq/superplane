import { screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { createWithAgentViewFromSession } from "../planningSessionView";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import {
  analysisChat,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  renderIntentDocument,
} from "./WorkOrderIntentDocument.testHelpers";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({ resolveUser: () => null, isLoading: false }),
}));

describe("queued task planning", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it.each([
    ["parallelism_limit", "Parallelism limit reached."],
    ["", "Waiting for a runner."],
  ])("shows queued planning without a working indicator (%s)", (queueReason, reason) => {
    const session = { state: "running", executionId: "execution-1", queueReason };
    const document = (runnerState: string) => (
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[]}
        isAnalyzing
        analysis={analysisChat({
          isAnalyzing: true,
          view: createWithAgentViewFromSession(
            { ...session, runnerState },
            { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
          ),
        })}
      />
    );
    const rendered = renderIntentDocument(document("queued"));

    const queue = screen.getByRole("status", { name: "Queued for planning" });
    expect(queue).toHaveTextContent(reason);
    expect(queue).toHaveTextContent("Planning starts automatically when a slot is free.");
    expect(screen.queryByTestId("split-run-intent-thinking")).not.toBeInTheDocument();

    rendered.rerender(<TooltipProvider>{document("running")}</TooltipProvider>);

    expect(screen.queryByRole("status", { name: "Queued for planning" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting analysis…");
  });
});
