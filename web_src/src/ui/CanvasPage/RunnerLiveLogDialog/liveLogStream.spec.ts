import { describe, expect, it, vi } from "bun:test";

import {
  consumeLiveLogNdjsonLine,
  LiveLogStream,
  reducePromptUsageFromLiveLogLines,
  type LiveLogStreamHandlers,
} from "./liveLogStream";

function handlers(overrides: Partial<LiveLogStreamHandlers> = {}): LiveLogStreamHandlers {
  return {
    onLogLine: vi.fn(),
    onStreamError: vi.fn(),
    onTurn: vi.fn(),
    ...overrides,
  };
}

describe("consumeLiveLogNdjsonLine", () => {
  it("routes a structured turn record to onTurn", () => {
    const next = handlers();
    consumeLiveLogNdjsonLine('{"type":"turn","turn":3,"usage":{"input_tokens":10,"output_tokens":2}}', next);

    expect(next.onTurn).toHaveBeenCalledWith(3, { input_tokens: 10, output_tokens: 2 }, undefined);
    expect(next.onLogLine).not.toHaveBeenCalled();
  });

  it("routes the agent message from a turn record", () => {
    const next = handlers();
    consumeLiveLogNdjsonLine(
      '{"type":"turn","turn":1,"usage":{"input_tokens":10,"output_tokens":2},"message":"I will inspect the remotes."}',
      next,
    );

    expect(next.onTurn).toHaveBeenCalledWith(1, { input_tokens: 10, output_tokens: 2 }, "I will inspect the remotes.");
  });

  it("unwraps a turn record that arrived as a plaintext line", () => {
    const next = handlers();
    consumeLiveLogNdjsonLine(
      JSON.stringify({
        type: "line",
        text: '{"type":"turn","turn":4,"usage":{"input_tokens":20,"output_tokens":1}}',
      }),
      next,
    );

    expect(next.onTurn).toHaveBeenCalledWith(
      4,
      {
        input_tokens: 20,
        output_tokens: 1,
        cache_read_input_tokens: 0,
        cache_creation_input_tokens: 0,
        reasoning_tokens: 0,
      },
      undefined,
    );
    expect(next.onLogLine).not.toHaveBeenCalled();
  });

  it("keeps ordinary log lines", () => {
    const next = handlers();
    consumeLiveLogNdjsonLine(JSON.stringify({ type: "line", text: "Claude Code started" }), next);

    expect(next.onLogLine).toHaveBeenCalledWith("Claude Code started");
    expect(next.onTurn).not.toHaveBeenCalled();
  });

  it("forwards a command index on a log line", () => {
    const next = handlers();
    consumeLiveLogNdjsonLine(JSON.stringify({ type: "line", index: 2, text: "Hello" }), next);

    expect(next.onLogLine).toHaveBeenCalledWith("Hello", 2);
  });

  it("sends version 2 activity records only to the activity consumer", () => {
    const next = handlers({ onRecord: vi.fn(), onToolStart: vi.fn() });
    const record = { schema_version: 2, type: "tool_start", id: "call-1", kind: "bash", text: "pwd" };

    consumeLiveLogNdjsonLine(JSON.stringify(record), next);

    expect(next.onRecord).toHaveBeenCalledWith(record);
    expect(next.onToolStart).not.toHaveBeenCalled();
    expect(next.onLogLine).not.toHaveBeenCalled();
  });

  it("unwraps a namespaced tool record that arrived as a plaintext line", () => {
    const next = handlers({ onRecord: vi.fn(), onToolStart: vi.fn() });
    const record = {
      schema_version: 2,
      type: "activity_tool_start",
      activity_id: "activity-1",
      event_id: "activity-1:3",
      sequence: 3,
      id: "call-1",
      kind: "bash",
      input: "git status",
    };

    consumeLiveLogNdjsonLine(JSON.stringify({ type: "line", text: JSON.stringify(record) }), next);

    expect(next.onRecord).toHaveBeenCalledTimes(1);
    expect(next.onRecord).toHaveBeenCalledWith(record);
    expect(next.onToolStart).not.toHaveBeenCalled();
    expect(next.onLogLine).not.toHaveBeenCalled();
  });

  it("splits each prompt command into its own usage series", () => {
    const series = reducePromptUsageFromLiveLogLines([
      JSON.stringify({ type: "cmd_start", index: 2, text: "Implementation", kind: "prompt" }),
      JSON.stringify({ type: "line", text: '{"type":"turn","turn":1,"usage":{"input_tokens":10,"output_tokens":4}}' }),
      JSON.stringify({ type: "tool_start", id: "a", kind: "bash", text: "git status" }),
      JSON.stringify({ type: "line", text: '{"type":"turn","turn":2,"usage":{"input_tokens":8,"output_tokens":3}}' }),
      JSON.stringify({ type: "cmd_end", index: 2, status: "passed", duration_ms: 1000 }),
      JSON.stringify({ type: "cmd_start", index: 4, text: "Generate PR title and description", kind: "prompt" }),
      JSON.stringify({ type: "line", text: '{"type":"turn","turn":1,"usage":{"input_tokens":2,"output_tokens":5}}' }),
      JSON.stringify({ type: "tool_start", id: "b", kind: "bash", text: "git log main..feature --oneline" }),
    ]);

    expect(series).toHaveLength(2);
    expect(series[0].name).toBe("Implementation");
    expect(series[0].telemetry.num_turns).toBe(2);
    expect(series[0].telemetry.turns[0].tools[0].text).toBe("git status");
    expect(series[1].name).toBe("Generate PR title and description");
    expect(series[1].telemetry.num_turns).toBe(1);
    expect(series[1].telemetry.turns[0].usage.input_tokens).toBe(2);
    expect(series[1].telemetry.turns[0].tools[0].text).toContain("git log");
  });

  it("keeps the agent message from the turn record only", () => {
    const series = reducePromptUsageFromLiveLogLines([
      JSON.stringify({ type: "cmd_start", index: 2, text: "Implementation", kind: "prompt" }),
      JSON.stringify({ type: "line", text: "I will inspect the remotes." }),
      JSON.stringify({
        type: "line",
        text: '{"type":"turn","turn":1,"usage":{"input_tokens":10,"output_tokens":4},"message":"I will inspect the remotes."}',
      }),
      JSON.stringify({ type: "line", text: "partial stream" }),
      JSON.stringify({
        type: "line",
        text: '{"type":"turn","turn":2,"usage":{"input_tokens":8,"output_tokens":3}}',
      }),
    ]);

    expect(series[0].telemetry.turns[0].message).toBe("I will inspect the remotes.");
    expect(series[0].telemetry.turns[1].message).toBeUndefined();
  });
});

