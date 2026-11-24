import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { TooltipProvider } from "@/ui/tooltip";
import { AutomationRunChat } from "./AutomationRunChat";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "../../lib/customAutomation";

vi.mock("@/hooks/useCanvasData", () => ({
  useDescribeRun: () => ({ data: undefined }),
  useEventExecutions: () => ({ data: undefined }),
  useCanvas: () => ({
    data: { spec: { nodes: [{ metadata: { factoryTemplate: { id: CUSTOM_AUTOMATION_TEMPLATE_ID } } }] } },
  }),
}));
vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: async () => ({
    id: "chat-session",
    state: "ended",
    messages: [{ id: "reply", role: "agent", text: "What outcome do you want?" }],
  }),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

it("shows the conversation and composer without opening another dialog", async () => {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TooltipProvider>
        <AutomationRunChat
          organizationId="org"
          factoryId="factory"
          workOrderId="task"
          canUpdate
          phase={{
            id: "custom",
            name: "Chat",
            componentName: "Chat",
            status: "passed",
            duration: "1m",
            artifacts: [],
            stream: [],
            canvasSteps: [],
            appId: "app",
            runId: "run",
          }}
        />
      </TooltipProvider>
    </QueryClientProvider>,
  );
  expect(await screen.findByRole("textbox", { name: "Message the agent" })).toBeEnabled();
  expect(screen.getByText("What outcome do you want?")).toBeVisible();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
