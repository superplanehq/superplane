import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { reduceAgentTelemetryRecords } from "@/lib/agentRunTelemetry";
import { AgentRunUsageChart } from "./AgentRunUsageChart";

const telemetry = reduceAgentTelemetryRecords([
  { type: "turn", turn: 1, usage: { input_tokens: 100, output_tokens: 20 }, message: "I will generate protobufs." },
  { type: "tool_start", turn: 1, kind: "bash", text: "make pb.gen" },
  { type: "tool_start", turn: 1, kind: "bash", text: "git status" },
  {
    type: "turn",
    turn: 2,
    usage: { input_tokens: 80, output_tokens: 10 },
    message: "Next I will read the generated file.",
  },
  { type: "tool_start", turn: 2, kind: "read", text: "pkg/protos/canvases.pb.go" },
]);

const longBashCommand =
  "find /Users/lucaspin/Projects/superplanehq/superplane/pkg/components/runner -type f \\( -name '*.go' -o -name '*.js' \\) -print0 | xargs -0 rg --files-with-matches 'EmitAndContinue' | head -n 40";

const longLabelTelemetry = reduceAgentTelemetryRecords([
  { type: "turn", turn: 1, usage: { input_tokens: 100, output_tokens: 20 } },
  { type: "tool_start", turn: 1, kind: "bash", text: "git status" },
  { type: "turn", turn: 2, usage: { input_tokens: 80, output_tokens: 10 } },
  { type: "tool_start", turn: 2, kind: "bash", text: longBashCommand },
]);

