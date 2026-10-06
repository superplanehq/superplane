import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import RunnerTasks from "./RunnerTasks";
import { FLEET_REQUEST_TIMEOUT_MS, REFRESH_INTERVAL_MS } from "./fleetAdmin";

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const json = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

const fleetList = (id: string) => json({ fleets: [{ id, enabled: true }] });

const capacity = (queued: string) =>
  json({
    runnableTasks: queued,
    pendingRunners: "0",
    idleRunners: "1",
    busyRunners: "0",
    terminatedRunners: "0",
    generation: "gen",
  });

const runners = (id: string, hasNextPage = false) =>
  json({
    runners: [
      {
        id,
        state: "idle",
        runnerVersion: "1.0.0",
        ephemeral: false,
        createdAt: "2026-10-06T12:00:00Z",
      },
    ],
    totalCount: "60",
    hasNextPage,
  });

const tasks = () => json({ tasks: [], totalCount: "0", hasNextPage: false });

const broker = () => json({ configured: false, tasks: [] });

const renderPage = () => render(<RunnerTasks />, { wrapper: MemoryRouter });

const hold = (pending: Array<{ url: URL; resolve: (response: Response) => void }>, url: URL) =>
  new Promise<Response>((resolve) => {
    pending.push({ url, resolve });
  });

describe("RunnerTasks fleet polling", () => {
  it("requests the next runner page while a refresh is still running", async () => {
    vi.useFakeTimers();
    const pending: Array<{ url: URL; resolve: (response: Response) => void }> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/installation/fleets") {
          return fleetList("e1-large-amd64");
        }
        if (url.pathname === "/admin/api/runner/tasks") {
          return broker();
        }
        if (url.pathname.includes("/fleets/e1-large-amd64/")) {
          return hold(pending, url);
        }
        return json({});
      }),
    );

    renderPage();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    await release(pending.splice(0), "runner-page-1", true);
    expect(await screen.findByText("runner-page-1")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(REFRESH_INTERVAL_MS);
    });
    const refresh = pending.splice(0);
    expect(
      refresh.some((item) => item.url.pathname.endsWith("/runners") && !item.url.searchParams.has("afterId")),
    ).toBe(true);

    fireEvent.click(screen.getByTestId("fleet-runners-next"));
    const nextPage = pending.filter(
      (item) => item.url.pathname.endsWith("/runners") && item.url.searchParams.get("afterId") === "runner-page-1",
    );
    expect(nextPage).toHaveLength(1);

    await release(refresh, "stale-refresh", false);
    expect(screen.queryByText("stale-refresh")).not.toBeInTheDocument();

    await release(pending, "runner-page-2", false);
    expect(await screen.findByText("runner-page-2")).toBeInTheDocument();
    expect(screen.queryByText("stale-refresh")).not.toBeInTheDocument();
    expect(screen.queryByText("runner-page-1")).not.toBeInTheDocument();
  });

  it("shows a fleet list after a stalled request is abandoned", async () => {
    vi.useFakeTimers();
    const listReleases: Array<(response: Response) => void> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/installation/fleets") {
          return new Promise<Response>((resolve) => {
            listReleases.push(resolve);
          });
        }
        if (url.pathname === "/admin/api/runner/tasks") {
          return broker();
        }
        if (url.pathname.endsWith("/runners")) {
          return runners("recovered-runner");
        }
        if (url.pathname.endsWith("/capacity")) {
          return capacity("1");
        }
        if (url.pathname.endsWith("/tasks")) {
          return tasks();
        }
        return json({});
      }),
    );

    renderPage();
    expect(screen.getByText("Loading fleets...")).toBeInTheDocument();
    await advancePastRequestTimeout();
    expect(listReleases.length).toBeGreaterThanOrEqual(2);

    await act(async () => {
      listReleases[listReleases.length - 1](fleetList("recovered-fleet"));
    });
    expect(await screen.findByText("recovered-runner")).toBeInTheDocument();
    expect(screen.getByTestId("admin-fleet-machine-type")).toHaveTextContent("recovered-fleet");

    await act(async () => {
      listReleases[0](fleetList("stale-fleet"));
    });
    expect(screen.getByTestId("admin-fleet-machine-type")).toHaveTextContent("recovered-fleet");
    expect(screen.getByTestId("admin-fleet-machine-type")).not.toHaveTextContent("stale-fleet");
  });

  it("shows fleet capacity after a stalled detail request is abandoned", async () => {
    vi.useFakeTimers();
    const capacityReleases: Array<(response: Response) => void> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/installation/fleets") {
          return fleetList("stalled-fleet");
        }
        if (url.pathname === "/admin/api/runner/tasks") {
          return broker();
        }
        if (url.pathname.endsWith("/capacity")) {
          return new Promise<Response>((resolve) => {
            capacityReleases.push(resolve);
          });
        }
        if (url.pathname.endsWith("/runners")) {
          return runners("stalled-runner");
        }
        if (url.pathname.endsWith("/tasks")) {
          return tasks();
        }
        return json({});
      }),
    );

    renderPage();
    expect(await screen.findByText("Loading fleet capacity...")).toBeInTheDocument();
    expect(capacityReleases).toHaveLength(1);
    await advancePastRequestTimeout();
    expect(capacityReleases.length).toBeGreaterThanOrEqual(2);

    await act(async () => {
      capacityReleases[capacityReleases.length - 1](capacity("4"));
    });
    expect(screen.getByText("Queued tasks").parentElement).toHaveTextContent("4");

    await act(async () => {
      capacityReleases[0](capacity("9"));
    });
    expect(screen.getByText("Queued tasks").parentElement).toHaveTextContent("4");
  });
});

const release = async (
  pending: Array<{ url: URL; resolve: (response: Response) => void }>,
  runnerId: string,
  hasNextPage: boolean,
) => {
  await act(async () => {
    for (const item of pending) {
      item.resolve(detailResponse(item.url, runnerId, hasNextPage));
    }
  });
};

const detailResponse = (url: URL, runnerId: string, hasNextPage: boolean) => {
  if (url.pathname.endsWith("/capacity")) {
    return capacity("1");
  }
  if (url.pathname.endsWith("/tasks")) {
    return tasks();
  }
  return runners(runnerId, hasNextPage);
};

const advancePastRequestTimeout = async () => {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(FLEET_REQUEST_TIMEOUT_MS + REFRESH_INTERVAL_MS);
  });
};
