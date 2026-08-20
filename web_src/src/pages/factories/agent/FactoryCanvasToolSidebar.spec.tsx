import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { FactoryCanvasToolSidebar } from "./FactoryCanvasToolSidebar";

const { chatState, chatRefetch } = vi.hoisted(() => {
  const state = {
    hasChat: true,
    isError: false,
    isFetching: false,
    isLoading: false,
    status: "idle",
    refetchStatus: "idle",
    error: null as unknown,
  };

  return {
    chatState: state,
    chatRefetch: vi.fn(async () => ({ data: { id: "chat-1", status: state.refetchStatus } })),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: { spec: { nodes: [] } } }),
  useDescribeCanvasVersion: () => ({ data: undefined }),
  useCanvasVersion: () => ({ data: null }),
  useInfiniteCanvasRuns: () => ({ data: { pages: [] } }),
}));

vi.mock("@/hooks/useAgentChats", () => ({
  useCanvasAgentChat: () => ({
    data: chatState.hasChat ? { id: "chat-1", status: chatState.status } : undefined,
    error: chatState.error,
    isError: chatState.isError,
    isFetching: chatState.isFetching,
    isLoading: chatState.isLoading,
    refetch: chatRefetch,
  }),
  useAgentChatMessages: () => ({
    data: {
      pages: [
        {
          hasMore: false,
          messages: [
            {
              id: "m-1",
              role: "assistant",
              content: "Hello from the agent",
              toolName: "",
              toolCallId: "",
              toolStatus: "",
              createdAt: null,
            },
          ],
        },
      ],
      pageParams: [""],
    },
    isLoading: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(async () => undefined),
  }),
  useSendAgentChatMessage: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useResetCanvasAgentChat: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useInterruptAgentChat: () => ({ isPending: false, mutate: vi.fn() }),
  useDefineAgentOutcome: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/useAgentSessionWebsocket", () => ({
  useAgentSessionWebsocket: () => undefined,
}));

vi.mock("./FactoryRichMessage", () => ({
  FactoryRichMessage: ({ content }: { content: string }) => <div data-testid="rich-message">{content}</div>,
}));

function makeToolSidebarState(overrides: Partial<CanvasToolSidebarState> = {}): CanvasToolSidebarState {
  return {
    canvasId: "canvas-1",
    organizationId: "org-1",
    liveCanvasVersionId: undefined,
    headerMode: undefined,
    isRunInspectionMode: false,
    onAgentStagingReady: undefined,
    onAgentStagingCommit: undefined,
    isEditing: false,
    isAutoLayoutOnUpdateEnabled: false,
    readOnly: false,
    isToolSidebarOpen: true,
    showToolSidebarToggle: true,
    isAgentEnabled: true,
    agentUnavailable: false,
    markAgentUnavailable: vi.fn(),
    markAgentAvailable: vi.fn(),
    handleToolSidebarToggle: vi.fn(),
    openToolSidebar: vi.fn(),
    closeToolSidebar: vi.fn(),
    agentMode: "operator" as AgentMode,
    switchAgentMode: vi.fn(),
    ...overrides,
  };
}

describe("FactoryCanvasToolSidebar", () => {
  beforeEach(() => {
    chatState.hasChat = true;
    chatState.isError = false;
    chatState.isFetching = false;
    chatState.isLoading = false;
    chatState.status = "idle";
    sessionStorage.clear();
  });

  it("renders factory agent chrome with Inter typography", async () => {
    render(<FactoryCanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    const sidebar = await screen.findByTestId("canvas-tool-sidebar");
    expect(sidebar).toHaveAttribute("data-factory-agent", "true");
    expect(sidebar).toHaveClass("font-inter");
    expect(sidebar).toHaveClass("font-normal");
    expect(sidebar).toHaveClass("text-[13px]");
    expect(sidebar).toHaveClass("factory-agent-sidebar");
    const messages = await screen.findAllByTestId("agent-assistant-message");
    expect(messages.length).toBeGreaterThan(0);
    for (const message of messages) {
      expect(message).toHaveClass("factory-agent-message");
      expect(message).toHaveClass("text-[15px]");
      expect(message).toHaveClass("font-normal");
    }
    expect(await screen.findByPlaceholderText("Ask the agent…")).toBeInTheDocument();
  });

  it("does not render while the sidebar is closed", () => {
    render(<FactoryCanvasToolSidebar toolSidebarState={makeToolSidebarState({ isToolSidebarOpen: false })} />);

    expect(screen.queryByTestId("canvas-tool-sidebar")).not.toBeInTheDocument();
  });
});
