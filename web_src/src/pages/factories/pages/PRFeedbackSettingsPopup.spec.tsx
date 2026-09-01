import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

const automationGraph = prFeedbackGraph();

function prFeedbackGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-pr-feedback", name: "Address PR feedback", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" },
          { id: "find", name: "Find Pull Request", type: "TYPE_ACTION", component: "findPullRequest" },
          {
            id: "runner",
            name: "Address PR feedback",
            type: "TYPE_ACTION",
            component: "runnerClaude",
          },
        ],
        edges: [
          { channel: "default", sourceId: "comment", targetId: "find" },
          { channel: "found", sourceId: "find", targetId: "runner" },
        ],
      },
    },
    [{ name: "github.onPRComment", label: "On PR Comment" }],
    [
      {
        name: "findPullRequest",
        label: "Find Pull Request",
        outputChannels: [{ name: "found" }, { name: "notFound" }],
      },
      { name: "runnerClaude", label: "Run Claude Code", outputChannels: [{ name: "passed" }, { name: "failed" }] },
    ],
    {},
    {},
    {},
    "app-pr-feedback",
    new QueryClient(),
    null,
    "live",
  );

  return { nodes, edges, factoryId: "factory-1" };
}

function renderPopup() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PRFeedbackSettingsPopup
              settings={{
                name: "Address PR feedback",
                repository: "acme/payments",
                mention: "@superplaneagent",
                ignoreBots: true,
                allowedBots: [],
              }}
              healthy
              automationGraph={automationGraph}
              onSave={vi.fn()}
              onClose={vi.fn()}
              editAutomationHref="/org-1/workspaces/RF/apps/app-pr-feedback?configure=1&agent=1"
              initialTab="automation"
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PRFeedbackSettingsPopup", () => {
  it("shows the automation in display mode at native zoom", () => {
    renderPopup();

    const automation = screen.getByTestId("pr-feedback-automation");
    expect(automation).toHaveAccessibleName("Automation");
    expect(within(automation).getAllByText("Find Pull Request").length).toBeGreaterThan(0);
    expect(within(automation).getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-pr-feedback?configure=1&agent=1",
    );
    expect(document.querySelector(".sp-canvas-editing")).toBeNull();
    expect(within(automation).queryByRole("button", { name: /Add next component/ })).not.toBeInTheDocument();
    expect(within(automation).queryByText("passed")).not.toBeInTheDocument();
    expect(within(automation).queryByText("failed")).not.toBeInTheDocument();
    expect(within(automation).queryByText("notFound")).not.toBeInTheDocument();
  });
});
