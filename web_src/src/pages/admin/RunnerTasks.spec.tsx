import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import RunnerTasks from "./RunnerTasks";
import { REFRESH_INTERVAL_MS } from "./fleetAdmin";

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

type FleetFixture = {
  id: string;
  enabled?: boolean;
  queued?: string;
  pending?: string;
  idle?: string;
  busy?: string;
  runners?: Array<{ id: string; state?: string }>;
  tasks?: Array<{ id: string; state?: string }>;
  runnerTotal?: string;
  taskTotal?: string;
  runnersHaveNextPage?: boolean;
  brokerConfigured?: boolean;
  brokerTasks?: Array<{ id: string; status: string; fleet_id: string; created_at: string }>;
};

const json = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

const brokerBody = (fleets: FleetFixture[]) =>
  json({
    configured: fleets[0]?.brokerConfigured ?? false,
    tasks: fleets[0]?.brokerTasks ?? [],
  });

const fleetListBody = (fleets: FleetFixture[]) =>
  json({
    fleets: fleets.map((fleet) => ({ id: fleet.id, enabled: fleet.enabled ?? true })),
  });

const capacityBody = (fleet: FleetFixture) =>
  json({
    runnableTasks: fleet.queued ?? "0",
    pendingRunners: fleet.pending ?? "0",
    idleRunners: fleet.idle ?? "0",
    busyRunners: fleet.busy ?? "0",
    terminatedRunners: "0",
    generation: "gen",
  });

const runnersBody = (fleet: FleetFixture) =>
  json({
    runners: (fleet.runners ?? []).map((runner) => ({
      id: runner.id,
      state: runner.state ?? "idle",
      runnerVersion: "1.0.0",
      ephemeral: false,
      createdAt: "2026-10-06T12:00:00Z",
    })),
    totalCount: fleet.runnerTotal ?? String(fleet.runners?.length ?? 0),
    hasNextPage: fleet.runnersHaveNextPage ?? false,
  });

const tasksBody = (fleet: FleetFixture) =>
  json({
    tasks: (fleet.tasks ?? []).map((task) => ({
      id: task.id,
      organizationId: "org-1",
      state: task.state ?? "queued",
      queuedAt: "2026-10-06T12:00:00Z",
    })),
    totalCount: fleet.taskTotal ?? String(fleet.tasks?.length ?? 0),
    hasNextPage: false,
  });

const fleetRecordBody = (url: URL, fleets: FleetFixture[]) => {
  const fleet = fleets.find((item) => url.pathname.includes(`/fleets/${item.id}/`));
  if (!fleet) {
    return json({});
  }
  if (url.pathname.endsWith("/capacity")) {
    return capacityBody(fleet);
  }
  if (url.pathname.endsWith("/runners")) {
    return runnersBody(fleet);
  }
  if (url.pathname.endsWith("/tasks")) {
    return tasksBody(fleet);
  }
  return json({});
};

const responseFor = (url: URL, fleets: FleetFixture[]) => {
  if (url.pathname === "/admin/api/runner/tasks") {
    return brokerBody(fleets);
  }
  if (url.pathname === "/admin/api/installation/fleets") {
    return fleetListBody(fleets);
  }
  return fleetRecordBody(url, fleets);
};

const installFetch = (fleets: FleetFixture[]) => {
  const calls: string[] = [];
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = new URL(String(input), "http://localhost");
    calls.push(`${url.pathname}${url.search}`);
    return responseFor(url, fleets);
  });
  vi.stubGlobal("fetch", fetchMock);
  return { calls, fetchMock };
};

const renderPage = () => render(<RunnerTasks />, { wrapper: MemoryRouter });

const fleetCalls = (calls: string[], fleetId: string, suffix: string) =>
  calls.filter((call) => call.includes(`/fleets/${fleetId}${suffix}`));

