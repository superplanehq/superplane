import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { loadChatSessions, loadChatConversation, sendChatPrompt } = vi.hoisted(() => ({
  loadChatSessions: vi.fn(),
  loadChatConversation: vi.fn(),
  sendChatPrompt: vi.fn(),
}));

vi.mock("../CanvasPage", () => ({
  COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY: "sp:test-sidebar-width",
}));

vi.mock("./CategorySection", () => ({
  CategorySection: () => <div>category section</div>,
}));

vi.mock("@/components/ui/tabs", async () => {
  const React = await import("react");
  const TabsContext = React.createContext<{
    value: string;
    onValueChange?: (value: string) => void;
  }>({ value: "" });

  return {
    Tabs: ({
      value,
      onValueChange,
      children,
      className,
    }: {
      value: string;
      onValueChange?: (value: string) => void;
      children: ReactNode;
      className?: string;
    }) => (
      <TabsContext.Provider value={{ value, onValueChange }}>
        <div className={className}>{children}</div>
      </TabsContext.Provider>
    ),
    TabsList: ({ children, className }: { children: ReactNode; className?: string }) => (
      <div className={className}>{children}</div>
    ),
    TabsTrigger: ({ value, children, className }: { value: string; children: ReactNode; className?: string }) => {
      const context = React.useContext(TabsContext);

      return (
        <button
          type="button"
          role="tab"
          data-state={context.value === value ? "active" : "inactive"}
          className={className}
          onClick={() => context.onValueChange?.(value)}
        >
          {children}
        </button>
      );
    },
    TabsContent: ({ value, children, className }: { value: string; children: ReactNode; className?: string }) => {
      const context = React.useContext(TabsContext);

      if (context.value !== value) {
        return null;
      }

      return <div className={className}>{children}</div>;
    },
  };
});

vi.mock("./AiBuilderChatPanel", () => ({
  AiBuilderChatPanel: ({
    chatSessions,
    currentChatId,
    aiMessages,
    onSelectChat,
  }: {
    chatSessions: Array<{ id: string }>;
    currentChatId: string | null;
    aiMessages: Array<{ id: string }>;
    onSelectChat: (chatId: string) => void;
  }) => (
    <div>
      <div>ai builder panel</div>
      <div data-testid="chat-session-count">{chatSessions.length}</div>
      <div data-testid="current-chat-id">{currentChatId ?? "none"}</div>
      <div data-testid="ai-message-count">{aiMessages.length}</div>
      <button
        type="button"
        onClick={() => {
          if (chatSessions[0]?.id) {
            onSelectChat(chatSessions[0].id);
          }
        }}
      >
        select first chat
      </button>
    </div>
  ),
}));

vi.mock("../componentBase", () => ({
  ComponentBase: () => <div>component base</div>,
}));

vi.mock("./agentChat", () => ({
  loadChatSessions,
  loadChatConversation,
  sendChatPrompt,
  pushAiMessages: (previous: unknown[], next: unknown | unknown[]) => [
    ...previous,
    ...(Array.isArray(next) ? next : [next]),
  ],
}));

import { BuildingBlocksSidebar } from "./index";

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolver) => {
    resolve = resolver;
  });

  return { promise, resolve };
}

const defaultProps = {
  isOpen: true,
  onToggle: vi.fn(),
  blocks: [],
  showAiBuilderTab: true,
  canvasId: "canvas-1",
  organizationId: "org-1",
};

describe("BuildingBlocksSidebar", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    loadChatSessions.mockResolvedValue([]);
    loadChatConversation.mockResolvedValue([]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const flushEffects = async () => {
    await act(async () => {
      vi.runAllTimers();
      await Promise.resolve();
      await Promise.resolve();
    });
  };

  it("does not load chat sessions before the AI tab is opened", async () => {
    render(<BuildingBlocksSidebar {...defaultProps} />);

    await flushEffects();

    expect(screen.getByRole("tab", { name: "Components" })).toHaveAttribute("data-state", "active");
    expect(loadChatSessions).not.toHaveBeenCalled();
    expect(loadChatConversation).not.toHaveBeenCalled();
  });

  it("loads chat sessions when the AI tab is opened", async () => {
    render(<BuildingBlocksSidebar {...defaultProps} />);

    fireEvent.click(screen.getByRole("tab", { name: "AI Builder" }));
    await flushEffects();

    expect(loadChatSessions).toHaveBeenCalledWith({
      canvasId: "canvas-1",
      organizationId: "org-1",
    });
  });

  it("does not refetch AI data when reopening the AI tab", async () => {
    loadChatSessions.mockResolvedValue([{ id: "chat-1", title: "Chat 1" }]);
    loadChatConversation.mockResolvedValue([{ id: "message-1", role: "assistant", content: "hello" }]);

    render(<BuildingBlocksSidebar {...defaultProps} />);

    fireEvent.click(screen.getByRole("tab", { name: "AI Builder" }));
    await flushEffects();

    fireEvent.click(screen.getByRole("button", { name: "select first chat" }));
    await flushEffects();

    expect(screen.getByTestId("current-chat-id")).toHaveTextContent("chat-1");
    expect(screen.getByTestId("ai-message-count")).toHaveTextContent("1");

    fireEvent.click(screen.getByRole("tab", { name: "Components" }));
    await flushEffects();

    fireEvent.click(screen.getByRole("tab", { name: "AI Builder" }));
    await flushEffects();

    expect(screen.getByTestId("current-chat-id")).toHaveTextContent("chat-1");
    expect(screen.getByTestId("ai-message-count")).toHaveTextContent("1");
    expect(loadChatSessions).toHaveBeenCalledTimes(1);
    expect(loadChatConversation).toHaveBeenCalledTimes(1);
  });

  it("clears AI state when the canvas changes", async () => {
    const deferredChatSessions = createDeferred<Array<{ id: string; title: string }>>();

    loadChatSessions
      .mockResolvedValueOnce([{ id: "chat-1", title: "Chat 1" }])
      .mockImplementationOnce(() => deferredChatSessions.promise);

    const { rerender } = render(<BuildingBlocksSidebar {...defaultProps} />);

    fireEvent.click(screen.getByRole("tab", { name: "AI Builder" }));
    await flushEffects();

    expect(screen.getByTestId("chat-session-count")).toHaveTextContent("1");

    rerender(<BuildingBlocksSidebar {...defaultProps} canvasId="canvas-2" />);
    await act(async () => {
      await Promise.resolve();
    });

    fireEvent.click(screen.getByRole("tab", { name: "AI Builder" }));
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByTestId("chat-session-count")).toHaveTextContent("0");
    expect(screen.getByTestId("current-chat-id")).toHaveTextContent("none");
    expect(screen.getByTestId("ai-message-count")).toHaveTextContent("0");
  });
});
