import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import OwnerSetup from "./OwnerSetup";
import { OWNER_SETUP_COPY } from "./ownerSetup/ownerSetupCopy";

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

const { createToken } = vi.hoisted(() => ({ createToken: vi.fn() }));

vi.mock("@/hooks/useUserTokens", () => ({
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("./ownerSetup/LicenseStep", () => ({
  LicenseStep: ({ onContinue }: { onContinue: () => void }) => (
    <button type="button" data-testid="owner-setup-license-skip" onClick={onContinue}>
      Continue with Community
    </button>
  ),
}));

const ONBOARDING_LABELS = ["Discover", "Awaiting code", "Verify", "01 Plan", "02 Build", "03 Check", "04 Review"];

function expectOwnerSetupPreview() {
  const root = screen.getByTestId("owner-setup");
  expect(root).toHaveAttribute("data-visual", "preview");
  expect(screen.getByTestId("first-run-logo")).toHaveAttribute("alt", "SuperPlane");
  expect(root.querySelector("canvas")).toBeNull();
  expect(screen.queryByTestId("owner-setup-pane")).not.toBeInTheDocument();
  expect(screen.queryByTestId("first-run-art-pill")).not.toBeInTheDocument();
  const pane = screen.getByTestId("first-run-art-pane");
  expect(pane).toHaveAttribute("data-art-mode", "globe");
  expect(pane).toHaveAttribute("data-art-background", "#b09532");
  expect(pane).toHaveAttribute("data-art-arrow-color", "#eeede9");
  expect(pane).toHaveAttribute("data-art-count", "400");
  for (const label of ONBOARDING_LABELS) {
    expect(screen.queryByText(label)).not.toBeInTheDocument();
  }
}

const fleetManagerMetadata = {
  defaultRunnerVersion: "v0.1.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  runnerVersions: ["v0.1.0"],
  stableRunnerVersion: "v0.2.0",
  installationType: "",
  fleetPresets: [
    {
      id: "e1-large-amd64",
      spec: {
        operatingSystem: "linux",
        architecture: "amd64",
        cpuMillicores: 2000,
        memoryMb: 8192,
        diskGb: 30,
        capabilities: ["docker"],
      },
    },
  ],
  infrastructureDefaults: {
    source: "terraform",
    provider: "aws",
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    fields: {
      superplaneUrl: "https://superplane.example.com",
      region: "us-east-1",
      subnetIds: ["subnet-a"],
      securityGroupIds: ["sg-runner"],
      runnerIdentity: "runner-profile",
      fleetManagerIdentity: "arn:aws:iam::123456789012:role/fleet-manager",
      imageAmd64: "ami-amd64",
    },
  },
};

interface OwnerSetupFetchStubs {
  status?: unknown;
  account?: { status: number; body?: unknown };
  organizations?: unknown;
  fleetManager?: typeof fleetManagerMetadata;
}

function stubAccountResponse(stubs: OwnerSetupFetchStubs) {
  const account = stubs.account ?? { status: 401 };
  if (account.status !== 200) {
    return new Response("", { status: account.status });
  }
  return jsonResponse(account.body ?? { email: "ada@example.com", installation_admin: true });
}

function stubOwnerSetupFetch(stubs: OwnerSetupFetchStubs = {}) {
  const routes: Array<[string, () => Response]> = [
    ["/api/v1/setup-owner/status", () => jsonResponse(stubs.status ?? { status: "required" })],
    ["/api/v1/setup-owner/progress", () => jsonResponse({})],
    ["/api/v1/setup-owner", () => jsonResponse({ organization_id: "org-1", organization_slug: "acme" })],
    ["/account", () => stubAccountResponse(stubs)],
    ["/organizations", () => jsonResponse(stubs.organizations ?? [])],
    ["/admin/api/installation/fleet-manager/setup", () => jsonResponse(stubs.fleetManager ?? fleetManagerMetadata)],
  ];

  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    const route = routes.find(([suffix]) => url.endsWith(suffix));
    return route ? route[1]() : new Response("", { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderOwnerSetup() {
  render(
    <MemoryRouter>
      <OwnerSetup />
    </MemoryRouter>,
  );
}

async function reachFleetStep() {
  fireEvent.change(await screen.findByPlaceholderText("First name"), { target: { value: "Ada" } });
  fireEvent.change(screen.getByPlaceholderText("Last name"), { target: { value: "Lovelace" } });
  fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "ada@example.com" } });
  fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "Secret1x" } });
  fireEvent.change(screen.getByPlaceholderText("Confirm password"), { target: { value: "Secret1x" } });
  fireEvent.click(screen.getByRole("button", { name: "Continue" }));

  expect(await screen.findByTestId("owner-setup-license-skip")).toBeInTheDocument();
  fireEvent.click(screen.getByTestId("owner-setup-license-skip"));

  expect(await screen.findByTestId("owner-setup-smtp")).toBeInTheDocument();
  fireEvent.click(screen.getByTestId("owner-setup-smtp-skip"));

  expect(await screen.findByTestId("owner-setup-fleet")).toBeInTheDocument();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
}

