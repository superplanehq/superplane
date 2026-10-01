import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { idleStream, implementPhase, implementStage, RUNNING_RUNNER } from "./LiveAgentSteps.testHelpers";

vi.mock("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas: vi.fn() }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({ useLiveLogStream: vi.fn() }));

describe("LiveAgentSteps finished runs", () => {
  beforeEach(() => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
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

  it("keeps the empty sentence for a finished run with no steps", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
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
