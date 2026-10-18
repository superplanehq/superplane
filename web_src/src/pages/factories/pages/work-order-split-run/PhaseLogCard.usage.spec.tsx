import { render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

function usageTelemetry(inputTokens: number, tools: Array<{ kind: string; text: string }>) {
  return {
    num_turns: 1,
    usage: {
      input_tokens: inputTokens,
      output_tokens: 0,
      cache_read_input_tokens: 0,
      cache_creation_input_tokens: 0,
      reasoning_tokens: 0,
    },
    tool_counts: Object.fromEntries(tools.map((tool) => [tool.kind, 1])),
    turns: [
      {
        turn: 1,
        usage: {
          input_tokens: inputTokens,
          output_tokens: 0,
          cache_read_input_tokens: 0,
          cache_creation_input_tokens: 0,
          reasoning_tokens: 0,
        },
        tools,
      },
    ],
  };
}

describe("PhaseLogCard usage", () => {
  function renderCard(ui: ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it.each([
    { name: "hosted", sources: ["hosted_cost_usd", "hosted_cost_usd"], dollars: "$0.01" },
    { name: "BYOK", sources: ["total_cost_usd", "total_cost_usd"], dollars: "$0.02" },
    { name: "mixed", sources: ["hosted_cost_usd", "total_cost_usd"], dollars: "$0.01" },
  ])("retains fractional cents across $name prompts", async ({ sources, dollars }) => {
    const series = sources.map((source, index) => {
      const telemetry = usageTelemetry(210, []);
      return {
        name: `Prompt ${index + 1}`,
        telemetry: { ...telemetry, usage: { ...telemetry.usage, [source]: 0.006 } },
      };
    });
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      usageSeries: series,
    });
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", costCents: "0", totalTokens: "0", duration: "12s" }}
        expanded={false}
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "agent",
            componentName: "Agent",
            component: "runnerClaudeCode",
            executionId: "exec-1",
            status: "running",
          }),
        ]}
      />,
    );
    await waitFor(() => expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent(dollars));
  });

  it("shows live token counts as plain text before the phase records spend", async () => {
    const live = usageTelemetry(210, [{ kind: "bash", text: "git status" }]);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", costCents: "0", totalTokens: "0", duration: "12s" }}
        expanded={false}
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent",
            component: "runnerClaudeCode",
            executionId: "exec-1",
            status: "running",
          }),
        ]}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("210"));
    expect(screen.queryByRole("button", { name: "Show usage" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("shows the reported runner model next to spend", () => {
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210", model: "openai/gpt-4.1" }}
        expanded={false}
      />,
    );

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("gpt-4.1");
    expect(screen.getByTestId("split-run-phase-duration-plan")).not.toHaveTextContent("claude-sonnet-4-6");
  });
});
