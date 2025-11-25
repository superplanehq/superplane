import { describe, it, expect, beforeEach, afterEach, vi, Mock } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import useWebSocket from "react-use-websocket";
import React from "react";
import { useWorkflowWebsocket } from "./useWorkflowWebsocket";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { WorkflowsWorkflowNodeExecution, WorkflowsWorkflowEvent, WorkflowsWorkflowNodeQueueItem } from "@/api-client";

vi.mock("react-use-websocket", () => ({
  default: vi.fn(),
}));

vi.mock("@/stores/nodeExecutionStore", () => ({
  useNodeExecutionStore: vi.fn(),
}));

vi.mock("./useWorkflowData", () => ({
  workflowKeys: {
    eventExecution: vi.fn((workflowId: string, eventId: string) => ["workflow", workflowId, "eventExecution", eventId]),
  },
}));

describe("useWorkflowWebsocket", () => {
  let mockStore: {
    updateNodeEvent: Mock;
    updateNodeExecution: Mock;
    addNodeQueueItem: Mock;
    removeNodeQueueItem: Mock;
  };
  let mockOnMessage: Mock;
  let mockUseWebSocket: Mock;
  let queryClient: QueryClient;
  let mockOnNodeEvent: Mock;

  const mockExecution: WorkflowsWorkflowNodeExecution = {
    id: "exec-1",
    nodeId: "node-1",
    status: "finished",
    rootEvent: { id: "event-1" },
  };

  const mockEvent: WorkflowsWorkflowEvent = {
    id: "event-1",
    nodeId: "node-1",
    type: "webhook",
  };

  const mockQueueItem: WorkflowsWorkflowNodeQueueItem = {
    id: "queue-1",
    nodeId: "node-1",
    status: "pending",
  };

  beforeEach(() => {
    vi.useFakeTimers();

    mockStore = {
      updateNodeEvent: vi.fn(),
      updateNodeExecution: vi.fn(),
      addNodeQueueItem: vi.fn(),
      removeNodeQueueItem: vi.fn(),
    };

    mockOnNodeEvent = vi.fn();
    mockUseWebSocket = vi.mocked(useWebSocket);

    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    queryClient.invalidateQueries = vi.fn();

    (useNodeExecutionStore as Mock).mockReturnValue(mockStore);

    mockUseWebSocket.mockImplementation((url, options) => {
      mockOnMessage = options?.onMessage;
      return {};
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  const renderHookWithProvider = (props: {
    workflowId: string;
    organizationId: string;
    onNodeEvent?: (nodeId: string, event: string) => void;
  }) => {
    return renderHook(() => useWorkflowWebsocket(props.workflowId, props.organizationId, props.onNodeEvent), {
      wrapper: ({ children }) => React.createElement(QueryClientProvider, { client: queryClient }, children),
    });
  };

  describe("Basic Hook Setup", () => {
    it("should initialize websocket connection with correct URL", () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      expect(mockUseWebSocket).toHaveBeenCalledWith(
        expect.stringContaining("workflow-1?organization_id=org-1"),
        expect.objectContaining({
          shouldReconnect: expect.any(Function),
          reconnectAttempts: 10,
          heartbeat: false,
          reconnectInterval: 3000,
          share: false,
          onMessage: expect.any(Function),
        }),
      );
    });

    it("should use websocket URL based on current protocol", () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      expect(mockUseWebSocket).toHaveBeenCalledWith(
        expect.stringMatching(/^wss?:\/\/.*\/ws\/workflow-1\?organization_id=org-1$/),
        expect.any(Object),
      );
    });
  });

  describe("Message Queuing", () => {
    it("should queue messages instead of processing them immediately", () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const message1 = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      const message2 = {
        data: JSON.stringify({
          event: "execution_created",
          payload: mockExecution,
        }),
      };

      act(() => {
        mockOnMessage({ data: message1.data });
        mockOnMessage({ data: message2.data });
      });

      expect(mockStore.updateNodeEvent).not.toHaveBeenCalled();
      expect(mockStore.updateNodeExecution).not.toHaveBeenCalled();
    });

    it("should handle malformed JSON gracefully", () => {
      const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      act(() => {
        mockOnMessage({ data: "invalid json" });
      });

      expect(consoleErrorSpy).toHaveBeenCalledWith("Error parsing message:", expect.any(Error));
      consoleErrorSpy.mockRestore();
    });
  });

  describe("Sequential Processing and Ordering", () => {
    it("should process messages in sequence", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const firstMessage = {
        data: JSON.stringify({
          event: "event_created",
          payload: { ...mockEvent, id: "event-first" },
        }),
      };

      const secondMessage = {
        data: JSON.stringify({
          event: "event_created",
          payload: { ...mockEvent, id: "event-second" },
        }),
      };

      act(() => {
        mockOnMessage({ data: firstMessage.data });
        mockOnMessage({ data: secondMessage.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledTimes(2);
      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", { ...mockEvent, id: "event-first" });
      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", { ...mockEvent, id: "event-second" });
    });

    it("should handle message processing without errors", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const message = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", mockEvent);
    });

    it("should prevent concurrent processing", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const message1 = {
        data: JSON.stringify({
          event: "event_created",
          payload: { ...mockEvent, id: "event-1" },
        }),
      };

      const message2 = {
        data: JSON.stringify({
          event: "event_created",
          payload: { ...mockEvent, id: "event-2" },
        }),
      };

      act(() => {
        mockOnMessage({ data: message1.data });
        mockOnMessage({ data: message2.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledTimes(2);
      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", { ...mockEvent, id: "event-1" });
      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", { ...mockEvent, id: "event-2" });
    });
  });

  describe("Batched Updates", () => {
    it("should batch multiple operations for the same node", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const eventMessage = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      const executionMessage = {
        data: JSON.stringify({
          event: "execution_created",
          payload: mockExecution,
        }),
      };

      act(() => {
        mockOnMessage({ data: eventMessage.data });
        mockOnMessage({ data: executionMessage.data });
      });

      expect(mockStore.updateNodeEvent).not.toHaveBeenCalled();
      expect(mockStore.updateNodeExecution).not.toHaveBeenCalled();

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", mockEvent);
      expect(mockStore.updateNodeExecution).toHaveBeenCalledWith("node-1", mockExecution);
      expect(mockOnNodeEvent).toHaveBeenCalledTimes(2);
    });

    it("should reset batch timer on new messages", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const message = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(10);
      });

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(10);
      });

      expect(mockStore.updateNodeEvent).not.toHaveBeenCalled();

      act(() => {
        vi.advanceTimersByTime(10);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledTimes(2);
    });
  });

  describe("Event Type Handling", () => {
    it("should handle event_created messages", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const message = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", mockEvent);
      expect(mockOnNodeEvent).toHaveBeenCalledWith("node-1", "event_created");
    });

    it("should handle execution lifecycle events", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const events = ["execution_created", "execution_started", "execution_finished"];

      for (const eventType of events) {
        const message = {
          data: JSON.stringify({
            event: eventType,
            payload: { ...mockExecution, id: `exec-${eventType}` },
          }),
        };

        act(() => {
          mockOnMessage({ data: message.data });
        });
      }

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeExecution).toHaveBeenCalledTimes(3);
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(3);
    });

    it("should handle child execution nodeId extraction", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const childExecution = {
        ...mockExecution,
        nodeId: "parent-node:child-node",
        parentExecutionId: "parent-exec-1",
      };

      const message = {
        data: JSON.stringify({
          event: "execution_created",
          payload: childExecution,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeExecution).toHaveBeenCalledWith("parent-node", childExecution);
    });

    it("should handle queue item operations", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
        onNodeEvent: mockOnNodeEvent,
      });

      const createMessage = {
        data: JSON.stringify({
          event: "queue_item_created",
          payload: mockQueueItem,
        }),
      };

      act(() => {
        mockOnMessage({ data: createMessage.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.addNodeQueueItem).toHaveBeenCalledWith("node-1", mockQueueItem);
      expect(mockOnNodeEvent).toHaveBeenCalledWith("node-1", "queue_item_created");

      const consumeMessage = {
        data: JSON.stringify({
          event: "queue_item_consumed",
          payload: mockQueueItem,
        }),
      };

      act(() => {
        mockOnMessage({ data: consumeMessage.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.removeNodeQueueItem).toHaveBeenCalledWith("node-1", "queue-1");
      expect(mockOnNodeEvent).toHaveBeenCalledWith("node-1", "queue_item_consumed");
    });

    it("should handle queue_item_consumed without id gracefully", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const queueItemWithoutId = { ...mockQueueItem, id: undefined };
      const message = {
        data: JSON.stringify({
          event: "queue_item_consumed",
          payload: queueItemWithoutId,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.removeNodeQueueItem).not.toHaveBeenCalled();
    });

    it("should ignore unknown event types", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const message = {
        data: JSON.stringify({
          event: "unknown_event",
          payload: { someData: "test" },
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).not.toHaveBeenCalled();
      expect(mockStore.updateNodeExecution).not.toHaveBeenCalled();
      expect(mockStore.addNodeQueueItem).not.toHaveBeenCalled();
      expect(mockStore.removeNodeQueueItem).not.toHaveBeenCalled();
    });

    it("should handle messages without nodeId gracefully", async () => {
      renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const eventWithoutNodeId = { ...mockEvent, nodeId: undefined };
      const message = {
        data: JSON.stringify({
          event: "event_created",
          payload: eventWithoutNodeId,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      act(() => {
        vi.advanceTimersByTime(20);
      });

      expect(mockStore.updateNodeEvent).not.toHaveBeenCalled();
    });
  });

  describe("Cleanup", () => {
    it("should process remaining batched updates on unmount", async () => {
      const { unmount } = renderHookWithProvider({
        workflowId: "workflow-1",
        organizationId: "org-1",
      });

      const message = {
        data: JSON.stringify({
          event: "event_created",
          payload: mockEvent,
        }),
      };

      act(() => {
        mockOnMessage({ data: message.data });
      });

      unmount();

      expect(mockStore.updateNodeEvent).toHaveBeenCalledWith("node-1", mockEvent);
    });
  });
});
