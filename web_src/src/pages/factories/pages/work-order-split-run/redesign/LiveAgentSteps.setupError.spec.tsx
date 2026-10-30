import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "vitest";

import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { idleStream, implementPhase, implementStage, RUNNING_RUNNER } from "./LiveAgentSteps.testHelpers";

vi.mock("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas: vi.fn() }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({ useLiveLogStream: vi.fn() }));

const FAILED_RUN_HREF = "/demo/workspaces/newwo/automations/app/runs/run-1?from=task&orderNumber=78";

it("shows the runner setup error when no transcript exists", async () => {
  const message =
    'failed to resolve environmentFrom integration secrets: integration "CircleCI" does not provide secrets';
  vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
    enabled: true,
    isError: false,
    isLoading: false,
    canvas: undefined,
    rootEventId: undefined,
    stream: [{ ...RUNNING_RUNNER, status: "failed", detail: message }],
  });
  vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false }));

  render(
    <MemoryRouter>
      <LiveAgentSteps
        stage={{ ...implementStage(), status: "failed", agentSteps: [] }}
        phase={{ ...implementPhase(), status: "failed" }}
        organizationId="org-1"
        emptyNote="No steps for this run."
        runHref={FAILED_RUN_HREF}
      />
    </MemoryRouter>,
  );

  expect(await screen.findByRole("alert")).toHaveTextContent(message);
  expect(screen.getByRole("link", { name: "View run details" })).toHaveAttribute(
    "href",
    `${FAILED_RUN_HREF}&activity=exec-implementation`,
  );
  expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
});
