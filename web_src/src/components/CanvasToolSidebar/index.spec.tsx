import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CanvasToolSidebar } from ".";
import { CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT } from "./events";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const richMessageRenderSpy = vi.fn();

const { sendMutation, resetMutation, chatState, chatRefetch } = vi.hoisted(() => {
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
    sendMutation: {
      isPending: false,
      mutateAsync: vi.fn(),
    },
    resetMutation: {
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
  useSendAgentChatMessage: () => sendMutation,
  useResetCanvasAgentChat: () => resetMutation,
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
    agentMode: "operator" as const,
    switchAgentMode: vi.fn(),
    ...overrides,
  };
}

describe("CanvasToolSidebar", () => {
  beforeEach(() => {
    richMessageRenderSpy.mockClear();
    chatState.hasChat = true;
    chatState.isError = false;
    chatState.isFetching = false;
    chatState.isLoading = false;
    chatState.status = "idle";
    chatState.refetchStatus = "idle";
    chatState.error = null;
    chatRefetch.mockClear();
    sendMutation.isPending = false;
    sendMutation.mutateAsync.mockReset();
    sendMutation.mutateAsync.mockResolvedValue(null);
    resetMutation.isPending = false;
    resetMutation.mutateAsync.mockReset();
    resetMutation.mutateAsync.mockResolvedValue(null);
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the agent panel when the sidebar is open", async () => {
    const markAgentAvailable = vi.fn();

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ markAgentAvailable })} />);

    expect(await screen.findByPlaceholderText("Ask the agent…")).toBeInTheDocument();
    await waitFor(() => expect(markAgentAvailable).toHaveBeenCalledTimes(1));
  });

  it("keeps the agent visible while a cached chat error is refetching", async () => {
    const markAgentUnavailable = vi.fn();
    chatState.hasChat = false;
    chatState.isError = true;
    chatState.isFetching = true;

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ markAgentUnavailable })} />);

    expect(await screen.findByText(/Setting up/)).toBeInTheDocument();
    expect(markAgentUnavailable).not.toHaveBeenCalled();
  });

  it("marks the agent unavailable after a settled chat setup failure", async () => {
    const markAgentUnavailable = vi.fn();
    chatState.hasChat = false;
    chatState.isError = true;
    chatState.isFetching = false;
    chatState.error = { code: 14, message: "agents are not enabled on this installation" };

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ markAgentUnavailable })} />);

    expect(await screen.findByText("The SuperPlane agent isn't available on this instance.")).toBeInTheDocument();
    await waitFor(() => expect(markAgentUnavailable).toHaveBeenCalledTimes(1));
  });

  it.each([
    [{ response: { data: { code: 14, message: "agents are not enabled on this installation" } } }],
    [{ error: { code: 14, message: "agents are not enabled on this installation" } }],
  ])("marks the agent unavailable when the disabled status is nested", async (error) => {
    const markAgentUnavailable = vi.fn();
    chatState.hasChat = false;
    chatState.isError = true;
    chatState.isFetching = false;
    chatState.error = error;

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ markAgentUnavailable })} />);

    expect(await screen.findByText("The SuperPlane agent isn't available on this instance.")).toBeInTheDocument();
    await waitFor(() => expect(markAgentUnavailable).toHaveBeenCalledTimes(1));
  });

  it("keeps non-disabled chat setup failures retryable", async () => {
    const user = userEvent.setup();
    const markAgentUnavailable = vi.fn();
    chatState.hasChat = false;
    chatState.isError = true;
    chatState.isFetching = false;
    chatState.error = { code: 7, message: "agent chat is not allowed" };

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState({ markAgentUnavailable })} />);

    expect(
      await screen.findByText("I couldn't set up the SuperPlane agent. Try again in a moment."),
    ).toBeInTheDocument();
    expect(markAgentUnavailable).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(chatRefetch).toHaveBeenCalledTimes(1);
  });

  it("clears the chat when the header reset button is clicked", async () => {
    const user = userEvent.setup();

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    await user.click(await screen.findByTestId("agent-clear-chat-button"));

    expect(resetMutation.mutateAsync).toHaveBeenCalledTimes(1);
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
      autoLayoutOnUpdateEnabled: false,
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
