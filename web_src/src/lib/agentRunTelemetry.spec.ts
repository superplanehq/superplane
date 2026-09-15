import { describe, expect, it } from "bun:test";
import {
  applyAgentTelemetryRecord,
  applyPromptUsageRecord,
  emptyAgentRunTelemetry,
  emptyPromptUsageState,
  parseAgentTurnLiveLogText,
  preferLiveTelemetry,
  promptUsageSeries,
  reduceAgentTelemetryRecords,
  startPromptUsageSeries,
  telemetryFromExecutionOutputs,
  telemetryFromFinishedResult,
  telemetrySeriesFromFinishedResult,
} from "./agentRunTelemetry";
import { chartPointsForTelemetry } from "./agentRunTelemetryChart";

describe("reduceAgentTelemetryRecords", () => {
  it("joins turn usage with tool records on the same turn", () => {
    const telemetry = reduceAgentTelemetryRecords([
      { type: "turn", turn: 1, usage: { input_tokens: 100, output_tokens: 20 } },
      { type: "tool_start", turn: 1, id: "a", kind: "bash", text: "make pb.gen" },
      { type: "tool_end", turn: 1, id: "a", status: "passed", duration_ms: 40 },
      { type: "tool_start", turn: 1, id: "b", kind: "bash", text: "git status" },
      { type: "turn", turn: 2, usage: { input_tokens: 180, output_tokens: 30 } },
      { type: "tool_start", turn: 2, id: "c", kind: "read", text: "pkg/protos/canvases.pb.go" },
    ]);

    expect(telemetry.num_turns).toBe(2);
    expect(telemetry.usage.input_tokens).toBe(280);
    expect(telemetry.tool_counts).toEqual({ bash: 2, read: 1 });
    expect(telemetry.turns[0].tools).toEqual([
      { id: "a", kind: "bash", text: "make pb.gen", status: "passed", duration_ms: 40 },
      { id: "b", kind: "bash", text: "git status", status: "running" },
    ]);
    expect(telemetry.turns[1].tools[0]).toMatchObject({ kind: "read", text: "pkg/protos/canvases.pb.go" });
  });

  it("keeps the agent message on the turn", () => {
    const telemetry = reduceAgentTelemetryRecords([
      { type: "turn", turn: 1, usage: { input_tokens: 10, output_tokens: 4 }, message: "I will inspect the remotes." },
      { type: "tool_start", turn: 1, kind: "bash", text: "git remote -v" },
    ]);
    expect(telemetry.turns[0].message).toBe("I will inspect the remotes.");
  });

  it("keeps consecutive turns that share a usage snapshot", () => {
    const telemetry = reduceAgentTelemetryRecords([
      { type: "turn", turn: 1, usage: { input_tokens: 2, output_tokens: 5, cache_read_input_tokens: 1448 } },
      { type: "turn", turn: 2, usage: { input_tokens: 2, output_tokens: 5, cache_read_input_tokens: 1448 } },
      { type: "turn", turn: 3, usage: { input_tokens: 2, output_tokens: 21, cache_read_input_tokens: 5394 } },
    ]);
    expect(telemetry.num_turns).toBe(3);
    expect(telemetry.turns.map((turn) => turn.turn)).toEqual([1, 2, 3]);
  });

  it("creates a turn when tools arrive before a usage record", () => {
    const telemetry = applyAgentTelemetryRecord(emptyAgentRunTelemetry(), {
      type: "tool_start",
      kind: "bash",
      text: "true",
    });
    expect(telemetry.turns).toHaveLength(1);
    expect(telemetry.turns[0].turn).toBe(1);
    expect(telemetry.turns[0].tools[0].text).toBe("true");
  });
});

