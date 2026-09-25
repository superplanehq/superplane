import { describe, expect, it } from "bun:test";

import type { AgentPromptUsageSeries, AgentRunTelemetry } from "@/lib/agentRunTelemetry";

import { headerSpendFromUsageSeries, planningHeaderSpendToReport } from "./planningHeaderSpend";

function prompt(tokens: number, usd: number): AgentPromptUsageSeries {
  const telemetry: AgentRunTelemetry = {
    num_turns: 1,
    usage: {
      input_tokens: tokens,
      output_tokens: 0,
      cache_read_input_tokens: 0,
      cache_creation_input_tokens: 0,
      reasoning_tokens: 0,
      total_cost_usd: usd,
    },
    tool_counts: {},
    turns: [
      {
        turn: 1,
        usage: {
          input_tokens: tokens,
          output_tokens: 0,
          cache_read_input_tokens: 0,
          cache_creation_input_tokens: 0,
          reasoning_tokens: 0,
        },
        tools: [],
      },
    ],
  };
  return { name: "Prompt", telemetry };
}

describe("headerSpendFromUsageSeries", () => {
  it("rounds the summed prompt cost once", () => {
    expect(headerSpendFromUsageSeries([prompt(10, 0.006), prompt(12, 0.006)])).toEqual({
      tokens: 22,
      cents: 1,
    });
  });
});

describe("planningHeaderSpendToReport", () => {
  it("adds a follow-up run to saved planning usage", () => {
    const reported = planningHeaderSpendToReport(
      undefined,
      "exec-2",
      { tokens: 2000, cents: 20 },
      { tokens: 1000, cents: 10 },
    );
    expect(reported.spend).toEqual({ tokens: 3000, cents: 30 });
  });

  it("does not add the live log when saved usage is that same run", () => {
    const reported = planningHeaderSpendToReport(
      undefined,
      "exec-1",
      { tokens: 2100, cents: 50 },
      { tokens: 2100, cents: 45 },
    );
    expect(reported.spend).toEqual({ tokens: 2100, cents: 45 });
  });

  it("keeps earlier usage when the current run is saved later", () => {
    const first = planningHeaderSpendToReport(
      undefined,
      "exec-2",
      { tokens: 2000, cents: 20 },
      { tokens: 1000, cents: 10 },
    );
    const saved = planningHeaderSpendToReport(
      first.memory,
      "exec-2",
      { tokens: 3000, cents: 30 },
      { tokens: 1000, cents: 10 },
    );
    expect(saved.spend).toEqual({ tokens: 3000, cents: 30 });
  });

  it("does not add the live run again when the saved cost includes machine time", () => {
    const first = planningHeaderSpendToReport(
      undefined,
      "exec-2",
      { tokens: 2000, cents: 20 },
      { tokens: 1000, cents: 10 },
    );
    const saved = planningHeaderSpendToReport(
      first.memory,
      "exec-2",
      { tokens: 3000, cents: 35 },
      { tokens: 1000, cents: 10 },
    );
    expect(saved.spend).toEqual({ tokens: 3000, cents: 30 });
  });

  it("adds earlier usage that arrives after the live run starts", () => {
    const first = planningHeaderSpendToReport(
      undefined,
      "exec-2",
      { tokens: 0, cents: 0 },
      { tokens: 1000, cents: 10 },
    );
    const saved = planningHeaderSpendToReport(
      first.memory,
      "exec-2",
      { tokens: 2000, cents: 20 },
      { tokens: 1000, cents: 10 },
    );
    expect(saved.spend).toEqual({ tokens: 3000, cents: 30 });
  });
});
