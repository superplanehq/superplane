import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { liveLogScrollTrigger, useLiveLogStream } from "./useLiveLogStream";

const { captureExceptionMock, pumpMock, stopMock } = vi.hoisted(() => ({
  captureExceptionMock: vi.fn(),
  pumpMock: vi.fn(),
  stopMock: vi.fn(),
}));

vi.mock("@/sentry", () => ({
  Sentry: { captureException: captureExceptionMock },
}));

vi.mock("./liveLogStream", () => {
  class LiveLogStreamMock {
    pump = pumpMock;
    stop = stopMock;
  }

  return { LiveLogStream: LiveLogStreamMock };
});

vi.mock("@/hooks/useOrganizationId", () => ({
  useOrganizationId: () => undefined,
}));

vi.mock("@/hooks/useCanvasId", () => ({
  useCanvasId: () => undefined,
}));

beforeEach(() => {
  captureExceptionMock.mockReset();
  pumpMock.mockReset();
  stopMock.mockReset();
});

type ActivityHandlers = {
  onOpen?: () => void;
  onCmdStart?: (index: number, text: string, startedAtMs: number | null, kind?: string, preview?: string) => void;
  onCmdEnd?: (index: number, status: "passed" | "failed", durationMs: number) => void;
  onRecord?: (record: Record<string, unknown>) => void;
};

function playVersion2Activity(onRecord?: (record: Record<string, unknown>) => void) {
  const base = {
    schema_version: 2,
    activity_id: "act-1",
    provider: "claude",
    turn: 1,
  };
  onRecord?.({
    ...base,
    type: "content_start",
    id: "msg-block-0",
    channel: "assistant",
    event_id: "act-1:2",
    sequence: 2,
  });
  onRecord?.({
    ...base,
    type: "line",
    text: "I will verify the seams before updating the plan.",
    channel: "assistant",
    content_id: "msg-block-0",
    event_id: "act-1:3",
    sequence: 3,
  });
  onRecord?.({
    ...base,
    type: "content_end",
    id: "msg-block-0",
    channel: "assistant",
    event_id: "act-1:4",
    sequence: 4,
  });
  onRecord?.({
    ...base,
    type: "tool_start",
    id: "tool-1",
    kind: "bash",
    input: '{"command":"git status"}',
    event_id: "act-1:5",
    sequence: 5,
  });
  onRecord?.({
    ...base,
    type: "tool_end",
    id: "tool-1",
    status: "passed",
    duration_ms: 549,
    event_id: "act-1:6",
    sequence: 6,
  });
}

describe("useLiveLogStream agent activity", () => {
  it("reduces version 2 activity onto the current command section", async () => {
    pumpMock.mockImplementation(async (handlers: ActivityHandlers) => {
      handlers.onOpen?.();
      handlers.onCmdStart?.(5, "Implementation", 1, "prompt", "You are implementing");
      playVersion2Activity(handlers.onRecord);
    });

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.sections[0]?.activities).toHaveLength(1));
    const activity = result.current.sections[0]?.activities?.[0];
    expect(activity?.items).toMatchObject([
      { type: "content", kind: "assistant", text: "I will verify the seams before updating the plan." },
      { type: "tool", kind: "bash", status: "passed", durationMs: 549 },
    ]);
  });

  it("fills empty bash start input from later tool_input_delta records", async () => {
    pumpMock.mockImplementation(async (handlers: ActivityHandlers) => {
      handlers.onOpen?.();
      handlers.onCmdStart?.(5, "Implementation", 1, "prompt", "You are implementing");
      const base = {
        schema_version: 2,
        activity_id: "act-1",
        provider: "claude",
        turn: 1,
      };
      handlers.onRecord?.({
        ...base,
        type: "tool_start",
        id: "tool-1",
        kind: "bash",
        name: "Bash",
        input: "",
        event_id: "act-1:2",
        sequence: 2,
      });
      handlers.onRecord?.({
        ...base,
        type: "tool_input_delta",
        id: "tool-1",
        partial_json: '{"command":"git status"}',
        complete: true,
        event_id: "act-1:3",
        sequence: 3,
      });
      handlers.onRecord?.({
        ...base,
        type: "tool_input_delta",
        id: "tool-1",
        partial_json: "",
        complete: true,
        event_id: "act-1:4",
        sequence: 4,
      });
    });

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", false, "passed", null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.sections[0]?.activities?.[0]?.items).toHaveLength(1));
    expect(result.current.sections[0]?.activities?.[0]?.items).toMatchObject([
      { type: "tool", kind: "bash", name: "Bash", input: '{"command":"git status"}' },
    ]);
  });

  it("does not duplicate version 2 activity after a reconnect replay", async () => {
    const play = (handlers: ActivityHandlers) => {
      handlers.onOpen?.();
      handlers.onCmdStart?.(5, "Implementation", 1, "prompt", "You are implementing");
      playVersion2Activity(handlers.onRecord);
      handlers.onCmdEnd?.(5, "passed", 20);
    };

    pumpMock.mockImplementationOnce(async (handlers: ActivityHandlers) => {
      play(handlers);
    });
    pumpMock.mockImplementationOnce(async (handlers: ActivityHandlers) => {
      play(handlers);
      return new Promise(() => undefined);
    });

    const { result } = renderHook(() =>
      useLiveLogStream("execution-1", true, null, null, {
        organizationId: "organization-1",
        canvasId: "canvas-1",
      }),
    );

    await waitFor(() => expect(result.current.sections[0]?.activities?.[0]?.items.length).toBe(2));
    await waitFor(() => expect(pumpMock).toHaveBeenCalledTimes(2), { timeout: 5000 });
    expect(result.current.sections).toHaveLength(1);
    expect(result.current.sections[0]?.activities).toHaveLength(1);
    expect(result.current.sections[0]?.activities?.[0]?.items).toHaveLength(2);
  });

  it("changes the scroll trigger when activity sequence advances without new items", () => {
    const section = {
      index: 0,
      text: "Implementation",
      lines: [] as string[],
      events: [],
      activities: [
        {
          id: "act-1",
          provider: "claude",
          status: "running" as const,
          sequence: 3,
          items: [
            {
              type: "content" as const,
              id: "msg",
              kind: "assistant" as const,
              text: "Hello",
              status: "running" as const,
              truncated: false,
            },
          ],
          truncated: false,
        },
      ],
      status: "running" as const,
      duration_ms: null,
      started_at: 1,
      collapsed: false,
    };
    const before = liveLogScrollTrigger({ sections: [section], orphanLines: [] });
    const after = liveLogScrollTrigger({
      sections: [
        {
          ...section,
          activities: [
            {
              ...section.activities[0],
              sequence: 4,
              items: [{ ...section.activities[0].items[0], text: "Hello world" }],
            },
          ],
        },
      ],
      orphanLines: [],
    });

    expect(before).not.toEqual(after);
  });
});
