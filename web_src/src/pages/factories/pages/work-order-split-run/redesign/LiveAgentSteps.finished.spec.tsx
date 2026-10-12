import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { idleStream, implementPhase, implementStage, RUNNING_RUNNER } from "./LiveAgentSteps.testHelpers";

const FAILED_RUN_HREF =
  "/demo/workspaces/newwo/automations/a455baf7-cee0-42ed-bb87-ef8849a24ff2?run=b3bac36f-7a97-488c-9a12-c11e5bdef8e6&from=task&orderNumber=78";

function bashSection(status: "running" | "passed" | "failed") {
  return {
    index: 1,
    text: "Run script",
    kind: "bash",
    preview: "bash script.sh",
    lines: ["ok"],
    events: [],
    status,
    duration_ms: status === "running" ? null : 20,
    started_at: 1,
    collapsed: status !== "running",
  };
}

vi.mock("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas: vi.fn() }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({ useLiveLogStream: vi.fn() }));

describe("LiveAgentSteps finished runs", () => {
  beforeEach(() => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [RUNNING_RUNNER],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream());
  });

  it("shows steps from a finished runner while another runner still loads", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [
        { ...RUNNING_RUNNER, id: "other-runner", executionId: "exec-other", nodeId: "node-other", status: "passed" },
        { ...RUNNING_RUNNER, status: "passed" as const },
      ],
    });
    vi.mocked(useLiveLogStream).mockImplementation((executionId: string) => {
      if (executionId === "exec-other") {
        return idleStream({
          isStreaming: false,
          isLoading: false,
          sections: [
            {
              index: 1,
              text: "Clone Repo",
              kind: "bash",
              preview: "git clone",
              lines: [],
              events: [],
              status: "passed",
              duration_ms: 20,
              started_at: 1,
              collapsed: true,
            },
          ],
        });
      }
      return idleStream({ isStreaming: true, isLoading: true });
    });
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("Clone Repo")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("shows finished steps while the log stream stays open", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        isStreaming: true,
        isLoading: false,
        sections: [
          {
            index: 1,
            text: "Clone Repo",
            kind: "bash",
            preview: "git clone",
            lines: [],
            events: [],
            status: "passed",
            duration_ms: 20,
            started_at: 1,
            collapsed: true,
          },
          {
            index: 2,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [{ kind: "note", text: "I will download the bun zip and extract it." }],
            status: "passed",
            duration_ms: 40,
            started_at: 2,
            collapsed: true,
          },
        ],
      }),
    );
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Implementation")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("does not keep the log skeleton on an idle open stream with no transcript", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: true, isLoading: false }));
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("No steps for this run.")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
  });

  it("shows the log skeleton while a finished run still fetches its transcript", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: true, isLoading: true }));
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(screen.getByRole("status", { name: "Waiting for logs" })).toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("shows the log skeleton while the finished run canvas is still loading", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: true,
      canvas: undefined,
      rootEventId: undefined,
      stream: [],
    });
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByRole("status", { name: "Waiting for logs" })).toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("shows a dedicated error when a non-runner node failed and no agent ran", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [
        {
          id: "on-run",
          nodeId: "on-run",
          at: "09:00:00",
          componentName: "On run",
          status: "passed",
          action: "triggered",
          kind: "trigger",
        },
        {
          id: "broken-node",
          nodeId: "broken-node",
          at: "09:00:01",
          componentName: "Broken node",
          status: "failed",
          action: "failed",
          kind: "action",
          detail: "unknown name asdad (1:1)\n | asdad\n | ^",
        },
      ],
    });
    const stage = { ...implementStage(), status: "failed" as const, agentSteps: [] };
    const phase = { ...implementPhase(), status: "failed" as const };

    render(
      <MemoryRouter>
        <LiveAgentSteps
          stage={stage}
          phase={phase}
          organizationId="org-1"
          emptyNote="No steps for this run."
          runHref={FAILED_RUN_HREF}
        />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Broken node")).toBeInTheDocument();
    expect(screen.getByText(/unknown name asdad/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Debug" })).toHaveAttribute("href", FAILED_RUN_HREF);
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("shows the dedicated error below agent steps when a non-runner also failed", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [
        {
          id: "fail-if",
          nodeId: "fail-if",
          at: "09:00:01",
          componentName: "Fail intentionally",
          status: "failed",
          action: "failed",
          kind: "action",
          detail: "unknown name asdad (1:1)\n | asdad\n | ^",
        },
        { ...RUNNING_RUNNER, status: "passed" as const },
      ],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        isStreaming: false,
        isLoading: false,
        sections: [
          {
            index: 1,
            text: "Clone Repo",
            kind: "bash",
            preview: "git clone",
            lines: [],
            events: [],
            status: "passed",
            duration_ms: 20,
            started_at: 1,
            collapsed: true,
          },
        ],
      }),
    );
    const stage = { ...implementStage(), status: "failed" as const, agentSteps: [] };
    const phase = { ...implementPhase(), status: "failed" as const };

    render(
      <MemoryRouter>
        <LiveAgentSteps
          stage={stage}
          phase={phase}
          organizationId="org-1"
          emptyNote="No steps for this run."
          runHref={FAILED_RUN_HREF}
        />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Fail intentionally")).toBeInTheDocument();
    expect(screen.getByText(/unknown name asdad/)).toBeInTheDocument();
    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Debug" })).toHaveAttribute("href", FAILED_RUN_HREF);
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("stops a stored running command on a failed task without marking it failed", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "failed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false, isLoading: false, sections: [] }));
    const stage = {
      ...implementStage(),
      status: "failed" as const,
      agentSteps: [
        {
          id: "command",
          title: "Run script",
          type: "bash" as const,
          status: "running" as const,
          summary: "",
          toolCount: 0,
          output: "bash script.sh",
          events: [],
        },
      ],
    };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={{ ...implementPhase(), status: "failed" as const }}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    const command = await screen.findByRole("button", { name: /Run script/ });
    expect(command).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
    expect(screen.queryByText("Canceled")).not.toBeInTheDocument();

    fireEvent.click(command);

    expect(screen.queryByLabelText("failed")).not.toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
  });

  it("does not mark a successful command failed when a later step fails", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: true, sections: [bashSection("running")] }));
    const stage = { ...implementStage(), status: "failed" as const, agentSteps: [] };
    const phase = { ...implementPhase(), status: "failed" as const };
    const view = render(
      <LiveAgentSteps
        stage={stage}
        phase={phase}
        organizationId="org-1"
        expandSteps
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("Run script")).toBeInTheDocument();
    expect(screen.getByText("bash script.sh")).toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("failed")).not.toBeInTheDocument();

    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false, sections: [bashSection("passed")] }));
    view.rerender(
      <LiveAgentSteps
        stage={stage}
        phase={phase}
        organizationId="org-1"
        expandSteps
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("<1s")).toBeInTheDocument();
    expect(screen.getByText("bash script.sh")).toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("failed")).not.toBeInTheDocument();
  });

  it("marks a command failed when that command failed", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false, sections: [bashSection("failed")] }));
    const stage = { ...implementStage(), status: "failed" as const, agentSteps: [] };
    const phase = { ...implementPhase(), status: "failed" as const };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={phase}
        organizationId="org-1"
        expandSteps
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(screen.getByText("Run script")).toBeInTheDocument();
    expect(screen.getByLabelText("failed")).toBeInTheDocument();
  });

  it("keeps the empty sentence for a finished run with no steps", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      rootEventId: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false }));
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("No steps for this run.")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });
});
