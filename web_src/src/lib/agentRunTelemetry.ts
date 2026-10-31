import { parseAgentActivityRecordText } from "@/lib/agentActivity";

export type UsageMeasurementState = "pending" | "measured" | "partial" | "unavailable";

export function usageMeasurementState(value: unknown): UsageMeasurementState | undefined {
  return value === "pending" || value === "measured" || value === "partial" || value === "unavailable"
    ? value
    : undefined;
}

export type AgentTurnUsage = {
  input_tokens: number;
  output_tokens: number;
  cache_read_input_tokens: number;
  cache_creation_input_tokens: number;
  reasoning_tokens: number;
  total_cost_usd?: number;
  hosted_cost_usd?: number;
  total_tokens?: number;
};

export type AgentTurnTool = {
  id?: string;
  kind: string;
  text: string;
  status?: "passed" | "failed" | "running";
  duration_ms?: number;
};

export type AgentTurnSnapshot = {
  usage_state?: UsageMeasurementState;
  turn: number;
  usage: AgentTurnUsage;
  tools: AgentTurnTool[];
  message?: string;
};

export type AgentRunTelemetry = {
  usage_state?: UsageMeasurementState;
  num_turns: number;
  usage: AgentTurnUsage;
  tool_counts: Record<string, number>;
  turns: AgentTurnSnapshot[];
};

export type AgentTelemetryLiveRecord =
  | {
      type: "turn";
      turn: number;
      usage: Partial<AgentTurnUsage>;
      message?: string;
      usage_state?: UsageMeasurementState;
    }
  | { type: "tool_start"; turn?: number; id?: string; kind?: string; text?: string }
  | {
      type: "tool_end";
      turn?: number;
      id?: string;
      kind?: string;
      status?: AgentTurnTool["status"];
      duration_ms?: number;
    };

const EMPTY_USAGE: AgentTurnUsage = {
  input_tokens: 0,
  output_tokens: 0,
  cache_read_input_tokens: 0,
  cache_creation_input_tokens: 0,
  reasoning_tokens: 0,
};

export function emptyAgentRunTelemetry(): AgentRunTelemetry {
  return {
    num_turns: 0,
    usage: { ...EMPTY_USAGE },
    tool_counts: {},
    turns: [],
  };
}

