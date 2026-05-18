import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgentSidebar } from "./index";
import type { AgentMessage, AgentChat } from "./types";

const richMessageRenderSpy = vi.fn();

vi.mock("./widgets/RichMessage", () => ({
  RichMessage: ({ content }: { content: string }) => {
    richMessageRenderSpy();
    return <div data-testid="rich-message">{content}</div>;
  },
}));

vi.mock("@/hooks/useAgentSessionWebsocket", () => ({
  useAgentSessionWebsocket: () => undefined,
}));

type MockInfiniteData<TPage> = { pages: TPage[]; pageParams: unknown[] };
type AgentMessagesPage = { messages: AgentMessage[]; hasMore: boolean };

const mockChat: AgentChat = {
  id: "chat-1",
  canvasId: "canvas-1",
  provider: "test",
  status: "idle",
  createdAt: null,
  updatedAt: null,
};

const assistantMessage: AgentMessage = {
  id: "m-1",
  role: "assistant",
  content: "Hello from the agent",
  toolName: "",
  toolCallId: "",
  toolStatus: "",
  createdAt: null,
};

const messagesData: MockInfiniteData<AgentMessagesPage> = {
  pages: [{ messages: [assistantMessage], hasMore: false }],
  pageParams: [""],
};

const sendMutation = {
  isPending: false,
  mutateAsync: vi.fn(async () => undefined),
};

vi.mock("@/hooks/useAgentChats", () => ({
  useCanvasAgentChat: () => ({ isLoading: false, data: mockChat }),
  useAgentChatMessages: () => ({
    isLoading: false,
    data: messagesData,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(async () => undefined),
  }),
  useSendAgentChatMessage: () => sendMutation,
}));

describe("AgentSidebar", () => {
  beforeEach(() => {
    richMessageRenderSpy.mockClear();
  });

  it("does not re-render the message list while typing in the composer", async () => {
    const user = userEvent.setup();

    render(
      <AgentSidebar
        agentState={{
          canvasId: "canvas-1",
          organizationId: "org-1",
          isEditing: false,
          readOnly: false,
          isAgentSidebarOpen: true,
          showAgentSidebarToggle: true,
          handleAgentSidebarToggle: vi.fn(),
          closeSidebar: vi.fn(),
        }}
      />,
    );

    // Initial message render.
    expect(await screen.findByTestId("rich-message")).toHaveTextContent("Hello from the agent");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(1);

    // Typing only updates the local composer state, so the message list should
    // not re-render.
    await user.type(screen.getByTestId("agent-input"), "typing...");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(1);
  });
});