describe("chartPointsForTelemetry", () => {
  const telemetry = reduceAgentTelemetryRecords([
    { type: "turn", turn: 1, usage: { input_tokens: 100, output_tokens: 20 } },
    { type: "tool_start", turn: 1, kind: "bash", text: "make pb.gen" },
    { type: "turn", turn: 2, usage: { input_tokens: 80, output_tokens: 10 } },
    { type: "tool_start", turn: 2, kind: "read", text: "a.go" },
    { type: "tool_start", turn: 2, kind: "bash", text: "git status" },
  ]);

  it("uses each turn's own usage for tokens", () => {
    const points = chartPointsForTelemetry(telemetry);
    expect(points[0].tokens).toBe(120);
    expect(points[1].input_tokens).toBe(80);
    expect(points[1].output_tokens).toBe(10);
    expect(points[1].tokens).toBe(90);
    expect(points[0].inputBar).toBe(100);
    expect(points[0].outputBar).toBe(20);
    expect(points[1].inputBar).toBe(80);
    expect(points[1].outputBar).toBe(10);
  });

  it("does not count cached context in the bar", () => {
    const cached = reduceAgentTelemetryRecords([
      {
        type: "turn",
        turn: 1,
        usage: { input_tokens: 2, output_tokens: 5, cache_read_input_tokens: 1448, cache_creation_input_tokens: 100 },
      },
      {
        type: "turn",
        turn: 2,
        usage: {
          input_tokens: 100,
          output_tokens: 20,
          cache_read_input_tokens: 5000,
          cache_creation_input_tokens: 50,
        },
      },
    ]);
    const points = chartPointsForTelemetry(cached);
    expect(points[0].tokens).toBe(107);
    expect(points[0].inputBar).toBe(102);
    expect(points[0].outputBar).toBe(5);
    expect(points[0].cache_read_tokens).toBe(1448);
    expect(points[1].tokens).toBe(170);
    expect(points[1].inputBar).toBe(150);
    expect(points[1].outputBar).toBe(20);
    expect(points[1].cache_read_tokens).toBe(5000);
  });
});

describe("promptUsageSeries", () => {
  it("starts a new series on each prompt command", () => {
    let state = startPromptUsageSeries(emptyPromptUsageState(), "Implementation");
    state = applyPromptUsageRecord(state, { type: "turn", turn: 1, usage: { input_tokens: 10, output_tokens: 2 } });
    state = applyPromptUsageRecord(state, { type: "tool_start", turn: 1, kind: "bash", text: "git status" });
    state = startPromptUsageSeries(state, "Generate PR title and description");
    state = applyPromptUsageRecord(state, {
      type: "turn",
      turn: 1,
      usage: { input_tokens: 4, output_tokens: 1 },
    });
    state = applyPromptUsageRecord(state, {
      type: "tool_start",
      turn: 1,
      kind: "bash",
      text: "git log main..feature/shuffle-deck-endpoint --oneline",
    });

    const series = promptUsageSeries(state);
    expect(series).toHaveLength(2);
    expect(series[0].name).toBe("Implementation");
    expect(series[0].telemetry.num_turns).toBe(1);
    expect(series[0].telemetry.turns[0].tools[0].text).toBe("git status");
    expect(series[1].name).toBe("Generate PR title and description");
    expect(series[1].telemetry.num_turns).toBe(1);
    expect(series[1].telemetry.turns[0].tools[0].text).toContain("git log");
  });

  it("does not start the same prompt index twice", () => {
    let state = startPromptUsageSeries(emptyPromptUsageState(), "Implementation", 2);
    state = applyPromptUsageRecord(state, { type: "turn", turn: 1, usage: { input_tokens: 10, output_tokens: 2 } });
    state = startPromptUsageSeries(state, "Implementation", 2);
    state = applyPromptUsageRecord(state, { type: "turn", turn: 2, usage: { input_tokens: 8, output_tokens: 1 } });

    const series = promptUsageSeries(state);
    expect(series).toHaveLength(1);
    expect(series[0].name).toBe("Implementation");
    expect(series[0].telemetry.num_turns).toBe(2);
  });

  it("does not let a later prompt overwrite earlier turns with the same numbers", () => {
    let state = startPromptUsageSeries(emptyPromptUsageState(), "Implementation", 2);
    state = applyPromptUsageRecord(state, { type: "turn", turn: 1, usage: { input_tokens: 10, output_tokens: 4 } });
    state = applyPromptUsageRecord(state, { type: "turn", turn: 2, usage: { input_tokens: 8, output_tokens: 3 } });
    state = startPromptUsageSeries(state, "Generate PR title and description", 4);
    state = applyPromptUsageRecord(state, { type: "turn", turn: 1, usage: { input_tokens: 2, output_tokens: 5 } });

    const series = promptUsageSeries(state);
    expect(series).toHaveLength(2);
    expect(series[0].telemetry.turns.map((turn) => turn.usage.input_tokens)).toEqual([10, 8]);
    expect(series[1].telemetry.turns[0].usage.input_tokens).toBe(2);
  });
});

