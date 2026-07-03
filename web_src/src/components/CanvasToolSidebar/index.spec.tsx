import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CanvasToolSidebar } from ".";
import { CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT } from "./events";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const richMessageRenderSpy = vi.fn();

const { sendMutation, chatState, chatRefetch } = vi.hoisted(() => {
  const state = {
    status: "idle",
    refetchStatus: "idle",
  };

  return {
    sendMutation: {
      isPending: false,
      mutateAsync: vi.fn(),
    },
    chatState: state,
    chatRefetch: vi.fn(async () => ({ data: { id: "chat-1", status: state.refetchStatus } })),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: { spec: { nodes: [] } } }),
  useCanvasVersions: () => ({ data: [] }),
  useCanvasVersion: () => ({ data: null }),
  useInfiniteCanvasRuns: () => ({ data: { pages: [] } }),
}));

vi.mock("@/hooks/useAgentChats", () => ({
  useCanvasAgentChat: () => ({
    data: { id: "chat-1", status: chatState.status },
    isLoading: false,
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
  useSendAgentChatMessage: () => sendMutation,
  useInterruptAgentChat: () => ({ isPending: false, mutate: vi.fn() }),
  useDefineAgentOutcome: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/useAgentSessionWebsocket", () => ({
  useAgentSessionWebsocket: () => undefined,
}));

vi.mock("@/components/AgentSidebar/widgets/RichMessage", () => ({
  RichMessage: ({ content }: { content: string }) => {
    richMessageRenderSpy();
    return <div data-testid="rich-message">{content}</div>;
  },
}));

function makeToolSidebarState(overrides: Partial<CanvasToolSidebarState> = {}) {
  return {
    canvasId: "canvas-1",
    organizationId: "org-1",
    isEditing: false,
    readOnly: false,
    isToolSidebarOpen: true,
    showToolSidebarToggle: true,
    isAgentEnabled: true,
    agentUnavailable: false,
    markAgentUnavailable: vi.fn(),
    handleToolSidebarToggle: vi.fn(),
    openToolSidebar: vi.fn(),
    closeToolSidebar: vi.fn(),
    agentMode: "operator" as const,
    switchAgentMode: vi.fn(),
    ...overrides,
  };
}

describe("CanvasToolSidebar", () => {
  beforeEach(() => {
    richMessageRenderSpy.mockClear();
    chatState.status = "idle";
    chatState.refetchStatus = "idle";
    chatRefetch.mockClear();
    sendMutation.isPending = false;
    sendMutation.mutateAsync.mockReset();
    sendMutation.mutateAsync.mockResolvedValue(null);
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the agent panel when the sidebar is open", async () => {
    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    expect(await screen.findByPlaceholderText("Ask the agent…")).toBeInTheDocument();
  });

  it("allows retrying a failed session", async () => {
    const user = userEvent.setup();
    chatState.status = "failed";

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    expect(await screen.findByText("Message failed. Try again.")).toBeInTheDocument();
    await user.type(screen.getByTestId("agent-input"), "retry");
    await user.click(screen.getByTestId("agent-send-message-button"));

    expect(sendMutation.mutateAsync).toHaveBeenCalledWith({
      chatId: "chat-1",
      content: "retry",
      mode: "operator",
      images: [],
    });
  });

  it("does not render when managed agents are disabled", () => {
    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ isAgentEnabled: false })} />);

    expect(screen.queryByPlaceholderText("Ask the agent…")).not.toBeInTheDocument();
  });

  it("does not render while the sidebar is closed", () => {
    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ isToolSidebarOpen: false })} />);

    expect(screen.queryByPlaceholderText("Ask the agent…")).not.toBeInTheDocument();
  });

  it("opens the sidebar when the agent tab event is dispatched", () => {
    const openToolSidebar = vi.fn();

    render(
      <CanvasToolSidebar toolSidebarState={makeToolSidebarState({ isToolSidebarOpen: false, openToolSidebar })} />,
    );

    window.dispatchEvent(new CustomEvent(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, { detail: { tab: "agent" } }));

    expect(openToolSidebar).toHaveBeenCalledTimes(1);
  });

  it("does not re-render agent messages while typing in the composer", async () => {
    const user = userEvent.setup();

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    const messages = await screen.findAllByTestId("rich-message");
    expect(messages).toHaveLength(2);
    expect(messages[1]).toHaveTextContent("Hello from the agent");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(2);

    await user.type(screen.getByTestId("agent-input"), "typing...");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(2);
  });

  it("does not wipe newly typed draft while a send is in flight", async () => {
    const user = userEvent.setup();
    let resolveSend: (() => void) | null = null;
    sendMutation.mutateAsync.mockImplementation(
      async () =>
        await new Promise<void>((resolve) => {
          resolveSend = resolve;
        }),
    );

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    const input = await screen.findByTestId("agent-input");
    await user.type(input, "first");
    await user.click(screen.getByTestId("agent-send-message-button"));

    await user.type(input, "second");
    expect(input).toHaveValue("second");

    await act(async () => {
      resolveSend?.();
    });
    expect(input).toHaveValue("second");
  });

  it("clears stale streaming state when a durable chat refetch returns idle", async () => {
    vi.useFakeTimers();
    chatState.status = "streaming";
    chatState.refetchStatus = "streaming";

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    expect(screen.getByTestId("agent-thinking")).toBeInTheDocument();
    expect(screen.getByText("Agent is running...")).toBeInTheDocument();

    chatState.refetchStatus = "idle";
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
    });

    expect(screen.queryByTestId("agent-thinking")).not.toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-stop-button")).not.toBeInTheDocument();
  });

  it("keeps streaming state while durable chat refetches are still streaming", async () => {
    vi.useFakeTimers();
    chatState.status = "streaming";
    chatState.refetchStatus = "streaming";

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    expect(screen.getByTestId("agent-thinking")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
    });

    expect(screen.getByTestId("agent-thinking")).toBeInTheDocument();
    expect(screen.getByText("Agent is running...")).toBeInTheDocument();
    expect(screen.getByTestId("agent-stop-button")).toBeInTheDocument();
  });
});