function asNumber(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

export function normalizeAgentTurnUsage(raw: unknown): AgentTurnUsage {
  if (!raw || typeof raw !== "object") {
    return { ...EMPTY_USAGE };
  }
  const src = raw as Record<string, unknown>;
  const usage: AgentTurnUsage = {
    input_tokens: asNumber(src.input_tokens ?? src.prompt_tokens),
    output_tokens: asNumber(src.output_tokens ?? src.completion_tokens),
    cache_read_input_tokens: asNumber(src.cache_read_input_tokens ?? src.cached_input_tokens ?? src.cache_read_tokens),
    cache_creation_input_tokens: asNumber(src.cache_creation_input_tokens ?? src.cache_write_tokens),
    reasoning_tokens: asNumber(src.reasoning_tokens),
  };
  if (src.total_cost_usd != null && Number.isFinite(Number(src.total_cost_usd))) {
    usage.total_cost_usd = Number(src.total_cost_usd);
  }
  if (src.hosted_cost_usd != null && Number.isFinite(Number(src.hosted_cost_usd))) {
    usage.hosted_cost_usd = Number(src.hosted_cost_usd);
  }
  if (src.total_tokens != null && Number.isSafeInteger(src.total_tokens) && Number(src.total_tokens) >= 0) {
    usage.total_tokens = Number(src.total_tokens);
  }
  return usage;
}

function countTools(turns: AgentTurnSnapshot[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const turn of turns) {
    for (const tool of turn.tools) {
      counts[tool.kind] = (counts[tool.kind] || 0) + 1;
    }
  }
  return counts;
}

function addUsage(total: AgentTurnUsage, delta: AgentTurnUsage): AgentTurnUsage {
  const next: AgentTurnUsage = {
    input_tokens: total.input_tokens + delta.input_tokens,
    output_tokens: total.output_tokens + delta.output_tokens,
    cache_read_input_tokens: total.cache_read_input_tokens + delta.cache_read_input_tokens,
    cache_creation_input_tokens: total.cache_creation_input_tokens + delta.cache_creation_input_tokens,
    reasoning_tokens: total.reasoning_tokens + delta.reasoning_tokens,
  };
  if (total.total_cost_usd != null || delta.total_cost_usd != null) {
    next.total_cost_usd = (total.total_cost_usd ?? 0) + (delta.total_cost_usd ?? 0);
  }
  if (total.hosted_cost_usd != null || delta.hosted_cost_usd != null) {
    next.hosted_cost_usd = (total.hosted_cost_usd ?? 0) + (delta.hosted_cost_usd ?? 0);
  }
  if (total.total_tokens != null || delta.total_tokens != null) {
    next.total_tokens = (total.total_tokens ?? 0) + (delta.total_tokens ?? 0);
  }
  return next;
}

function sumTurnUsage(turns: AgentTurnSnapshot[]): AgentTurnUsage {
  return turns.reduce((total, turn) => addUsage(total, turn.usage), { ...EMPTY_USAGE });
}

function rebuild(turns: AgentTurnSnapshot[]): AgentRunTelemetry {
  return {
    num_turns: turns.length,
    usage: sumTurnUsage(turns),
    tool_counts: countTools(turns),
    turns,
    ...combinedMeasurementState(turns.map((turn) => turn.usage_state)),
  };
}

export function combinedMeasurementState(states: Array<UsageMeasurementState | undefined>): {
  usage_state?: UsageMeasurementState;
} {
  const known = states.filter((state): state is UsageMeasurementState => state !== undefined);
  if (!known.length) return {};
  if (
    known.includes("partial") ||
    (known.includes("measured") && (known.includes("unavailable") || known.includes("pending")))
  )
    return { usage_state: "partial" };
  if (known.includes("unavailable")) return { usage_state: "unavailable" };
  if (known.includes("pending")) return { usage_state: "pending" };
  return { usage_state: "measured" };
}

function upsertTurn(
  turns: AgentTurnSnapshot[],
  turn: number,
  usage?: AgentTurnUsage,
  message?: string,
): AgentTurnSnapshot[] {
  const existing = turns.find((item) => item.turn === turn);
  if (existing) {
    return turns.map((item) =>
      item.turn === turn
        ? {
            ...item,
            usage: usage ?? item.usage,
            message: message?.trim() || item.message,
          }
        : item,
    );
  }
  const next = [
    ...turns,
    {
      turn,
      usage: usage ?? { ...EMPTY_USAGE },
      tools: [],
      ...(message?.trim() ? { message: message.trim() } : {}),
    },
  ];
  return [...next].sort((a, b) => a.turn - b.turn);
}

function turnForTool(state: AgentRunTelemetry, recordTurn: number | undefined): number {
  if (recordTurn != null && recordTurn > 0) {
    return recordTurn;
  }
  return state.turns.at(-1)?.turn ?? 1;
}

function findToolIndex(tools: AgentTurnTool[], id?: string): number {
  if (id) {
    const index = tools.findIndex((tool) => tool.id === id);
    if (index >= 0) {
      return index;
    }
  }
  return tools.length - 1;
}

export function applyAgentTelemetryRecord(
  state: AgentRunTelemetry,
  record: AgentTelemetryLiveRecord,
): AgentRunTelemetry {
  if (record.type === "turn") {
    const usage = normalizeAgentTurnUsage(record.usage);
    const turns = upsertTurn(state.turns, record.turn, usage, record.message).map((turn) =>
      turn.turn === record.turn
        ? { ...turn, ...(record.usage_state ? { usage_state: record.usage_state } : {}) }
        : turn,
    );
    return rebuild(turns);
  }

  const turn = turnForTool(state, record.turn);
  const turns = upsertTurn(state.turns, turn);
  const snapshot = turns.find((item) => item.turn === turn);
  if (!snapshot) {
    return rebuild(turns);
  }

  if (record.type === "tool_start") {
    const tool: AgentTurnTool = {
      id: record.id,
      kind: record.kind?.trim() || "tool",
      text: record.text ?? record.kind ?? "tool",
      status: "running",
    };
    const nextSnapshot = { ...snapshot, tools: [...snapshot.tools, tool] };
    return rebuild(turns.map((item) => (item.turn === turn ? nextSnapshot : item)));
  }

  const index = findToolIndex(snapshot.tools, record.id);
  if (index < 0) {
    return rebuild(turns);
  }
  const current = snapshot.tools[index];
  const nextTools = snapshot.tools.map((tool, toolIndex) => {
    if (toolIndex !== index) {
      return tool;
    }
    return {
      ...current,
      status: record.status ?? current.status,
      duration_ms: record.duration_ms ?? current.duration_ms,
    };
  });
  return rebuild(turns.map((item) => (item.turn === turn ? { ...snapshot, tools: nextTools } : item)));
}

export function reduceAgentTelemetryRecords(records: AgentTelemetryLiveRecord[]): AgentRunTelemetry {
  return records.reduce(applyAgentTelemetryRecord, emptyAgentRunTelemetry());
}

export function telemetryFromFinishedResult(result: unknown): AgentRunTelemetry | null {
  if (!result || typeof result !== "object") {
    return null;
  }
  const telemetry = (result as { telemetry?: unknown }).telemetry;
  if (!telemetry || typeof telemetry !== "object") {
    return null;
  }
  const raw = telemetry as {
    usage_state?: unknown;
    num_turns?: unknown;
    usage?: unknown;
    tool_counts?: unknown;
    turns?: unknown;
  };
  if (!Array.isArray(raw.turns)) {
    return null;
  }
  const turns = raw.turns.flatMap((item): AgentTurnSnapshot[] => {
    if (!item || typeof item !== "object") {
      return [];
    }
    const turn = item as { turn?: unknown; usage?: unknown; tools?: unknown; message?: unknown; usage_state?: unknown };
    const tools = Array.isArray(turn.tools)
      ? turn.tools.flatMap((tool): AgentTurnTool[] => {
          if (!tool || typeof tool !== "object") {
            return [];
          }
          const row = tool as AgentTurnTool;
          return [
            {
              id: row.id,
              kind: String(row.kind || "tool"),
              text: String(row.text || row.kind || "tool"),
              status: row.status,
              duration_ms: row.duration_ms,
            },
          ];
        })
      : [];
    const message = typeof turn.message === "string" && turn.message.trim() ? turn.message : undefined;
    return [
      {
        ...combinedMeasurementState([usageMeasurementState(turn.usage_state)]),
        turn: asNumber(turn.turn),
        usage: normalizeAgentTurnUsage(turn.usage),
        tools,
        ...(message ? { message } : {}),
      },
    ];
  });
  const parsed = { ...rebuild(turns), ...combinedMeasurementState([usageMeasurementState(raw.usage_state)]) };
  if (raw.usage) {
    parsed.usage = normalizeAgentTurnUsage(raw.usage);
  }
  if (asNumber(raw.num_turns) > parsed.num_turns) {
    parsed.num_turns = asNumber(raw.num_turns);
  }
  return parsed;
}

export function telemetrySeriesFromFinishedResult(result: unknown): AgentPromptUsageSeries[] {
  if (!result || typeof result !== "object") {
    return [];
  }
  const telemetry = (result as { telemetry?: unknown }).telemetry;
  if (!telemetry || typeof telemetry !== "object") {
    return [];
  }
  const prompts = (telemetry as { prompts?: unknown }).prompts;
  if (Array.isArray(prompts)) {
    return prompts.flatMap((item): AgentPromptUsageSeries[] => {
      if (!item || typeof item !== "object") {
        return [];
      }
      const row = item as { name?: unknown; telemetry?: unknown };
      const seriesTelemetry = telemetryFromFinishedResult({ telemetry: row.telemetry ?? item });
      if (!seriesTelemetry || (seriesTelemetry.turns.length === 0 && !seriesTelemetry.usage_state)) {
        return [];
      }
      return [{ name: typeof row.name === "string" ? row.name : "", telemetry: seriesTelemetry }];
    });
  }
  const single = telemetryFromFinishedResult(result);
  if (!single || (single.turns.length === 0 && !single.usage_state)) {
    return [];
  }
  return [{ name: "", telemetry: single }];
}

export function telemetryFromExecutionOutputs(outputs: unknown): AgentRunTelemetry | null {
  if (!outputs || typeof outputs !== "object") {
    return null;
  }
  for (const channel of Object.values(outputs as Record<string, unknown>)) {
    const events = Array.isArray(channel) ? channel : [channel];
    for (const event of events) {
      if (!event || typeof event !== "object") {
        continue;
      }
      const row = event as { data?: { result?: unknown }; result?: unknown };
      const result = row.data?.result ?? row.result ?? row.data ?? event;
      const telemetry = telemetryFromFinishedResult(result);
      if (telemetry && (telemetry.turns.length > 0 || telemetry.usage_state)) {
        return telemetry;
      }
    }
  }
  return null;
}

export function usageDelta(current: AgentTurnUsage, previous?: AgentTurnUsage): AgentTurnUsage {
  if (!previous) {
    return { ...current };
  }
  const delta: AgentTurnUsage = {
    input_tokens: Math.max(0, current.input_tokens - previous.input_tokens),
    output_tokens: Math.max(0, current.output_tokens - previous.output_tokens),
    cache_read_input_tokens: Math.max(0, current.cache_read_input_tokens - previous.cache_read_input_tokens),
    cache_creation_input_tokens: Math.max(
      0,
      current.cache_creation_input_tokens - previous.cache_creation_input_tokens,
    ),
    reasoning_tokens: Math.max(0, current.reasoning_tokens - previous.reasoning_tokens),
  };
  if (current.total_cost_usd != null) {
    delta.total_cost_usd = Math.max(0, current.total_cost_usd - (previous.total_cost_usd ?? 0));
  }
  if (current.hosted_cost_usd != null) {
    delta.hosted_cost_usd = Math.max(0, current.hosted_cost_usd - (previous.hosted_cost_usd ?? 0));
  }
  return delta;
}

export function parseAgentTurnLiveLogText(
  text: string,
): { turn: number; usage: AgentTurnUsage; message?: string; usage_state?: UsageMeasurementState } | null {
  const trimmed = text.trim();
  if (!trimmed.startsWith("{")) {
    return null;
  }
  try {
    const rec = JSON.parse(trimmed) as {
      type?: unknown;
      turn?: unknown;
      usage?: unknown;
      message?: unknown;
      usage_state?: unknown;
    };
    if (rec.type !== "turn" || typeof rec.turn !== "number") {
      return null;
    }
    const message = typeof rec.message === "string" && rec.message.trim() ? rec.message : undefined;
    return {
      turn: rec.turn,
      usage: normalizeAgentTurnUsage(rec.usage),
      ...combinedMeasurementState([usageMeasurementState(rec.usage_state)]),
      ...(message ? { message } : {}),
    };
  } catch {
    return null;
  }
}

export function isRawAgentTurnLiveLogText(text: string): boolean {
  return parseAgentTurnLiveLogText(text) != null;
}

export function isSerializedAgentActivityLiveLogText(text: string): boolean {
  return parseAgentActivityRecordText(text) !== undefined;
}

export function isHiddenAgentLiveLogText(text: string): boolean {
  return isRawAgentTurnLiveLogText(text) || isSerializedAgentActivityLiveLogText(text);
}

export type AgentPromptUsageSeries = {
  name: string;
  telemetry: AgentRunTelemetry;
};

export type AgentPromptUsageState = {
  completed: AgentPromptUsageSeries[];
  currentName: string;
  current: AgentRunTelemetry;
  startedIndexes: number[];
};

export function emptyPromptUsageState(): AgentPromptUsageState {
  return { completed: [], currentName: "", current: emptyAgentRunTelemetry(), startedIndexes: [] };
}

export function startPromptUsageSeries(
  state: AgentPromptUsageState,
  name: string,
  index?: number,
): AgentPromptUsageState {
  if (index != null && state.startedIndexes.includes(index)) {
    return state;
  }
  const completed =
    state.current.turns.length > 0 || state.current.usage_state !== undefined
      ? [...state.completed, { name: state.currentName || name, telemetry: state.current }]
      : state.completed;
  return {
    completed,
    currentName: name,
    current: emptyAgentRunTelemetry(),
    startedIndexes: index != null ? [...state.startedIndexes, index] : state.startedIndexes,
  };
}

export function applyPromptUsageRecord(
  state: AgentPromptUsageState,
  record: AgentTelemetryLiveRecord,
): AgentPromptUsageState {
  return { ...state, current: applyAgentTelemetryRecord(state.current, record) };
}

export function promptUsageSeries(state: AgentPromptUsageState): AgentPromptUsageSeries[] {
  if (state.current.turns.length === 0 && !state.current.usage_state) {
    return state.completed;
  }
  return [...state.completed, { name: state.currentName, telemetry: state.current }];
}

export function preferLiveTelemetry(live: AgentRunTelemetry, finished: AgentRunTelemetry | null): AgentRunTelemetry {
  if (live.turns.length > 0) {
    return live;
  }
  return finished ?? live;
}
