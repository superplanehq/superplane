import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CanvasToolSidebar } from ".";

const richMessageRenderSpy = vi.fn();

const sendMutation = {
  isPending: false,
  mutateAsync: vi.fn(),
};

vi.mock("@/hooks/useAgentChats", () => ({
  useCanvasAgentChat: () => ({ data: { id: "chat-1" }, isLoading: false }),
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

function makeToolSidebarState() {
  return {
    canvasId: "canvas-1",
    organizationId: "org-1",
    isEditing: false,
    readOnly: false,
    isToolSidebarOpen: true,
    showToolSidebarToggle: true,
    handleToolSidebarToggle: vi.fn(),
    openToolSidebar: vi.fn(),
    closeToolSidebar: vi.fn(),
    agentMode: "operator" as const,
    switchAgentMode: vi.fn(),
  };
}

describe("CanvasToolSidebar", () => {
  beforeEach(() => {
    richMessageRenderSpy.mockClear();
    sessionStorage.clear();
  });

  it("enters runs mode from the runs tab", () => {
    const onSelectRuns = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="version-live"
        onSelectRuns={onSelectRuns}
        runsContent={<div>Runs content</div>}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Runs" }));

    expect(onSelectRuns).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Runs content")).toBeInTheDocument();
  });

  it("exits runs mode when switching back to the agent tab", () => {
    const onExitRunsMode = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="runs"
        onExitRunsMode={onExitRunsMode}
        runsContent={<div>Runs content</div>}
      />,
    );

    expect(screen.getByText("Runs content")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Agent" }));

    expect(onExitRunsMode).toHaveBeenCalledTimes(1);
    expect(screen.getByPlaceholderText("Ask the agent…")).toBeInTheDocument();
  });

  it("enters versions from the versions tab", () => {
    const onToggleVersionControl = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="version-live"
        isVersionControlOpen={false}
        onToggleVersionControl={onToggleVersionControl}
        versionsContent={<div>Versions content</div>}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Versions" }));

    expect(onToggleVersionControl).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Versions content")).toBeInTheDocument();
  });

  it("exits versions when switching back to the agent tab", () => {
    const onToggleVersionControl = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="version-live"
        isVersionControlOpen={true}
        onToggleVersionControl={onToggleVersionControl}
        versionsContent={<div>Versions content</div>}
      />,
    );

    expect(screen.getByText("Versions content")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Agent" }));

    expect(onToggleVersionControl).toHaveBeenCalledTimes(1);
    expect(screen.getByPlaceholderText("Ask the agent…")).toBeInTheDocument();
  });

  it("exits runs mode before closing the sidebar from the runs tab", () => {
    const toolSidebarState = makeToolSidebarState();
    const onExitRunsMode = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={toolSidebarState}
        mode="runs"
        onExitRunsMode={onExitRunsMode}
        runsContent={<div>Runs content</div>}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Close sidebar" }));

    expect(onExitRunsMode).toHaveBeenCalledTimes(1);
    expect(toolSidebarState.closeToolSidebar).toHaveBeenCalledTimes(1);
  });

  it("exits versions before closing the sidebar from the versions tab", () => {
    const toolSidebarState = makeToolSidebarState();
    const onToggleVersionControl = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={toolSidebarState}
        mode="version-live"
        isVersionControlOpen={true}
        onToggleVersionControl={onToggleVersionControl}
        versionsContent={<div>Versions content</div>}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Close sidebar" }));

    expect(onToggleVersionControl).toHaveBeenCalledTimes(1);
    expect(toolSidebarState.closeToolSidebar).toHaveBeenCalledTimes(1);
  });

  it("does not re-render agent messages while typing in the composer", async () => {
    const user = userEvent.setup();

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    // Initial message render.
    expect(await screen.findByTestId("rich-message")).toHaveTextContent("Hello from the agent");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(1);

    // Typing only updates local composer state, so the message list should not re-render.
    await user.type(screen.getByTestId("agent-input"), "typing...");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(1);
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
    await user.click(screen.getByRole("button", { name: "Send" }));

    // While the first send is still pending, user can type a new message.
    await user.type(input, "second");
    expect(input).toHaveValue("second");

    // Resolve the first send; the new draft should remain.
    await act(async () => {
      resolveSend?.();
    });
    expect(input).toHaveValue("second");
  });

  it("shows Stop while an outcome is still active after the chat turn ends", () => {
    sessionStorage.setItem(
      "outcome-chat-1",
      JSON.stringify({
        title: "Build plan",
        criteria: [],
        iteration: 1,
        maxIterations: 3,
        phase: "building",
        log: [{ phase: "building" }],
      }),
    );

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    expect(screen.getByTestId("agent-stop-button")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-send-message-button")).not.toBeInTheDocument();
  });
});