describe("AgentRunUsageChart", () => {
  it("shows an empty state when there are no turns", () => {
    render(<AgentRunUsageChart telemetry={reduceAgentTelemetryRecords([])} />);
    expect(screen.getByText("No usage data yet.")).toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("marks the chart as live while the run continues", () => {
    render(<AgentRunUsageChart telemetry={telemetry} live />);
    expect(screen.getByRole("status")).toHaveTextContent("Live. The run is not finished. New turns will appear here.");
    expect(screen.getByTestId("usage-live-slot")).toBeInTheDocument();
  });

  it("does not mark a finished run as live", () => {
    render(<AgentRunUsageChart telemetry={telemetry} />);
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.queryByTestId("usage-live-slot")).not.toBeInTheDocument();
  });

  it("shows a live empty state when no turns have arrived", () => {
    render(<AgentRunUsageChart telemetry={reduceAgentTelemetryRecords([])} live />);
    expect(screen.getByText("No usage data yet.")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Live. The run is not finished. New turns will appear here.");
  });

  it("lists the tools for the hovered turn", async () => {
    const user = userEvent.setup();
    render(<AgentRunUsageChart telemetry={telemetry} />);
    expect(screen.getByText("2 turns · 3 tool calls · 180 input · 30 output")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show input tokens only" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show output tokens only" })).toBeInTheDocument();
    expect(screen.getByText("Hover a turn to preview it. Click a turn to read the agent message.")).toBeInTheDocument();
    await user.hover(screen.getByRole("button", { name: "Turn 1: 100 input tokens, 20 output tokens" }));
    expect(screen.getByText("Turn 1 · 100 input · 20 output · 2 tool calls")).toBeInTheDocument();
    expect(screen.getByText("I will generate protobufs.")).toBeInTheDocument();
    expect(screen.getByText("make pb.gen")).toBeInTheDocument();
    expect(screen.getByText("git status")).toBeInTheDocument();
  });

  it("lets earlier turns stay hoverable when later turns are present", async () => {
    const user = userEvent.setup();
    const manyTurns = reduceAgentTelemetryRecords(
      Array.from({ length: 12 }, (_, index) => ({
        type: "turn" as const,
        turn: index + 1,
        usage: { input_tokens: 40 + index, output_tokens: 8 },
        message: `Turn ${index + 1} message`,
      })),
    );
    render(<AgentRunUsageChart telemetry={manyTurns} />);
    await user.hover(screen.getByRole("button", { name: "Turn 1: 40 input tokens, 8 output tokens" }));
    expect(screen.getByText("Turn 1 message")).toBeInTheDocument();
    await user.hover(screen.getByRole("button", { name: "Turn 12: 51 input tokens, 8 output tokens" }));
    expect(screen.getByText("Turn 12 message")).toBeInTheDocument();
    expect(screen.queryByText("Turn 1 message")).not.toBeInTheDocument();
  });

  it("keeps the tool list after a click when the pointer leaves the column", async () => {
    const user = userEvent.setup();
    render(<AgentRunUsageChart telemetry={telemetry} />);
    await user.click(screen.getByRole("button", { name: "Turn 1: 100 input tokens, 20 output tokens" }));
    await user.unhover(screen.getByRole("button", { name: "Turn 1: 100 input tokens, 20 output tokens" }));
    expect(screen.getByText("make pb.gen")).toBeInTheDocument();
    expect(screen.getByText("git status")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Turn 2: 80 input tokens, 10 output tokens" }));
    expect(screen.getByText("pkg/protos/canvases.pb.go")).toBeInTheDocument();
    expect(screen.queryByText("make pb.gen")).not.toBeInTheDocument();
  });

  it("keeps a reserved details pane before hover so the dialog size stays stable", () => {
    render(<AgentRunUsageChart telemetry={telemetry} />);
    expect(screen.getByTestId("agent-run-turn-details")).toHaveClass("h-48");
    expect(screen.getByText("Hover a turn to preview it. Click a turn to read the agent message.")).toBeInTheDocument();
  });

  it("separates new tokens from cached context on a turn", async () => {
    const user = userEvent.setup();
    const cached = reduceAgentTelemetryRecords([
      {
        type: "turn",
        turn: 1,
        usage: { input_tokens: 2, output_tokens: 2, cache_read_input_tokens: 70247, cache_creation_input_tokens: 491 },
      },
      { type: "tool_start", turn: 1, kind: "bash", text: "git remote -v" },
    ]);
    render(<AgentRunUsageChart telemetry={cached} />);
    expect(screen.getByText("1 turn · 1 tool call · 493 input · 2 output · 70.2k cached")).toBeInTheDocument();
    await user.hover(screen.getByRole("button", { name: "Turn 1: 493 input tokens, 2 output tokens" }));
    expect(screen.getByText("Turn 1 · 493 input · 2 output · 70.2k cached · 1 tool call")).toBeInTheDocument();
    expect(
      screen.getByText("Cached tokens come from earlier turns. Tool calls do not create them."),
    ).toBeInTheDocument();
    expect(screen.getByText("git remote -v")).toBeInTheDocument();
  });

  it("collapses a long tool label until the user expands it", async () => {
    const user = userEvent.setup();
    render(<AgentRunUsageChart telemetry={longLabelTelemetry} />);
    await user.click(screen.getByRole("button", { name: "Turn 2: 80 input tokens, 10 output tokens" }));
    expect(screen.getByText(longBashCommand)).toHaveClass("line-clamp-2");
    expect(screen.getByText(longBashCommand)).not.toHaveClass("whitespace-pre-wrap");
    await user.click(screen.getByRole("button", { name: "Expand tool call" }));
    expect(screen.getByText(longBashCommand)).toHaveClass("whitespace-pre-wrap");
    expect(screen.getByText(longBashCommand)).not.toHaveClass("line-clamp-2");
    await user.click(screen.getByRole("button", { name: "Collapse tool call" }));
    expect(screen.getByText(longBashCommand)).toHaveClass("line-clamp-2");
  });

  it("shows only input bars when Input is selected", async () => {
    const user = userEvent.setup();
    render(<AgentRunUsageChart telemetry={telemetry} />);
    expect(screen.getByRole("img", { name: "Input and output tokens by turn" })).toBeInTheDocument();
    expect(screen.getAllByTestId("usage-bar-input")).toHaveLength(2);
    expect(screen.getAllByTestId("usage-bar-output")).toHaveLength(2);
    await user.click(screen.getByRole("button", { name: "Show input tokens only" }));
    expect(screen.getByRole("button", { name: "Show input and output tokens" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByRole("img", { name: "Input tokens by turn" })).toBeInTheDocument();
    expect(screen.getAllByTestId("usage-bar-input")).toHaveLength(2);
    expect(screen.queryByTestId("usage-bar-output")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show input and output tokens" }));
    expect(screen.getAllByTestId("usage-bar-output")).toHaveLength(2);
  });

  it("shows only output bars when Output is selected", async () => {
    const user = userEvent.setup();
    render(<AgentRunUsageChart telemetry={telemetry} />);
    await user.click(screen.getByRole("button", { name: "Show output tokens only" }));
    expect(screen.getByRole("img", { name: "Output tokens by turn" })).toBeInTheDocument();
    expect(screen.queryByTestId("usage-bar-input")).not.toBeInTheDocument();
    expect(screen.getAllByTestId("usage-bar-output")).toHaveLength(2);
  });
});
