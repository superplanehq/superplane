import type { AgentRunTelemetry, AgentTurnUsage } from "./agentRunTelemetry";

export function inputTokenTotal(usage: AgentTurnUsage): number {
  return usage.input_tokens + usage.cache_creation_input_tokens;
}

export function outputTokenTotal(usage: AgentTurnUsage): number {
  return usage.output_tokens + usage.reasoning_tokens;
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