describe("OwnerSetup", () => {
  const originalLocation = window.location;

  beforeEach(() => {
    window.localStorage.clear();
    createToken.mockReset();
    createToken.mockResolvedValue({ plaintext: "owner-setup-token" });
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, href: "http://localhost/setup" },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", { configurable: true, value: originalLocation });
    vi.unstubAllGlobals();
  });

  it("submits the checked install report on the owner account step", async () => {
    const fetchMock = stubOwnerSetupFetch({
      status: { status: "required", beacon_enabled: true },
    });
    renderOwnerSetup();

    const report = await screen.findByTestId("owner-setup-install-report");
    expect(report).toBeChecked();

    fireEvent.change(screen.getByPlaceholderText("First name"), { target: { value: "Ada" } });
    fireEvent.change(screen.getByPlaceholderText("Last name"), { target: { value: "Lovelace" } });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "ada@example.com" } });
    fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "Secret1x" } });
    fireEvent.change(screen.getByPlaceholderText("Confirm password"), { target: { value: "Secret1x" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => {
      const calls = fetchMock.mock.calls as unknown as Array<[RequestInfo | URL, RequestInit | undefined]>;
      const post = calls.find((call) => String(call[0]).endsWith("/api/v1/setup-owner") && call[1]?.method === "POST");
      expect(post?.[1]?.body).toBe(
        JSON.stringify({
          email: "ada@example.com",
          first_name: "Ada",
          last_name: "Lovelace",
          password: "Secret1x",
          share_owner_identity: true,
        }),
      );
    });
  });

  it("submits a cleared install report as opted out", async () => {
    const fetchMock = stubOwnerSetupFetch({
      status: { status: "required", beacon_enabled: true },
    });
    renderOwnerSetup();

    fireEvent.click(await screen.findByTestId("owner-setup-install-report"));
    fireEvent.change(screen.getByPlaceholderText("First name"), { target: { value: "Ada" } });
    fireEvent.change(screen.getByPlaceholderText("Last name"), { target: { value: "Lovelace" } });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "ada@example.com" } });
    fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "Secret1x" } });
    fireEvent.change(screen.getByPlaceholderText("Confirm password"), { target: { value: "Secret1x" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    await waitFor(() => {
      const calls = fetchMock.mock.calls as unknown as Array<[RequestInfo | URL, RequestInit | undefined]>;
      const post = calls.find((call) => String(call[0]).endsWith("/api/v1/setup-owner") && call[1]?.method === "POST");
      expect(JSON.parse(String(post?.[1]?.body)).share_owner_identity).toBe(false);
    });
  });

  it("hides the install report when the beacon is off", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Create the owner account" })).toBeInTheDocument();
    expect(screen.queryByTestId("owner-setup-install-report")).not.toBeInTheDocument();
  });

  it("shows the gold arrow preview without onboarding labels", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Create the owner account" })).toBeInTheDocument();
    expectOwnerSetupPreview();
    expect(screen.queryByTestId("first-run-signed-in")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-log-out")).not.toBeInTheDocument();
  });

  it("reaches workspace setup when SMTP and Fleet Manager are skipped", async () => {
    const fetchMock = stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    expectOwnerSetupPreview();
    expect(screen.getByTestId("first-run-signed-in")).toHaveTextContent("ada@example.com");
    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-progress")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/setup-owner/progress",
      expect.objectContaining({ method: "PATCH", body: JSON.stringify({ step: "" }) }),
    );
    expect(screen.queryByText(OWNER_SETUP_COPY.smtp.headline)).not.toBeInTheDocument();
  });

  it("opens the Fleet Manager wizard inside owner setup", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));

    expect(await screen.findByRole("heading", { name: "Choose where runners run" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /fleet manager setup/i })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-progress")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByTestId("owner-setup-fleet")).toBeInTheDocument();
  });

  it("navigates forward and backward through the Fleet Manager steps", async () => {
    stubOwnerSetupFetch();
    renderOwnerSetup();
    await reachFleetStep();

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));
    expect(await screen.findByRole("heading", { name: "Choose where runners run" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Configure the default fleet" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Generate install command" }));
    expect(await screen.findByRole("heading", { name: "Create the chart values" })).toBeInTheDocument();
    expect(screen.getByText(/cat > fleet-manager-values.yaml/)).toBeInTheDocument();
    expect(createToken).toHaveBeenCalledWith({ name: "Fleet Manager" });

    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Install the chart" })).toBeInTheDocument();

    for (const heading of [
      "Create the chart values",
      "Configure the default fleet",
      "Choose where runners run",
      OWNER_SETUP_COPY.fleet.headline,
    ]) {
      fireEvent.click(screen.getByRole("button", { name: "Back" }));
      expect(await screen.findByRole("heading", { name: heading })).toBeInTheDocument();
    }

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByRole("heading", { name: OWNER_SETUP_COPY.smtp.headline })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByTestId("owner-setup-license-skip")).toBeInTheDocument();
  });

  it("shows the single-VM runner command with the created token", async () => {
    stubOwnerSetupFetch({ fleetManager: { ...fleetManagerMetadata, installationType: "single-host" } });
    renderOwnerSetup();
    await reachFleetStep();

    expect(await screen.findByText(OWNER_SETUP_COPY.fleet.singleHostBody)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));
    expect(await screen.findByRole("heading", { name: "Runners on this VM" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "AWS" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "This VM" }));
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByRole("heading", { name: "Start runners on this VM" })).toBeInTheDocument();
    expect(screen.getByText("./enable-runners.sh 'owner-setup-token'")).toBeInTheDocument();
    expect(screen.queryByText(/helm upgrade --install fleet-manager/)).not.toBeInTheDocument();
  });

  it("leaves a cloud install off the This VM command", async () => {
    const fetchMock = stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-docker" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Choose where runners run" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "This VM" })).not.toBeInTheDocument();
    expect(screen.queryByText(/enable-runners.sh/)).not.toBeInTheDocument();
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/v1/setup-owner/progress",
        expect.objectContaining({ method: "PATCH", body: JSON.stringify({ step: "fleet-provider" }) }),
      ),
    );
  });

  it("leaves a single-host install off the cloud chart steps", async () => {
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-fleets" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
      fleetManager: { ...fleetManagerMetadata, installationType: "single-host" },
    });
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Runners on this VM" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Configure the default fleet" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "AWS" })).not.toBeInTheDocument();
  });

  it("resumes the stored fleet step for the signed-in admin", async () => {
    const replaceState = vi.spyOn(window.history, "replaceState");
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-fleets" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Configure the default fleet" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Create the owner account" })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-signed-in")).toHaveTextContent("ada@example.com");
    expect(replaceState).toHaveBeenCalledWith(null, "", "/setup?step=fleet-fleets");
  });

  it("sends a signed-out admin in progress to sign in and back to setup", async () => {
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "smtp" },
      account: { status: 401 },
    });
    renderOwnerSetup();

    await waitFor(() => expect(window.location.href).toBe("/login?redirect=/setup"));
    expect(screen.queryByRole("heading", { name: "Create the owner account" })).not.toBeInTheDocument();
  });

  it("sends a signed-in user away when setup is complete", async () => {
    stubOwnerSetupFetch({
      status: { status: "complete" },
      account: { status: 200 },
    });
    renderOwnerSetup();

    await waitFor(() => expect(window.location.href).toBe("/"));
    expect(screen.queryByRole("heading", { name: "Create the owner account" })).not.toBeInTheDocument();
  });

  it("sends a signed-in member away from an in-progress setup", async () => {
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "smtp" },
      account: { status: 200, body: { email: "member@example.com", installation_admin: false } },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    await waitFor(() => expect(window.location.href).toBe("/"));
    expect(screen.queryByRole("heading", { name: OWNER_SETUP_COPY.smtp.headline })).not.toBeInTheDocument();
  });

  it("prepares a personal token before showing a resumed values command", async () => {
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-values" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByText(/installationAdminToken: owner-setup-token/)).toBeInTheDocument();
    expect(createToken).toHaveBeenCalledTimes(1);
    expect(screen.queryByText(/<<'EOF'\s*EOF/)).not.toBeInTheDocument();
  });

  it("keeps the values command hidden when a resumed token cannot be created", async () => {
    createToken.mockRejectedValueOnce(new Error("no token"));
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-values" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByRole("alert")).toHaveTextContent(/could not create the Fleet Manager personal token/i);
    expect(screen.getByRole("button", { name: "Finish later" })).toBeInTheDocument();
    expect(screen.queryByText(/cat >/)).not.toBeInTheDocument();

    createToken.mockResolvedValue({ plaintext: "owner-setup-token" });
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText(/installationAdminToken: owner-setup-token/)).toBeInTheDocument();
  });

  it("keeps Finish later available while a resumed values command is prepared", async () => {
    let releaseToken: (value: { plaintext: string }) => void = () => undefined;
    createToken.mockImplementation(
      () =>
        new Promise((resolve) => {
          releaseToken = resolve;
        }),
    );
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-values" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByText("Preparing the values command...")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Finish later" })).toBeInTheDocument();

    releaseToken({ plaintext: "owner-setup-token" });
    expect(await screen.findByText(/installationAdminToken: owner-setup-token/)).toBeInTheDocument();
  });

  it("closes setup from Finish later when the resumed values command fails", async () => {
    createToken.mockRejectedValueOnce(new Error("no token"));
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-values" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    fireEvent.click(await screen.findByRole("button", { name: "Finish later" }));

    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
  });

  it("shows the Helm command when a resumed install step has no personal token", async () => {
    stubOwnerSetupFetch({
      status: { status: "in_progress", step: "fleet-install" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    renderOwnerSetup();

    expect(await screen.findByRole("heading", { name: "Install the chart" })).toBeInTheDocument();
    expect(screen.getByText(/helm upgrade --install fleet-manager/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Finish later" })).toBeInTheDocument();
    expect(createToken).not.toHaveBeenCalled();
  });

  it("stays on the page when closing setup fails", async () => {
    const fetchMock = stubOwnerSetupFetch();
    const original = fetchMock.getMockImplementation();
    let closeAttempts = 0;
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/api/v1/setup-owner/progress") && init?.body === JSON.stringify({ step: "" })) {
        closeAttempts += 1;
        if (closeAttempts === 1) return new Response("", { status: 500 });
      }
      return original?.(input) ?? new Response("", { status: 404 });
    });
    renderOwnerSetup();
    await reachFleetStep();

    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    expect(await screen.findByTestId("owner-setup-close-error")).toHaveTextContent(OWNER_SETUP_COPY.closeError);
    expect(window.location.href).toBe("http://localhost/setup");

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.closeRetry }));
    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
  });

  it("sends setup progress saves in navigation order", async () => {
    let releaseFirst: () => void = () => undefined;
    const firstSave = new Promise<void>((resolve) => {
      releaseFirst = resolve;
    });
    const steps: string[] = [];
    let progressCalls = 0;
    const fetchMock = stubOwnerSetupFetch({
      status: { status: "in_progress", step: "license" },
      account: { status: 200 },
      organizations: [{ slug: "acme" }],
    });
    const original = fetchMock.getMockImplementation();
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/api/v1/setup-owner/progress")) {
        progressCalls += 1;
        steps.push((JSON.parse(String(init?.body)) as { step: string }).step);
        if (progressCalls === 1) await firstSave;
      }
      return original?.(input) ?? new Response("", { status: 404 });
    });
    renderOwnerSetup();

    fireEvent.click(await screen.findByTestId("owner-setup-license-skip"));
    await waitFor(() => expect(progressCalls).toBe(1));
    fireEvent.click(screen.getByTestId("owner-setup-smtp-skip"));
    expect(progressCalls).toBe(1);

    releaseFirst();
    await waitFor(() => expect(steps).toEqual(["smtp", "fleet-intro"]));
  });

  it("sends a signed-out user to login when setup is complete", async () => {
    stubOwnerSetupFetch({
      status: { status: "complete" },
      account: { status: 401 },
    });
    renderOwnerSetup();

    await waitFor(() => expect(window.location.href).toBe("/login"));
  });
});