describe("RunnerTasks", () => {
  it("selects the first fleet and requests active runner and task states", async () => {
    const { calls } = installFetch([
      {
        id: "e1-large-amd64",
        queued: "1",
        pending: "2",
        idle: "3",
        busy: "4",
        runners: [{ id: "runner-1", state: "idle" }],
        tasks: [{ id: "task-1", state: "queued" }],
      },
      {
        id: "e1-tiny-arm64",
        enabled: false,
        queued: "1",
        idle: "1",
        busy: "1",
        runners: [{ id: "runner-2", state: "busy" }],
      },
    ]);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Fleets" })).toBeInTheDocument();
    expect(await screen.findByText("runner-1")).toBeInTheDocument();
    expect(screen.getByText("Queued tasks").parentElement).toHaveTextContent("1");
    expect(screen.getByText("Pending runners").parentElement).toHaveTextContent("2");
    expect(screen.getByText("Idle runners").parentElement).toHaveTextContent("3");
    expect(screen.getByText("Busy runners").parentElement).toHaveTextContent("4");
    expect(screen.queryByText("More queued tasks than ready runners.")).not.toBeInTheDocument();

    const runnerRequest = fleetCalls(calls, "e1-large-amd64", "/runners")[0];
    const runnerURL = new URL(runnerRequest, "http://localhost");
    expect(runnerURL.searchParams.getAll("states")).toEqual(["idle", "busy"]);
    expect(runnerURL.searchParams.get("limit")).toBe("50");
    expect(runnerURL.searchParams.has("afterId")).toBe(false);
    expect(runnerURL.searchParams.has("offset")).toBe(false);

    const taskRequest = fleetCalls(calls, "e1-large-amd64", "/tasks")[0];
    expect(new URL(taskRequest, "http://localhost").searchParams.getAll("states")).toEqual([
      "queued",
      "reserved",
      "running",
    ]);

    await user.click(screen.getByTestId("admin-fleet-machine-type"));
    await user.click(screen.getByRole("option", { name: "e1-tiny-arm64 (disabled)" }));

    await waitFor(() => {
      expect(fleetCalls(calls, "e1-tiny-arm64", "/runners").length).toBeGreaterThan(0);
    });
    expect(await screen.findByText("runner-2")).toBeInTheDocument();
  });

  it("shows the shortage sentence and pages runners with afterId", async () => {
    const { calls } = installFetch([
      {
        id: "e1-large-amd64",
        queued: "5",
        pending: "1",
        idle: "1",
        busy: "1",
        runners: [{ id: "runner-9", state: "idle" }],
        runnerTotal: "60",
        runnersHaveNextPage: true,
        brokerConfigured: true,
        brokerTasks: [
          {
            id: "broker-task-1",
            status: "queued",
            fleet_id: "legacy",
            created_at: "2026-10-06T12:00:00Z",
          },
        ],
      },
    ]);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("More queued tasks than ready runners.")).toBeInTheDocument();
    expect(screen.getByText("Ready runners").parentElement).toHaveTextContent("2");
    expect(await screen.findByText("broker-task-1")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Task broker" })).toBeInTheDocument();

    await user.click(screen.getByTestId("fleet-runners-next"));

    await waitFor(() => {
      const paged = fleetCalls(calls, "e1-large-amd64", "/runners").find((call) => call.includes("afterId="));
      expect(paged).toBeDefined();
      const url = new URL(paged ?? "", "http://localhost");
      expect(url.searchParams.get("afterId")).toBe("runner-9");
      expect(url.searchParams.has("offset")).toBe(false);
      expect(url.searchParams.get("limit")).toBe("50");
    });
  });

  it("keeps the task broker not-configured state", async () => {
    installFetch([]);
    renderPage();

    expect(await screen.findByText("No machine types are configured.")).toBeInTheDocument();
    expect(screen.queryByTestId("fleet-capacity-summary")).not.toBeInTheDocument();
    expect(await screen.findByText(/Runner task broker is not configured/)).toBeInTheDocument();
    expect(screen.getByText("TASK_BROKER_BASE_URL")).toBeInTheDocument();
  });

  it("shows broker tasks while the fleet list is still loading", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/installation/fleets") {
          return new Promise<Response>(() => {});
        }
        if (url.pathname === "/admin/api/runner/tasks") {
          return json({
            configured: true,
            tasks: [
              {
                id: "broker-while-loading",
                status: "queued",
                fleet_id: "legacy",
                created_at: "2026-10-06T12:00:00Z",
              },
            ],
          });
        }
        return json({});
      }),
    );

    renderPage();

    expect(await screen.findByText("broker-while-loading")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Task broker" })).toBeInTheDocument();
    expect(screen.getByText("Loading fleets...")).toBeInTheDocument();
  });

  it("ignores an older fleet list that finishes after a newer list", async () => {
    vi.useFakeTimers();
    const listResponses: Array<(response: Response) => void> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/installation/fleets") {
          return new Promise<Response>((resolve) => {
            listResponses.push(resolve);
          });
        }
        if (url.pathname === "/admin/api/runner/tasks") {
          return json({ configured: false, tasks: [] });
        }
        if (url.pathname.includes("/fleets/kept-fleet/runners")) {
          return runnersBody({ id: "kept-fleet", runners: [{ id: "kept-runner" }] });
        }
        if (url.pathname.includes("/fleets/removed-fleet/runners")) {
          return runnersBody({ id: "removed-fleet", runners: [{ id: "removed-runner" }] });
        }
        if (url.pathname.endsWith("/capacity")) {
          return capacityBody({ id: "fleet", queued: "1", idle: "1", busy: "1" });
        }
        if (url.pathname.endsWith("/tasks")) {
          return tasksBody({ id: "fleet" });
        }
        return json({});
      }),
    );

    renderPage();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(REFRESH_INTERVAL_MS);
    });
    expect(listResponses).toHaveLength(2);

    listResponses[1](fleetListBody([{ id: "kept-fleet" }]));
    expect(await screen.findByText("kept-runner")).toBeInTheDocument();

    listResponses[0](fleetListBody([{ id: "removed-fleet" }]));
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText("kept-runner")).toBeInTheDocument();
    expect(screen.queryByText("removed-runner")).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "removed-fleet" })).not.toBeInTheDocument();
  });

  it("keeps a later runner page when a refresh has no active rows", async () => {
    vi.useFakeTimers();
    let pagedRunnerCalls = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/admin/api/runner/tasks") {
          return json({ configured: false, tasks: [] });
        }
        if (url.pathname === "/admin/api/installation/fleets") {
          return fleetListBody([{ id: "e1-large-amd64" }]);
        }
        if (url.pathname.endsWith("/capacity")) {
          return capacityBody({ id: "e1-large-amd64", queued: "1", idle: "1", busy: "1" });
        }
        if (url.pathname.endsWith("/tasks")) {
          return tasksBody({ id: "e1-large-amd64" });
        }
        if (url.pathname.endsWith("/runners") && url.searchParams.get("afterId")) {
          pagedRunnerCalls += 1;
          if (pagedRunnerCalls === 1) {
            return runnersBody({
              id: "e1-large-amd64",
              runners: [{ id: "runner-page-2", state: "busy" }],
              runnerTotal: "60",
            });
          }
          return runnersBody({ id: "e1-large-amd64", runners: [], runnerTotal: "60" });
        }
        if (url.pathname.endsWith("/runners")) {
          return runnersBody({
            id: "e1-large-amd64",
            runners: [{ id: "runner-page-1", state: "idle" }],
            runnerTotal: "60",
            runnersHaveNextPage: true,
          });
        }
        return json({});
      }),
    );

    renderPage();
    expect(await screen.findByText("runner-page-1")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("fleet-runners-next"));
    expect(await screen.findByText("runner-page-2")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(REFRESH_INTERVAL_MS);
    });

    expect(await screen.findByText("No idle or busy runners.")).toBeInTheDocument();
    expect(screen.queryByText("runner-page-1")).not.toBeInTheDocument();
    expect(screen.queryByText("runner-page-2")).not.toBeInTheDocument();
    expect(screen.getByTestId("fleet-runners-previous")).toBeEnabled();
    expect(screen.getByText("Showing 0 of 60")).toBeInTheDocument();
  });
});
