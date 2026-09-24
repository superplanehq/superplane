import type { AgentRunTelemetry, AgentTurnTool, AgentTurnUsage } from "./agentRunTelemetry";

export type AgentChartPoint = {
  turn: number;
  tokens: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  tools: number;
  turnTools: AgentTurnTool[];
  message: string;
  inputBar: number;
  outputBar: number;
};

export function inputTokenTotal(usage: AgentTurnUsage): number {
  return usage.input_tokens + usage.cache_creation_input_tokens;
}

export function outputTokenTotal(usage: AgentTurnUsage): number {
  return usage.output_tokens + usage.reasoning_tokens;
}

export function chartPointsForTelemetry(telemetry: AgentRunTelemetry): AgentChartPoint[] {
  return telemetry.turns.map((snapshot) => {
    const usage = snapshot.usage;
    const inputBar = inputTokenTotal(usage);
    const outputBar = outputTokenTotal(usage);
    return {
      turn: snapshot.turn,
      tokens: inputBar + outputBar,
      input_tokens: usage.input_tokens,
      output_tokens: usage.output_tokens,
      cache_read_tokens: usage.cache_read_input_tokens,
      tools: snapshot.tools.length,
      turnTools: snapshot.tools,
      message: snapshot.message ?? "",
      inputBar,
      outputBar,
    };
  });
}

export function agentRunToolCallCount(telemetry: AgentRunTelemetry): number {
  return Object.values(telemetry.tool_counts).reduce((sum, count) => sum + count, 0);
}

export function agentRunNewTokenCount(telemetry: AgentRunTelemetry): number {
  return agentRunInputTokenCount(telemetry) + agentRunOutputTokenCount(telemetry);
}

export function agentRunInputTokenCount(telemetry: AgentRunTelemetry): number {
  return telemetry.turns.reduce((sum, turn) => sum + inputTokenTotal(turn.usage), 0);
}

export function agentRunOutputTokenCount(telemetry: AgentRunTelemetry): number {
  return telemetry.turns.reduce((sum, turn) => sum + outputTokenTotal(turn.usage), 0);
}

export function agentRunCacheReadCount(telemetry: AgentRunTelemetry): number {
  return telemetry.turns.reduce((sum, turn) => sum + turn.usage.cache_read_input_tokens, 0);
}
