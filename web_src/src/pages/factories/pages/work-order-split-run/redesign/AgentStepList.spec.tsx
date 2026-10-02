import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { logStatusTimeLabel } from "../logStatusTime";
import type { AgentActivity } from "../agentActivity";
import { AgentStepMarkers } from "./AgentStepList";
import type { AgentStep, AutomationStage } from "./automationsViewModel";
import { formatClock } from "./redesignFormat";

const LONG_PROMPT =
  "You refine draft tasks so a coding agent can build them in one run. You read the task and the repository.";

function stageWithStep(step: AgentStep): AutomationStage {
  return stageWithSteps([step]);
}

function stageWithSteps(steps: AgentStep[]): AutomationStage {
  return {
    id: "implement",
    name: "Implementation",
    componentName: "Implementation",
    status: "passed",
    statusLabel: "Passed",
    duration: "1m",
    checks: [],
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: steps,
    steps,
    rawLog: "",
  };
}

describe("AgentStepMarkers", () => {
  it("keeps a long prompt subtitle on one truncated line", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "refine",
          title: "Refine task",
          type: "prompt",
          status: "passed",
          summary: "",
          toolCount: 0,
          output: LONG_PROMPT,
          events: [],
        })}
      />,
    );

    const title = screen.getByText("Refine task");
    const subtitle = screen.getByText(LONG_PROMPT);
    expect(title).toHaveClass("truncate");
    expect(subtitle).toHaveClass("truncate");
    expect(subtitle).not.toHaveClass("break-words");
  });

  it("keeps Planning next step under a long live transcript", () => {
    const liveActivity: AgentActivity = {
      id: "live-1",
      provider: "runner",
      status: "running",
      sequence: 4,
      truncated: false,
      items: [
        {
          type: "content",
          id: "note-1",
          kind: "assistant",
          text: "Bun is not installed on the host.",
          status: "passed",
          truncated: false,
        },
        {
          type: "tool",
          id: "tool-1",
          kind: "read",
          name: "read",
          input: "README.md",
          output: "",
          outputStreams: [],
          status: "passed",
          truncated: false,
        },
      ],
    };

    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 1,
          events: [],
        })}
        liveActive
        liveActivity={liveActivity}
      />,
    );

    const status = screen.getByRole("status", { name: "Planning next step…" });
    const log = screen.getByTestId("redesign-step-log-implementation");
    expect(status).toBeInTheDocument();
    expect(log).not.toContainElement(status);
  });

  it("shows Thinking when the console hides an empty thought row", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 0,
          events: [],
        })}
        liveActive
        liveActivity={{
          id: "live-1",
          provider: "runner",
          status: "running",
          sequence: 2,
          truncated: false,
          items: [
            {
              type: "content",
              id: "thought-1",
              kind: "reasoning",
              text: "",
              status: "running",
              truncated: false,
            },
          ],
        }}
      />,
    );

    expect(screen.getByRole("status", { name: "Thinking" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveAccessibleName("Thinking");
  });

  it("names a streaming assistant reply Writing response", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 0,
          events: [],
        })}
        liveActive
        liveActivity={{
          id: "live-1",
          provider: "runner",
          status: "running",
          sequence: 2,
          truncated: false,
          items: [
            {
              type: "content",
              id: "note-1",
              kind: "assistant",
              text: "Bun is not installed on the host.",
              status: "running",
              truncated: false,
            },
          ],
        }}
      />,
    );

    expect(screen.getByRole("status", { name: "Writing response…" })).toBeInTheDocument();
  });

  it("opens a named SSH step to the command and output", async () => {
    const user = userEvent.setup();
    const script = [
      'git config --global user.email "superplaneagent@superplane.com"',
      'git config --global user.name "SuperPlane Agent"',
    ].join("\n");

    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "git-user",
            title: "Set Up Git User",
            commandScript: script,
            commandStdout: "configured",
            events: [
              {
                kind: "tools",
                id: "tools-1",
                label: "1 tool call",
                tools: [{ id: "git-1", type: "bash", name: "git status", status: "passed" }],
              },
            ],
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Set Up Git User/ }));

    expect(screen.queryByRole("button", { name: "Inspected Git" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Researched 1 source" })).not.toBeInTheDocument();
    expect(screen.getByText("$")).toBeInTheDocument();
    expect(screen.getByTestId("agent-tool-git-user-command").textContent).toBe(script);
    expect(screen.getByText("configured")).toBeInTheDocument();
  });

  it("does not nest an SSH command under itself", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "echo",
            title: "echo hi",
            commandScript: "echo hi",
            commandStdout: "hi",
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /echo hi/ }));

    expect(screen.getByText("hi")).toBeInTheDocument();
    expect(screen.queryByText("$")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-tool-echo-command")).not.toBeInTheDocument();
  });

  it("keeps a one-line SSH command with no output in line with chevron rows", () => {
    render(
      <AgentStepMarkers
        stage={stageWithSteps([
          bashStep({
            id: "setup",
            title: "Set Up Git User",
            commandScript: "git config user.email a\ngit config user.name b",
          }),
          bashStep({ id: "cd", title: "cd repo", commandScript: "cd repo" }),
        ])}
      />,
    );

    const named = screen.getByText("Set Up Git User").closest("[data-slot='marker']");
    const plain = screen.getByText("cd repo").closest("[data-slot='marker']");
    const namedIcon = named?.querySelector("[data-slot='marker-icon']");
    const plainIcon = plain?.querySelector("[data-slot='marker-icon']");

    expect(namedIcon?.querySelector("svg")).not.toBeNull();
    expect(plainIcon?.querySelector("svg")).toBeNull();
    expect(namedIcon).toHaveClass("size-4");
    expect(plainIcon).toHaveClass("size-4");
    expect(plain?.querySelector("[data-slot='marker-content'] svg")?.getAttribute("class")).toContain("opacity-0");
    expect(screen.queryByRole("button", { name: /cd repo/ })).not.toBeInTheDocument();
  });

  it("hides the prompt icon until the pointer is over the row", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "refine",
          title: "Refine task",
          type: "prompt",
          status: "passed",
          summary: "",
          toolCount: 0,
          output: "Ready for a coding agent.",
          events: [],
        })}
      />,
    );

    const icon = screen
      .getByText("Refine task")
      .closest("[data-slot='marker']")
      ?.querySelector("[data-slot='marker-content'] svg");
    expect(icon?.getAttribute("class")).toContain("opacity-0");
    expect(icon?.getAttribute("class")).toContain("group-hover/marker:opacity-100");
  });

  it("shows a red mark on a failed SSH command and opens to the error", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "fail",
            title: "false",
            status: "failed",
            commandScript: "false",
            commandStdout: "permission denied",
          }),
        )}
      />,
    );

    expect(screen.getByLabelText("failed")).toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /false/ }));

    expect(screen.getByText("permission denied")).toHaveClass("text-destructive");
  });

  it("opens a failed SSH command with no output to Command failed.", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "fail-empty",
            title: "false",
            status: "failed",
            commandScript: "false",
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /false/ }));

    expect(screen.getByText("Command failed.")).toBeInTheDocument();
  });

  it("shows SSH start time and duration on hover", () => {
    const startedAtMs = Date.parse("2026-09-30T00:59:00.000Z");
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "timed",
            title: "echo hi",
            commandScript: "echo hi",
            duration: "2s",
            startedAtMs,
          }),
        )}
      />,
    );

    const meta = screen.getByText(`${formatClock(new Date(startedAtMs).toISOString())} · ${logStatusTimeLabel("2s")}`);
    expect(meta).toHaveClass("opacity-0");
    expect(meta.className).toContain("group-hover/marker:opacity-100");
  });

  it("shows only the duration when an SSH row has no start time", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "duration-only",
            title: "echo hi",
            commandScript: "echo hi",
            duration: "2s",
          }),
        )}
      />,
    );

    const meta = screen.getByText(logStatusTimeLabel("2s"));
    expect(meta).toHaveTextContent(logStatusTimeLabel("2s"));
    expect(meta).not.toHaveTextContent("·");
  });

  it("shows nothing when an SSH row has no start time and no duration", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep(bashStep({ id: "untimed", title: "echo hi", commandScript: "echo hi" }))}
      />,
    );

    expect(screen.queryByText("·")).not.toBeInTheDocument();
    expect(screen.queryByText(logStatusTimeLabel("2s"))).not.toBeInTheDocument();
  });

  it("keeps live notes open while an SSH command runs", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "setup",
            title: "echo hi",
            status: "running",
            commandScript: "echo hi",
          }),
        )}
        liveActive
        liveActivity={{
          id: "live-1",
          provider: "runner",
          status: "running",
          sequence: 1,
          truncated: false,
          items: [
            {
              type: "content",
              id: "note-1",
              kind: "assistant",
              text: "Waiting for the command.",
              status: "passed",
              truncated: false,
            },
          ],
        }}
      />,
    );

    expect(screen.getByText("Waiting for the command.")).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Planning next step…" })).toBeInTheDocument();
  });

  it("shows Starting agent while a running SSH command has no output", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "setup",
            title: "echo hi",
            status: "running",
            commandScript: "echo hi",
          }),
        )}
        liveActive
      />,
    );

    expect(screen.getByRole("status", { name: "Starting agent…" })).toBeInTheDocument();
  });

  it("keeps captured SSH output separate when the command is unknown", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "clone",
            title: "Clone Repo",
            output: "Cloning into 'repo'...",
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Clone Repo/ }));

    expect(screen.getByText("Cloning into 'repo'...")).toBeInTheDocument();
    expect(screen.queryByText("$")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-tool-clone-command")).not.toBeInTheDocument();
  });

  it("opens captured SSH output that matches the row title", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "echo",
            title: "echo hi",
            output: "echo hi",
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /echo hi/ }));

    expect(screen.getAllByText("echo hi")).toHaveLength(2);
    expect(screen.queryByText("$")).not.toBeInTheDocument();
  });

  it("marks a failed named SSH command as failed", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep(
          bashStep({
            id: "git-user",
            title: "Set Up Git User",
            status: "failed",
            commandScript: "git config user.email a\ngit config user.name b",
            commandStdout: "permission denied",
          }),
        )}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Set Up Git User/ }));

    expect(screen.getByTestId("agent-tool-git-user-command")).toHaveAttribute("data-status", "failed");
    expect(screen.getByText("permission denied")).toHaveClass("text-destructive");
  });

  it("keeps an agent summary on a prompt row", async () => {
    const user = userEvent.setup();
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "passed",
          summary: "",
          toolCount: 1,
          events: [
            {
              kind: "tools",
              id: "tools-1",
              label: "1 tool call",
              tools: [{ id: "read-1", type: "read", name: "README.md", status: "passed" }],
            },
          ],
        })}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Implementation/ }));

    expect(screen.getByRole("button", { name: "Explored 1 file" })).toBeInTheDocument();
  });
});

function bashStep(overrides: Partial<AgentStep> & Pick<AgentStep, "id" | "title">): AgentStep {
  return {
    type: "bash",
    status: "passed",
    summary: "",
    toolCount: 0,
    events: [],
    ...overrides,
  };
}