describe("LiveLogStream integrated polling", () => {
  it("preserves the opaque cursor and stops at the archived state", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            backend: "integrated",
            stream_url: "/runner-logs",
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response('{"type":"line","text":"first"}\n', {
          status: 200,
          headers: {
            "X-SuperPlane-Log-Cursor": "cursor-1",
            "X-SuperPlane-Log-State": "active",
          },
        }),
      )
      .mockResolvedValueOnce(
        new Response('{"type":"line","text":"second"}\n', {
          status: 200,
          headers: {
            "X-SuperPlane-Log-Cursor": "cursor-2",
            "X-SuperPlane-Log-State": "archived",
          },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    try {
      const onLogLine = vi.fn();
      const final = new LiveLogStream("organization-1", "canvas-1", "execution-1", 0).pump(handlers({ onLogLine }));

      expect(await final).toBe(true);
      expect(fetchMock.mock.calls[2]?.[0]).toBe("/runner-logs?after=cursor-1");
      expect(onLogLine).toHaveBeenNthCalledWith(1, "first");
      expect(onLogLine).toHaveBeenNthCalledWith(2, "second");
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("clears a stale cursor before loading the final object", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ backend: "integrated", stream_url: "/runner-logs" }), { status: 200 }),
      )
      .mockResolvedValueOnce(
        new Response('{"type":"line","text":"active"}\n', {
          status: 200,
          headers: {
            "X-SuperPlane-Log-Cursor": "cursor-1",
            "X-SuperPlane-Log-State": "active",
          },
        }),
      )
      .mockResolvedValueOnce(
        new Response(null, {
          status: 409,
          headers: { "X-SuperPlane-Log-Reset": "true" },
        }),
      )
      .mockResolvedValueOnce(
        new Response('{"type":"line","text":"final"}\n', {
          status: 200,
          headers: {
            "X-SuperPlane-Log-Cursor": "cursor-2",
            "X-SuperPlane-Log-State": "archived",
          },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    try {
      const onReset = vi.fn();
      const final = await new LiveLogStream("organization-1", "canvas-1", "execution-1", 0).pump(handlers({ onReset }));

      expect(final).toBe(true);
      expect(onReset).toHaveBeenCalledTimes(1);
      expect(fetchMock.mock.calls[2]?.[0]).toBe("/runner-logs?after=cursor-1");
      expect(fetchMock.mock.calls[3]?.[0]).toBe("/runner-logs");
    } finally {
      vi.unstubAllGlobals();
    }
  });
});
