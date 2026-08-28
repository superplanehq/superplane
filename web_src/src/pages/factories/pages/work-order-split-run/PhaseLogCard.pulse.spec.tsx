import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";
import type { SplitRunStreamLine } from "./splitRunMocks";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

const LONG_NOTE =
  "Now let me check factories.proto Delete rpc absence explicitly and PermissionTooltip component quickly, plus check showSuccessToast import paths.";

function runningCommandStream(status: SplitRunStreamLine["status"] = "running"): SplitRunStreamLine[] {
  return [
    line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
    line({
      id: "step-write",
      note: true,
      componentName: "Write Implementation Plan",
      componentType: "prompt",
    }),
    line({
      id: "cmd-cat",
      note: true,
      noteParentId: "step-write",
      noteDepth: 1,
      componentName: "cat /tmp/ORDER.md",
      componentType: "bash",
      status,
      detail: status === "running" ? "## Goal\nAdd a menu." : undefined,
    }),
  ];
}

describe("PhaseLogCard running pulse", () => {
  it("keeps running tool groups collapsed until the user opens them", async () => {
    const user = userEvent.setup();
    const runningStream = runningCommandStream();
    const { rerender, unmount } = render(<PhaseLogCard phase={PHASE} expanded stream={runningStream} />);

    const group = screen.getByRole("button", { name: "Ran 1 command" });
    expect(group).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();

    rerender(<PhaseLogCard phase={PHASE} expanded stream={runningStream} />);
    expect(screen.getByRole("button", { name: "Ran 1 command" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ran 1 command" }));
    expect(screen.getByText("cat /tmp/ORDER.md")).toBeInTheDocument();

    unmount();
    render(<PhaseLogCard phase={PHASE} expanded stream={runningStream} />);
    expect(screen.getByRole("button", { name: "Ran 1 command" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
  });

  it("pulses the last visible line while the automation is running", () => {
    render(<PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded stream={runningCommandStream()} />);

    const last = screen.getByRole("button", { name: "Ran 1 command" });
    expect(last).toHaveAttribute("data-last-running-line");
    expect(screen.getByTestId("split-run-automation-header-plan")).not.toHaveAttribute("data-last-running-line");
    expect(screen.getByTestId("split-run-stream-line-step-write")).not.toHaveAttribute("data-last-running-line");
  });

  it("moves the pulse to the last output line after the user opens a running command", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded stream={runningCommandStream()} />);

    await user.click(screen.getByRole("button", { name: "Ran 1 command" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Ran 1 command" })).not.toHaveAttribute("data-last-running-line");
      expect(screen.getByText("Add a menu.").closest("[data-last-running-line]")).not.toBeNull();
    });
  });

  it("pulses a trailing agent note when that is the last line", () => {
    const runningStream: SplitRunStreamLine[] = [
      ...runningCommandStream("passed"),
      line({
        id: "cmd-note",
        note: true,
        noteParentId: "step-write",
        noteDepth: 1,
        componentName: LONG_NOTE,
        componentType: "note",
      }),
    ];

    render(<PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded stream={runningStream} />);

    expect(screen.getByText(LONG_NOTE).closest("[data-last-running-line]")).not.toBeNull();
    expect(screen.getByRole("button", { name: "Ran 1 command" })).not.toHaveAttribute("data-last-running-line");
  });

  it("does not pulse a later pending node", () => {
    render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running" }}
        expanded
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            componentType: "Run Claude Code",
            status: "running",
          }),
          line({
            id: "run-tests",
            nodeId: "run-tests",
            componentName: "Run tests",
            status: "pending",
          }),
        ]}
      />,
    );

    expect(screen.getByTestId("split-run-stream-line-planner-agent")).toHaveAttribute("data-last-running-line");
    expect(screen.getByTestId("split-run-stream-line-run-tests")).not.toHaveAttribute("data-last-running-line");
  });

  it("does not pulse a line after the automation finishes", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
        ]}
      />,
    );

    expect(document.querySelector("[data-last-running-line]")).toBeNull();
  });
});
