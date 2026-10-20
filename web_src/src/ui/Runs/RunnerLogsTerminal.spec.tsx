import { act, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { RunnerLogsTerminal } from "./RunInspectorRunnerLogs";

afterEach(() => vi.restoreAllMocks());

describe("RunnerLogsTerminal saved output", () => {
  it.each(["legacy", "integrated"] as const)(
    "keeps loading after the %s connection opens until saved output arrives",
    async (backend) => {
      const output = openSavedLogs(backend);
      await waitFor(() => expect(output.fetch).toHaveBeenCalledTimes(backend === "integrated" ? 3 : 2));

      expect(screen.getByRole("status")).toHaveTextContent("Loading logs");
      expect(screen.queryByText("No logs available")).not.toBeInTheDocument();

      await act(async () => {
        output.controller.enqueue(new TextEncoder().encode('{"type":"line","text":"Saved runner output"}\n'));
      });
      expect(await screen.findByTestId("run-inspector-runner-logs-terminal")).toHaveTextContent("Saved runner output");
      expect(screen.queryByText("Loading logs")).not.toBeInTheDocument();

      await act(async () => output.controller.close());
    },
  );

  it("shows the empty state only after the saved output has finished loading", async () => {
    const output = openSavedLogs("integrated");
    await waitFor(() => expect(output.fetch).toHaveBeenCalledTimes(3));

    expect(screen.getByRole("status")).toHaveTextContent("Loading logs");
    expect(screen.queryByText("No logs available")).not.toBeInTheDocument();

    await act(async () => output.controller.close());

    expect(await screen.findByText("No logs available")).toBeInTheDocument();
    expect(screen.queryByText("Loading logs")).not.toBeInTheDocument();
  });
});

function openSavedLogs(backend: "legacy" | "integrated") {
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const body = new ReadableStream<Uint8Array>({
    start(streamController) {
      controller = streamController;
    },
  });
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(Response.json({ backend, stream_url: "https://logs.example/stream", token: "token" }));
  if (backend === "integrated") {
    fetch.mockResolvedValueOnce(
      new Response(null, {
        headers: {
          "X-SuperPlane-Log-State": "archived",
          "X-SuperPlane-Log-URL": "https://logs.example/archive",
        },
      }),
    );
  }
  fetch.mockResolvedValueOnce(new Response(body));

  render(
    <MemoryRouter>
      <RunnerLogsTerminal
        execution={{ id: "execution", state: "STATE_FINISHED", result: "RESULT_PASSED" }}
        session={{ organizationId: "organization", canvasId: "canvas" }}
      />
    </MemoryRouter>,
  );
  return { controller, fetch };
}
