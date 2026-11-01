import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Link, MemoryRouter, Route, Routes, useParams } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { PermissionsContext } from "@/contexts/permissionsContextState";
import { OrganizationFleets, OrganizationFleetDetails } from "./OrganizationFleets";
import { fleetCapacityResponse } from "./fleetTestFixtures";
import { useOrganizationFleetAccess } from "@/hooks/useOrganizationFleetManagement";

const fleet = {
  id: "self-hosted",
  enabled: true,
  runnerVersion: "v1",
  spec: {
    operatingSystem: "linux",
    architecture: "amd64",
    cpuMillicores: 1000,
    memoryMb: 1024,
    diskGb: 20,
    capabilities: ["docker"],
  },
};
const syntheticToken = "synthetic-registration-secret";
const originalConfig = client.getConfig();
let enabled: boolean;
let writable: boolean;
let fleets: (typeof fleet)[];
let runnerState: string;
let ephemeral: boolean;
let registered: boolean;
let expired: boolean;
let createFailure: boolean;
let terminateConflict: boolean;
let capacityFailure: boolean;
let registryFailure: boolean;
let requests: Request[];
let bodies: Record<string, unknown>[];
let queryClient: QueryClient;

function FleetNavigation() {
  const { organizationId = "" } = useParams();
  const access = useOrganizationFleetAccess(organizationId);
  return access.showNavigation ? <Link to={`/${organizationId}/fleets`}>Fleets navigation</Link> : null;
}

function CatalogRoute() {
  const { organizationId = "" } = useParams();
  return (
    <>
      <FleetNavigation />
      <OrganizationFleets key={organizationId} organizationId={organizationId} basePath={`/${organizationId}/fleets`} />
    </>
  );
}

function runnerHost() {
  if (!registered) return {};
  return { registeredAt: new Date().toISOString(), hostname: "my-host", os: "linux", arch: "amd64", ip: "192.0.2.1" };
}

function mount(path = "/org-1/fleets", organizationId = "org-1") {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <PermissionsContext.Provider
        value={{
          permissions: [],
          isLoading: false,
          hasPermission: () => true,
          canAct: (_resource, action) => action === "read" || writable,
          browserNotificationPreferences: { enabled: false, showWhileViewing: true },
        }}
      >
        <MemoryRouter initialEntries={[path]}>
          <Link to="/org-2/fleets">Switch organization</Link>
          <Routes>
            <Route path="/:organizationId/fleets" element={<CatalogRoute />} />
            <Route
              path="/:organizationId/fleets/:fleetId"
              element={
                <OrganizationFleetDetails organizationId={organizationId} basePath={`/${organizationId}/fleets`} />
              }
            />
          </Routes>
        </MemoryRouter>
      </PermissionsContext.Provider>
    </QueryClientProvider>,
  );
}

async function refresh() {
  await act(async () => {
    await queryClient.invalidateQueries();
  });
}

async function generateRegistration() {
  fireEvent.click(await screen.findByRole("button", { name: "Connect runner" }));
  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  fireEvent.click(screen.getByRole("button", { name: "Generate registration" }));
}

beforeEach(() => {
  enabled = true;
  writable = true;
  fleets = [structuredClone(fleet)];
  runnerState = "pending";
  ephemeral = true;
  registered = false;
  expired = false;
  createFailure = false;
  terminateConflict = false;
  capacityFailure = false;
  registryFailure = false;
  vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
    registryFailure
      ? Response.json({}, { status: 503 })
      : Response.json({ features: [{ id: "organization_runners", released: false }] }),
  );
  requests = [];
  bodies = [];
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const request = input as Request;
      requests.push(request);
      const url = new URL(request.url);
      const runner = {
        id: "runner-1",
        fleetId: "self-hosted",
        state: runnerState,
        ephemeral,
        ...runnerHost(),
      };
      const json = (body: unknown, status = 200) => Response.json(body, { status });
      if (url.pathname.endsWith("/capacity")) return fleetCapacityResponse(capacityFailure);
      if (url.pathname.endsWith("/fleets")) {
        if (request.method === "POST") {
          const body = await request.json();
          bodies.push(body);
          return json({ fleet: { ...fleet, id: body.fleetId } });
        }
        return json({ fleets });
      }
      if (url.pathname.endsWith("/fleets/self-hosted")) {
        if (request.method === "PATCH") bodies.push(await request.json());
        return json({ fleet: fleets[0] });
      }
      if (url.pathname.endsWith("/runners")) {
        if (request.method === "POST") {
          const body = await request.json();
          bodies.push(body);
          ephemeral = body.ephemeral;
          if (createFailure) return json({ code: 9, message: syntheticToken }, 400);
          return json({
            runner,
            registrationToken: syntheticToken,
            runnerApiUrl: "https://runner.example",
            registrationExpiresAt: new Date(Date.now() + (expired ? -1000 : 60000)).toISOString(),
          });
        }
        return json({ runners: [runner], totalCount: "51", hasNextPage: !url.searchParams.has("afterId") });
      }
      if (url.pathname.endsWith("/runners/runner-1")) {
        if (request.method === "DELETE") {
          if (terminateConflict) return json({ code: 10 }, 409);
          if (runnerState !== "busy") {
            runnerState = "terminated";
            runner.state = "terminated";
          }
        }
        return json({ runner });
      }
      return json({
        organization: {
          metadata: { id: "org-1" },
          spec: { enabledExperimentalFeatures: enabled ? ["organization_runners"] : [] },
        },
      });
    },
  });
});

