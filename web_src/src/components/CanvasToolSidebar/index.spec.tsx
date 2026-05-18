import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CanvasToolSidebar } from ".";

vi.mock("@/hooks/useAgentChats", () => ({
  useCanvasAgentChat: () => ({ data: { id: "chat-1" }, isLoading: false }),
  useAgentChatMessages: () => ({
    data: { pages: [] },
    isLoading: false,
    hasNextPage: false,
    isFetchingNextPage: false,
  }),
  useSendAgentChatMessage: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/useAgentSessionWebsocket", () => ({
  useAgentSessionWebsocket: () => undefined,
}));

vi.mock("@/components/AgentSidebar/widgets/RichMessage", () => ({
  RichMessage: ({ content }: { content: string }) => <div>{content}</div>,
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
  };
}

describe("CanvasToolSidebar", () => {
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
});
