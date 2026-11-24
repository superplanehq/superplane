import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { TooltipProvider } from "@/ui/tooltip";
import { AutomationRunChat } from "./AutomationRunChat";
import { findPlanningSessionByWorkOrder, answerPlanningSessionSurvey } from "../planningSessionClient";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "../../lib/customAutomation";

vi.mock("@/hooks/useCanvasData", () => ({
  useDescribeRun: () => ({ data: undefined }),
  useEventExecutions: () => ({ data: undefined }),
  useCanvas: () => ({
    data: { spec: { nodes: [{ metadata: { factoryTemplate: { id: CUSTOM_AUTOMATION_TEMPLATE_ID } } }] } },
  }),
}));
vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(async () => ({
    id: "chat-session",
    state: "ended",
    messages: [{ id: "reply", role: "agent", text: "What outcome do you want?" }],
  })),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

function renderChat() {
  return render(
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
}

it("shows the conversation and composer without opening another dialog", async () => {
  renderChat();
  expect(await screen.findByRole("textbox", { name: "Message the agent" })).toBeEnabled();
  expect(screen.getByText("What outcome do you want?")).toBeVisible();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

it("submits survey answers from a custom automation conversation", async () => {
  vi.mocked(findPlanningSessionByWorkOrder).mockResolvedValueOnce({
    id: "chat-session",
    state: "ended",
    messages: [{ id: "question", role: "agent", text: "Choose an approach." }],
    survey: {
      id: "survey",
      questions: [
        { prompt: "Which approach?", options: ["Small patch", "Refactor"], recommendedOption: "Small patch" },
      ],
    },
  });
  vi.mocked(answerPlanningSessionSurvey).mockResolvedValueOnce({ id: "chat-session", state: "running" });
  renderChat();
  fireEvent.click(await screen.findByRole("button", { name: /Small patch/ }));
  await waitFor(() =>
    expect(answerPlanningSessionSurvey).toHaveBeenCalledWith(
      "org",
      "factory",
      "chat-session",
      expect.stringContaining("Small patch"),
    ),
  );
});
