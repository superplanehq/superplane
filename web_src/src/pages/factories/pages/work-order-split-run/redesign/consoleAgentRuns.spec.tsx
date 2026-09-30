import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import type { AutomationStage } from "./automationsViewModel";
import { AgentRunsPage } from "./consoleAgentRuns";

const LONG_COMMENT = [
  "## Embedded files block duplication",
  "When a task description contains an embedded file reference, this request sends the original file ID unchanged.",
  "Task creation then tries to bind that file to the new task.",
  "The server rejects it because it belongs to the original task.",
  "The description file references need handling even if attachments are not meant to be copied.",
  "## Duplicate ignores create permission",
  "A user who can read tasks but cannot create them still sees an active Duplicate button.",
].join("\n\n");

function commentRun(description: string): AutomationStage {
  return {
    id: "address-1",
    name: "@greptile-apps[bot] left a review",
    componentName: "Address PR feedback",
    status: "passed",
    statusLabel: "Passed",
    duration: "31m 9s",
    description,
    checks: [],
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: [],
    steps: [],
    rawLog: "",
  };
}

function renderRuns(runs: AutomationStage[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AgentRunsPage runs={runs} phases={[]} automationName="Address PR feedback" />
    </QueryClientProvider>,
  );
}

function stubElementHeights({ scrollHeight, clientHeight }: { scrollHeight: number; clientHeight: number }) {
  Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, value: scrollHeight });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, value: clientHeight });
  return () => {
    delete (HTMLElement.prototype as unknown as { scrollHeight?: number }).scrollHeight;
    delete (HTMLElement.prototype as unknown as { clientHeight?: number }).clientHeight;
  };
}

describe("AgentRunsPage run comments", () => {
  it("clips a long comment to five lines and expands it on Show more", async () => {
    const restoreHeights = stubElementHeights({ scrollHeight: 240, clientHeight: 100 });
    try {
      const user = userEvent.setup();
      renderRuns([commentRun(LONG_COMMENT)]);

      const body = screen.getByTestId("redesign-run-description-body");
      expect(body).toHaveClass("line-clamp-5");
      expect(screen.getByRole("button", { name: "Show more" })).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Show more" }));

      expect(body).not.toHaveClass("line-clamp-5");
      expect(screen.getByRole("button", { name: "Show less" })).toBeInTheDocument();
    } finally {
      restoreHeights();
    }
  });

  it("does not add Show more when a short comment fits", () => {
    const restoreHeights = stubElementHeights({ scrollHeight: 40, clientHeight: 40 });
    try {
      renderRuns([commentRun("Checks passed on the pull request.")]);

      expect(screen.queryByRole("button", { name: "Show more" })).not.toBeInTheDocument();
    } finally {
      restoreHeights();
    }
  });
});