describe("telemetryFromFinishedResult", () => {
  it("reads the joined series from a finished result", () => {
    const telemetry = telemetryFromFinishedResult({
      telemetry: {
        num_turns: 1,
        usage: { input_tokens: 4, output_tokens: 1 },
        tool_counts: { bash: 1 },
        turns: [
          {
            turn: 1,
            usage: { input_tokens: 4, output_tokens: 1 },
            tools: [{ kind: "bash", text: "true", status: "passed" }],
          },
        ],
      },
    });
    expect(telemetry?.turns[0].tools[0].text).toBe("true");
  });

  it("reads the agent message from a finished result", () => {
    const telemetry = telemetryFromFinishedResult({
      telemetry: {
        turns: [
          {
            turn: 1,
            usage: { input_tokens: 4, output_tokens: 1 },
            tools: [],
            message: "I will inspect the remotes.",
          },
        ],
      },
    });
    expect(telemetry?.turns[0].message).toBe("I will inspect the remotes.");
  });

  it("reads one series per prompt from a finished result", () => {
    const series = telemetrySeriesFromFinishedResult({
      telemetry: {
        prompts: [
          {
            name: "Implementation",
            telemetry: { turns: [{ turn: 1, usage: { input_tokens: 10 }, tools: [] }] },
          },
          {
            name: "Generate PR title and description",
            telemetry: { turns: [{ turn: 1, usage: { input_tokens: 2 }, tools: [] }] },
          },
        ],
      },
    });
    expect(series).toHaveLength(2);
    expect(series[0].name).toBe("Implementation");
    expect(series[1].telemetry.turns[0].usage.input_tokens).toBe(2);
  });

  it("reads telemetry from execution outputs", () => {
    const telemetry = telemetryFromExecutionOutputs({
      passed: [{ data: { result: { telemetry: { turns: [{ turn: 1, usage: { input_tokens: 2 }, tools: [] }] } } } }],
    });
    expect(telemetry?.turns[0].usage.input_tokens).toBe(2);
  });
});

describe("parseAgentTurnLiveLogText", () => {
  it("reads a turn record and ignores ordinary log text", () => {
    const raw =
      '{"type":"turn","turn":3,"usage":{"input_tokens":2000,"output_tokens":100,"cache_read_input_tokens":1200,"cache_creation_input_tokens":0,"reasoning_tokens":0}}';
    expect(parseAgentTurnLiveLogText(raw)).toEqual({
      turn: 3,
      usage: {
        input_tokens: 2000,
        output_tokens: 100,
        cache_read_input_tokens: 1200,
        cache_creation_input_tokens: 0,
        reasoning_tokens: 0,
      },
    });
    expect(parseAgentTurnLiveLogText("I'll start by reading the task.")).toBeNull();
  });

  it("reads the agent message from a turn record", () => {
    const raw =
      '{"type":"turn","turn":1,"usage":{"input_tokens":2,"output_tokens":4},"message":"I will inspect the remotes."}';
    expect(parseAgentTurnLiveLogText(raw)?.message).toBe("I will inspect the remotes.");
  });
});

describe("preferLiveTelemetry", () => {
  it("uses finished output when the live series is empty", () => {
    const finished = telemetryFromFinishedResult({
      telemetry: { turns: [{ turn: 1, usage: { input_tokens: 9 }, tools: [] }] },
    });
    expect(preferLiveTelemetry(emptyAgentRunTelemetry(), finished).usage.input_tokens).toBe(9);
  });
});
