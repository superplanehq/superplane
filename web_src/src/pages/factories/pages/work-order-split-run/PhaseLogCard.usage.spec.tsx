import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

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

  it("shows live token counts before the phase records spend", async () => {
    const user = userEvent.setup();
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

    const showUsage = await screen.findByRole("button", { name: "Show usage" });
    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("210");
    await user.click(showUsage);
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 210 input")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Live. The run is not finished. New turns will appear here.");
  });

  it("opens a usage dialog from the cost and token text", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: usageTelemetry(210, [
        { kind: "bash", text: "git status" },
        { kind: "read", text: "README.md" },
      ]),
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210" }}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByText("1 turn · 2 tool calls · 210 input")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Agent - Plan for GH Issue" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("heading", { name: "Plan usage" })).not.toBeInTheDocument();
  });

  it("shows a separate chart for each agent", async () => {
    const user = userEvent.setup();
    const implementUsage = {
      ...idleLiveLogStream(vi.fn()),
      telemetry: usageTelemetry(180, [{ kind: "bash", text: "make test" }]),
    };
    const pullRequestUsage = {
      ...idleLiveLogStream(vi.fn()),
      telemetry: usageTelemetry(40, [{ kind: "read", text: "PR.md" }]),
    };
    useLiveLogStreamMock.mockImplementation((executionId: string) =>
      executionId === "exec-pr" ? pullRequestUsage : implementUsage,
    );

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, name: "Implement", costCents: "80", totalTokens: "220" }}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "impl-agent",
            nodeId: "impl-agent",
            componentName: "Implement the change",
            component: "runnerClaudeCode",
            executionId: "exec-impl",
          }),
          line({
            id: "pr-agent",
            nodeId: "pr-agent",
            componentName: "Write the pull request",
            component: "runnerClaudeCode",
            executionId: "exec-pr",
          }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Implement usage" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Implement the change" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Write the pull request" })).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 180 input")).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 40 input")).toBeInTheDocument();
  });

  it("shows a separate chart for each prompt on one agent", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      usageSeries: [
        { name: "Implementation", telemetry: usageTelemetry(180, [{ kind: "bash", text: "make test" }]) },
        {
          name: "Generate PR title and description",
          telemetry: usageTelemetry(40, [{ kind: "bash", text: "git log main..feature" }]),
        },
      ],
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, name: "Implement", costCents: "80", totalTokens: "220" }}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "impl-agent",
            nodeId: "impl-agent",
            componentName: "Implement the change",
            component: "runnerClaudeCode",
            executionId: "exec-impl",
          }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Implement usage" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Implementation" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Generate PR title and description" })).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 180 input")).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 40 input")).toBeInTheDocument();
  });

  it("loads the chart from a collapsed run without opening the log", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: usageTelemetry(210, [{ kind: "bash", text: "git status" }]),
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210" }}
        expanded={false}
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByText("1 turn · 1 tool call · 210 input")).toBeInTheDocument();
    expect(screen.queryByText("No usage data yet.")).not.toBeInTheDocument();
  });

  it("shows a loading state while usage is still fetching on a collapsed run", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      isStreaming: true,
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210" }}
        expanded={false}
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading usage...");
    expect(screen.queryByText("No usage data yet.")).not.toBeInTheDocument();
  });

  it("keeps the loading state while a running run reconnects without turns", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      isStreaming: false,
    });

    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", costCents: "0", totalTokens: "210", duration: "12s" }}
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

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading usage...");
    expect(screen.queryByText("No usage data yet.")).not.toBeInTheDocument();
  });

  it("shows a loading state while the run stream is still loading", async () => {
    const user = userEvent.setup();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210" }}
        expanded={false}
        streamLoading
        organizationId="org-1"
        canvasId="canvas-1"
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(await screen.findByRole("heading", { name: "Plan usage" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading usage...");
    expect(screen.queryByText("No usage data yet.")).not.toBeInTheDocument();
  });

  it("notifies the parent when the usage dialog opens from a collapsed run", async () => {
    const user = userEvent.setup();
    const onUsageOpenChange = vi.fn();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "210" }}
        expanded={false}
        onUsageOpenChange={onUsageOpenChange}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Show usage" }));
    expect(onUsageOpenChange).toHaveBeenCalledWith(true);
  });
});
