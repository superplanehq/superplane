import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CanvasToolSidebar } from ".";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

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

function makeToolSidebarState(overrides: Partial<CanvasToolSidebarState> = {}) {
  return {
    canvasId: "canvas-1",
    organizationId: "org-1",
    isEditing: false,
    readOnly: false,
    isToolSidebarOpen: true,
    showToolSidebarToggle: true,
    isAgentEnabled: true,
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

  it("hides the agent tab when managed agents are disabled", () => {
    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState({ isAgentEnabled: false })}
        versionsContent={<div>Versions content</div>}
      />,
    );

    expect(screen.queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Runs" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Versions" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Versions content")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Ask the agent…")).not.toBeInTheDocument();
  });

  it("opens version control on first open when managed agents are disabled", async () => {
    const onOpenVersionControl = vi.fn();
    const onVersionControlAutoOpened = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState({ isAgentEnabled: false })}
        mode="version-live"
        onOpenVersionControl={onOpenVersionControl}
        onVersionControlAutoOpened={onVersionControlAutoOpened}
      />,
    );

    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "true");
    await waitFor(() => expect(onOpenVersionControl).toHaveBeenCalledTimes(1));
    expect(onVersionControlAutoOpened).toHaveBeenCalledTimes(1);
  });

  it("does not auto-open version control after it already auto-opened for the canvas", () => {
    const onOpenVersionControl = vi.fn();
    const onVersionControlAutoOpened = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState({ isAgentEnabled: false })}
        mode="version-live"
        hasAutoOpenedVersionControl={true}
        onOpenVersionControl={onOpenVersionControl}
        onVersionControlAutoOpened={onVersionControlAutoOpened}
      />,
    );

    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "true");
    expect(onOpenVersionControl).not.toHaveBeenCalled();
    expect(onVersionControlAutoOpened).not.toHaveBeenCalled();
  });

  it("does not switch from runs mode to the agent tab when managed agents are disabled", () => {
    const onExitRunsMode = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState({ isAgentEnabled: false })}
        mode="runs"
        onExitRunsMode={onExitRunsMode}
        runsContent={<div>Runs content</div>}
      />,
    );

    expect(screen.queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    expect(screen.getByText("Runs content")).toBeInTheDocument();
  });

  it("keeps versions selected when version control closes and managed agents are disabled", () => {
    const toolSidebarState = makeToolSidebarState({ isAgentEnabled: false });
    const { rerender } = render(
      <CanvasToolSidebar
        toolSidebarState={toolSidebarState}
        isVersionControlOpen={true}
        runsContent={<div>Runs content</div>}
        versionsContent={<div>Versions content</div>}
      />,
    );

    expect(screen.queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Versions content")).toBeInTheDocument();

    rerender(
      <CanvasToolSidebar
        toolSidebarState={toolSidebarState}
        isVersionControlOpen={false}
        runsContent={<div>Runs content</div>}
        versionsContent={<div>Versions content</div>}
      />,
    );

    expect(screen.getByRole("tab", { name: "Versions" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Versions content")).toBeInTheDocument();
  });

  it("enters versions from the versions tab", () => {
    const onOpenVersionControl = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="version-live"
        isVersionControlOpen={false}
        onOpenVersionControl={onOpenVersionControl}
        versionsContent={<div>Versions content</div>}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Versions" }));

    expect(onOpenVersionControl).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Versions content")).toBeInTheDocument();
  });

  it("exits versions when switching back to the agent tab", () => {
    const onCloseVersionControl = vi.fn();

    render(
      <CanvasToolSidebar
        toolSidebarState={makeToolSidebarState()}
        mode="version-live"
        isVersionControlOpen={true}
        onCloseVersionControl={onCloseVersionControl}
        versionsContent={<div>Versions content</div>}
      />,
    );

    expect(screen.getByText("Versions content")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Agent" }));

    expect(onCloseVersionControl).toHaveBeenCalledTimes(1);
    expect(screen.getByPlaceholderText("Ask the agent…")).toBeInTheDocument();
  });

  it("does not re-render agent messages while typing in the composer", async () => {
    const user = userEvent.setup();

    render(<CanvasToolSidebar toolSidebarState={makeToolSidebarState()} />);

    // Initial message render.
    const messages = await screen.findAllByTestId("rich-message");
    expect(messages).toHaveLength(2);
    expect(messages[1]).toHaveTextContent("Hello from the agent");
    expect(richMessageRenderSpy).toHaveBeenCalledTimes(2);

    // Typing only updates local composer state, so the message list should not re-render.
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

    // While the first send is still pending, user can type a new message.
    await user.type(input, "second");
    expect(input).toHaveValue("second");

    // Resolve the first send; the new draft should remain.
    await act(async () => {
      resolveSend?.();
    });
    expect(input).toHaveValue("second");
  });

  it("allows sending while an outcome is still active after the chat turn ends", async () => {
    const user = userEvent.setup();
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

    await user.type(screen.getByTestId("agent-input"), "Keep going");
    expect(screen.getByTestId("agent-send-message-button")).toBeEnabled();
  });
});