afterEach(() => {
  cleanup();
  queryClient?.clear();
  client.setConfig(originalConfig);
  vi.restoreAllMocks();
});

describe("organization fleet management", () => {
  it("restores configuration when Retry reloads a recovered feature registry", async () => {
    registryFailure = true;
    mount();
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load fleet configuration access");
    expect(screen.queryByRole("button", { name: "Create fleet" })).not.toBeInTheDocument();
    registryFailure = false;
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("button", { name: "Create fleet" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
  it("creates an enabled fleet and changes its runner version without changing its ID", async () => {
    const view = mount();
    fireEvent.click(await screen.findByRole("button", { name: "Create fleet" }));
    fireEvent.change(screen.getByLabelText("Fleet ID"), { target: { value: "new-fleet" } });
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.change(screen.getByLabelText("Runner version"), { target: { value: "v2" } });
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    const user = userEvent.setup();
    await user.click(screen.getByRole("combobox", { name: "Architecture" }));
    await user.click(await screen.findByRole("option", { name: "ARM64 (arm64)" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    for (const [label, value] of [
      ["CPU millicores", "2000"],
      ["Memory MB", "4096"],
      ["Disk GB", "40"],
      ["Capabilities (comma-separated)", "docker, gpu"],
    ]) {
      fireEvent.change(screen.getByLabelText(label), { target: { value } });
    }
    fireEvent.click(screen.getByRole("button", { name: "Create fleet" }));
    await waitFor(() => expect(bodies).toHaveLength(1));
    expect(bodies[0]).toEqual({
      fleetId: "new-fleet",
      enabled: true,
      runnerVersion: "v2",
      spec: {
        operatingSystem: "linux",
        architecture: "arm64",
        cpuMillicores: 2000,
        memoryMb: 4096,
        diskGb: 40,
        capabilities: ["docker", "gpu"],
      },
    });
    view.unmount();
    mount("/org-1/fleets/self-hosted");
    fireEvent.click(await screen.findByRole("button", { name: "Change runner version" }));
    fireEvent.change(screen.getByLabelText("Runner version"), { target: { value: "v3" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(bodies).toHaveLength(2));
    expect(bodies[1]).toEqual({ runnerVersion: "v3" });
    expect(bodies[1]).not.toHaveProperty("fleetId");
  });
  it("hides setup navigation only after a disabled empty catalog is confirmed", async () => {
    enabled = false;
    fleets = [];
    mount();
    await screen.findByText(/No organization fleets/);
    await waitFor(() => expect(screen.queryByRole("link", { name: "Fleets navigation" })).not.toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Create fleet" })).not.toBeInTheDocument();
  });

  it("ignores a deferred catalog from the previous organization", async () => {
    let resolveOld!: (response: Response) => void;
    const pending = new Promise<Response>((resolve) => {
      resolveOld = resolve;
    });
    const defaultFetch = client.getConfig().fetch!;
    let oldSignal: AbortSignal | undefined;
    client.setConfig({
      fetch: async (input) => {
        const request = input as Request;
        if (request.url.endsWith("/organizations/org-1/fleets")) {
          oldSignal = request.signal;
          return pending;
        }
        return defaultFetch(input);
      },
    });
    mount();
    await waitFor(() => expect(oldSignal).toBeDefined());
    fireEvent.click(screen.getByRole("link", { name: "Switch organization" }));
    await screen.findByRole("link", { name: "self-hosted" });
    await act(async () => {
      resolveOld(Response.json({ fleets: [{ ...fleet, id: "other-organization-private" }] }));
    });
    expect(oldSignal?.aborted).toBe(true);
    expect(screen.queryByText("other-organization-private")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "self-hosted" })).toHaveAttribute("href", "/org-2/fleets/self-hosted");
  });

  it("aborts an outstanding registration request on close", async () => {
    const defaultFetch = client.getConfig().fetch!;
    let signal: AbortSignal | undefined;
    let resolveCreate!: (response: Response) => void;
    const pending = new Promise<Response>((resolve) => {
      resolveCreate = resolve;
    });
    client.setConfig({
      fetch: async (input) => {
        const request = input as Request;
        if (request.method === "POST" && request.url.endsWith("/runners")) {
          signal = request.signal;
          return pending;
        }
        return defaultFetch(input);
      },
    });
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    await waitFor(() => expect(signal).toBeDefined());
    fireEvent.click(screen.getByRole("button", { name: "Back to fleet" }));
    await act(async () => {
      resolveCreate(
        Response.json({
          runner: { id: "late" },
          registrationToken: syntheticToken,
          runnerApiUrl: "https://runner.example",
          registrationExpiresAt: new Date(Date.now() + 60000).toISOString(),
        }),
      );
    });
    expect(signal?.aborted).toBe(true);
    expect(document.body.textContent).not.toContain(syntheticToken);
  });

  it("stops catalog, capacity, runner, and registration polling after navigation", async () => {
    const page = mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    await screen.findByText(/runner --url/);
    await waitFor(() => expect(requests.some((request) => request.url.includes("/runners/runner-1"))).toBe(true));
    page.unmount();
    const count = requests.length;
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 5100));
    });
    expect(requests).toHaveLength(count);
  });
  it("lists disabled organization fleets through management APIs and isolates capacity failures", async () => {
    fleets[0].enabled = false;
    mount();
    expect(await screen.findByRole("link", { name: "self-hosted" })).toHaveAttribute(
      "href",
      "/org-1/fleets/self-hosted",
    );
    expect(screen.getByRole("button", { name: "This fleet is disabled." })).toBeInTheDocument();
    await screen.findByRole("button", { name: "Idle runners: 1" });
    capacityFailure = true;
    await refresh();
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load capacity");
    expect(screen.getByRole("link", { name: "self-hosted" })).toBeInTheDocument();
    expect(requests.some((request) => new URL(request.url).pathname === "/api/v1/organizations/org-1/fleets")).toBe(
      true,
    );
    expect(
      requests.every(
        (request) => !request.url.includes("/available") && request.headers.get("x-organization-id") === "org-1",
      ),
    ).toBe(true);
  });

  it("removes configuration on open-page disablement and preserves inspection", async () => {
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    await screen.findByText(/runner --url/);
    enabled = false;
    await refresh();
    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Change runner version" })).not.toBeInTheDocument(),
    );
    expect(screen.queryByRole("button", { name: "Connect runner" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disable fleet" })).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.body.textContent).not.toContain(syntheticToken);
    expect(screen.getByRole("heading", { name: "Runners" })).toBeInTheDocument();
  });

  it("keeps administrator actions unavailable to read-only users", async () => {
    writable = false;
    mount("/org-1/fleets/self-hosted");
    await screen.findByText("runner-1");
    expect(screen.queryByRole("button", { name: "Terminate runner" })).not.toBeInTheDocument();
  });

  it("sends page cursors and retains the current page during refresh", async () => {
    mount("/org-1/fleets/self-hosted");
    fireEvent.click(await screen.findByRole("button", { name: "Next" }));
    await waitFor(() =>
      expect(requests.some((request) => new URL(request.url).searchParams.get("afterId") === "runner-1")).toBe(true),
    );
    await refresh();
    const lists = requests.filter((request) => new URL(request.url).pathname.endsWith("/runners"));
    const last = new URL(lists.at(-1)!.url);
    expect(last.searchParams.get("afterId")).toBe("runner-1");
    expect(last.searchParams.get("fleetId")).toBe("self-hosted");
    expect(last.searchParams.get("limit")).toBe("50");
    const user = userEvent.setup();
    for (const [label, state] of [
      ["Idle runners: 1", "idle"],
      ["Terminated runners: 5", "terminated"],
    ]) {
      await user.click(screen.getByRole("tab", { name: label }));
      await waitFor(() => {
        const matching = requests.filter((request) => new URL(request.url).pathname.endsWith("/runners"));
        const filtered = new URL(matching.at(-1)!.url);
        expect(filtered.searchParams.getAll("states")).toEqual([state]);
        expect(filtered.searchParams.has("afterId")).toBe(false);
      });
    }
  });

  for (const mode of ["ephemeral", "reusable"]) {
    it(`creates ${mode} capacity with a transient command and clears it on close`, async () => {
      const storage = vi.spyOn(Storage.prototype, "setItem");
      const diagnostics = vi.spyOn(console, "error");
      mount("/org-1/fleets/self-hosted");
      fireEvent.click(await screen.findByRole("button", { name: "Connect runner" }));
      if (mode === "reusable") {
        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: "Reusable" }));
      }
      fireEvent.click(screen.getByRole("button", { name: "Next" }));
      fireEvent.click(screen.getByRole("button", { name: "Generate registration" }));
      expect(await screen.findByText(/runner --url/)).toHaveTextContent(
        `./runner --url 'https://runner.example' --registration-token '${syntheticToken}'`,
      );
      const clipboard = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
      fireEvent.click(screen.getByRole("button", { name: "Copy command" }));
      await waitFor(() =>
        expect(clipboard).toHaveBeenCalledWith(
          `./runner --url 'https://runner.example' --registration-token '${syntheticToken}'`,
        ),
      );
      expect(bodies.at(-1)).toMatchObject({
        fleetId: "self-hosted",
        ephemeral: mode === "ephemeral",
        idempotencyKey: expect.any(String),
      });
      expect(bodies.at(-1)).not.toHaveProperty("taskId");
      expect(screen.getByText("Waiting for runner registration.")).toBeInTheDocument();
      const cachedQueries = queryClient.getQueryCache().getAll();
      expect(JSON.stringify(cachedQueries.map((query) => query.state.data))).not.toContain(syntheticToken);
      expect(queryClient.getMutationCache().getAll()).toHaveLength(0);
      expect(storage.mock.calls.flat().join()).not.toContain(syntheticToken);
      expect(diagnostics.mock.calls.flat().join()).not.toContain(syntheticToken);
      fireEvent.click(screen.getByRole("button", { name: "Back to fleet" }));
      expect(screen.queryByText(/runner --url/)).not.toBeInTheDocument();
    });
  }

  it("resolves registration evidence before marking an expired token expired", async () => {
    expired = true;
    registered = true;
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    expect(await screen.findByText(/Runner connected/)).toBeInTheDocument();
    expect(screen.queryByText(/runner --url/)).not.toBeInTheDocument();
  });

  it("expires pending registration and replaces its idempotency key", async () => {
    expired = true;
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    fireEvent.click(await screen.findByRole("button", { name: "Create new registration" }));
    await waitFor(() => expect(bodies).toHaveLength(2));
    expect(bodies[0].idempotencyKey).not.toBe(bodies[1].idempotencyKey);
    expect(screen.queryByText(/runner --url/)).not.toBeInTheDocument();
  });

  it("replaces a registered runner that terminated before the status check", async () => {
    registered = true;
    runnerState = "terminated";
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    const replacement = await screen.findByRole("button", { name: "Create new registration" });
    expect(screen.queryByText(/Runner connected/)).not.toBeInTheDocument();
    expect(screen.queryByText(/runner --url/)).not.toBeInTheDocument();
    runnerState = "pending";
    registered = false;
    fireEvent.click(replacement);
    expect(await screen.findByText("Waiting for runner registration.")).toBeInTheDocument();
    expect(bodies).toHaveLength(2);
    expect(bodies[0].idempotencyKey).not.toBe(bodies[1].idempotencyKey);
  });

  it("retries a failed registration without emitting server secrets or changing its key", async () => {
    createFailure = true;
    mount("/org-1/fleets/self-hosted");
    await generateRegistration();
    fireEvent.click(await screen.findByRole("button", { name: "Retry registration" }));
    await waitFor(() => expect(bodies).toHaveLength(2));
    expect(bodies[0].idempotencyKey).toBe(bodies[1].idempotencyKey);
    expect(document.body.textContent).not.toContain(syntheticToken);
  });

  for (const state of ["idle", "busy"]) {
    it(`waits for server-confirmed termination of an ${state} runner with the feature disabled`, async () => {
      enabled = false;
      runnerState = state;
      mount("/org-1/fleets/self-hosted");
      fireEvent.click(await screen.findByRole("button", { name: "Terminate runner" }));
      expect(screen.getByText("Termination does not delete the host machine.")).toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "Confirm termination" }));
      if (state === "busy") {
        expect(await screen.findByText("Termination requested.")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Terminate runner" })).toBeDisabled();
        runnerState = "terminated";
        await refresh();
      }
      await waitFor(() => expect(screen.queryByRole("button", { name: "Terminate runner" })).not.toBeInTheDocument());
      const deletion = requests.find((request) => request.method === "DELETE")!;
      expect(new URL(deletion.url).pathname).toBe("/api/v1/organizations/org-1/runners/runner-1");
      expect(new URL(deletion.url).searchParams.get("fleetId")).toBe("self-hosted");
    });
  }

  it("shows recovery after a busy reusable runner conflict", async () => {
    runnerState = "busy";
    ephemeral = false;
    terminateConflict = true;
    mount("/org-1/fleets/self-hosted");
    fireEvent.click(await screen.findByRole("button", { name: "Terminate runner" }));
    fireEvent.click(screen.getByRole("button", { name: "Confirm termination" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("wait for the task to finish and retry");
    expect(screen.getByRole("button", { name: "Terminate runner" })).toBeEnabled();
  });
});
