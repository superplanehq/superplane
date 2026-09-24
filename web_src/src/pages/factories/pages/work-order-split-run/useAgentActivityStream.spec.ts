import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type { AgentActivityRecord } from "./agentActivity";
import { useAgentActivityStream } from "./useAgentActivityStream";

const { pumpMock, stopMock } = vi.hoisted(() => ({
  pumpMock: vi.fn(),
  stopMock: vi.fn(),
}));

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/liveLogStream", () => {
  class LiveLogStreamMock {
    pump = pumpMock;
    stop = stopMock;
  }

  return { LiveLogStream: LiveLogStreamMock };
});

type StreamHandlers = {
  onOpen?: () => void;
  onRecord?: (record: AgentActivityRecord) => void;
};

const activityRecord: AgentActivityRecord = {
  schema_version: 2,
  type: "activity_start",
  activity_id: "activity-1",
  event_id: "event-1",
  sequence: 1,
  provider: "claude",
};

beforeEach(() => {
  pumpMock.mockReset();
  stopMock.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

function hangAfterOpen(onReady?: (handlers: StreamHandlers) => void) {
  return (handlers: StreamHandlers) => {
    handlers.onOpen?.();
    onReady?.(handlers);
    return new Promise<void>((resolve) => {
      stopMock.mockImplementation(() => resolve());
    });
  };
}

function renderStream() {
  return renderHook(() =>
    useAgentActivityStream({
      organizationId: "organization-1",
      canvasId: "canvas-1",
      executionId: "execution-1",
      active: true,
    }),
  );
}

describe("useAgentActivityStream", () => {
  it("stops a silent open feed after 15 seconds and starts another session", async () => {
    vi.useFakeTimers();
    pumpMock.mockImplementation(hangAfterOpen());

    const { result } = renderStream();

    await act(async () => {
      await Promise.resolve();
    });
    expect(pumpMock).toHaveBeenCalledTimes(1);
    expect(result.current.isConnected).toBe(true);
    expect(result.current.hasConnectedOnce).toBe(true);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(14_000);
    });
    expect(pumpMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1_000);
    });
    expect(stopMock).toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2_000);
    });
    expect(pumpMock).toHaveBeenCalledTimes(2);
    await waitFor(() => expect(result.current.isConnected).toBe(true));
    expect(result.current.hasConnectedOnce).toBe(true);
  });

  it("resets the quiet timer on a later record and does not reconnect again", async () => {
    vi.useFakeTimers();
    let handlers: StreamHandlers | undefined;
    pumpMock.mockImplementation(
      hangAfterOpen((next) => {
        handlers = next;
      }),
    );

    renderStream();

    await act(async () => {
      await Promise.resolve();
    });
    expect(pumpMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(15_000);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2_000);
    });
    expect(pumpMock).toHaveBeenCalledTimes(2);
    const stopsAfterReconnect = stopMock.mock.calls.length;

    act(() => {
      handlers?.onRecord?.(activityRecord);
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(14_000);
    });
    expect(pumpMock).toHaveBeenCalledTimes(2);
    expect(stopMock.mock.calls.length).toBe(stopsAfterReconnect);
  });
});
