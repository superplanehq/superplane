import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { FLEET_MANAGER_DRAFT_KEY, saveFleetManagerDraft } from "@/lib/fleetManagerSetup";
import { FleetManagerSetup } from "./FleetManagerSetup";

const fleetSpec = {
  operatingSystem: "linux",
  architecture: "amd64" as const,
  cpuMillicores: 2000,
  memoryMb: 8192,
  diskGb: 30,
  capabilities: ["docker"],
};

const metadata = {
  defaultRunnerVersion: "v0.1.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  fleetPresets: [
    { id: "e1-large-amd64", spec: fleetSpec },
    { id: "e1-large-arm64", spec: { ...fleetSpec, architecture: "arm64" as const } },
  ],
  infrastructureDefaults: {
    source: "terraform" as const,
    provider: "aws" as const,
    managerId: "terraform-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    fields: {
      superplaneUrl: "https://superplane.example.com",
      region: "us-east-1",
      subnetIds: ["subnet-a", "subnet-b"],
      securityGroupIds: ["sg-runner"],
      runnerIdentity: "runner-profile",
      fleetManagerIdentity: "arn:aws:iam::123456789012:role/fleet-manager",
    },
  },
  credentials: {
    existingSecret: "superplane-fleet-manager-token",
    tokenKey: "INSTALLATION_ADMIN_TOKEN",
    autoProvisioned: true,
  },
};

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

function fleetRequestCount(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.filter(([input]) => String(input).endsWith("/admin/api/installation/fleets")).length;
}

describe("FleetManagerSetup", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(metadata);
        if (url.endsWith("/admin/api/installation/fleets")) {
          return jsonResponse({ fleets: [{ id: "e1-large-amd64", runnerVersion: "v0.1.0", spec: fleetSpec }] });
        }
        return new Response("", { status: 404 });
      }),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  it("shows provider, fleets, infrastructure, and values on one page", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Choose runner fleets" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Confirm cloud infrastructure" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Install Fleet Manager" })).toBeInTheDocument();
    expect(screen.getByText("values.yaml")).toBeInTheDocument();
    expect(screen.getByText(/existingSecret: superplane-fleet-manager-token/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy values.yaml" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Download values.yaml" })).toBeInTheDocument();
    expect(screen.getByText("Run this command where kubectl can access your cluster.")).toBeInTheDocument();
    expect(screen.getByText(/--wait --timeout 10m/)).toBeInTheDocument();
    expect(screen.queryByText(/step \d+ of/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Continue" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Fleet Manager setup saved" })).not.toBeInTheDocument();
    expect(fleetRequestCount(vi.mocked(fetch))).toBe(0);
  });

  it("uses Terraform defaults without creating or displaying a token", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();
    expect(screen.getAllByText("Loaded from Terraform").length).toBeGreaterThan(0);
    expect(screen.queryByText(/create fleet manager token/i)).not.toBeInTheDocument();
    expect(screen.getByLabelText("AWS region")).toHaveValue("us-east-1");
    expect(screen.getByLabelText("Private subnet IDs")).toHaveValue("subnet-a, subnet-b");
    expect(screen.getByLabelText("SuperPlane URL")).toHaveValue("https://superplane.example.com");

    const yaml = screen.getByText(/existingSecret: superplane-fleet-manager-token/).textContent ?? "";
    expect(yaml).toContain("tokenKey: INSTALLATION_ADMIN_TOKEN");
    expect(yaml).not.toContain("kubernetes-api-token");
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("INSTALLATION_ADMIN_TOKEN");
  });

  it("uses Terraform defaults instead of an incompatible legacy draft", async () => {
    window.localStorage.setItem(
      "superplane.fleet-manager-setup.v1",
      JSON.stringify({
        provider: "aws",
        managerId: "my-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: window.location.origin,
        runnerVersion: "v0.1.0",
        selectedFleetIds: ["e1-large-amd64"],
        warmCapacity: 5,
        maxCapacity: 100,
        fields: {},
      }),
    );

    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Fleet Manager ID")).toHaveValue("terraform-cluster");
    expect(screen.getByLabelText("SuperPlane URL")).toHaveValue("https://superplane.example.com");
    expect(screen.getByLabelText("AWS region")).toHaveValue("us-east-1");
    expect(screen.getByLabelText("Private subnet IDs")).toHaveValue("subnet-a, subnet-b");
    expect(window.localStorage.getItem("superplane.fleet-manager-setup.v1")).toBeNull();
  });

  it("preserves explicit infrastructure changes in the current draft", async () => {
    saveFleetManagerDraft({
      provider: "aws",
      managerId: "my-cluster",
      namespace: "superplane",
      serviceAccountName: "custom-fleet-manager",
      superplaneUrl: "https://custom.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
      fields: { region: "us-west-2" },
    });

    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Fleet Manager ID")).toHaveValue("my-cluster");
    expect(screen.getByLabelText("Fleet Manager service account")).toHaveValue("custom-fleet-manager");
    expect(screen.getByLabelText("SuperPlane URL")).toHaveValue("https://custom.example.com");
    expect(screen.getByLabelText("AWS region")).toHaveValue("us-west-2");
  });

  it("configures capacity independently for each fleet", async () => {
    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("checkbox", { name: /e1-large-arm64/ }));

    fireEvent.change(screen.getByLabelText("Warm capacity for e1-large-amd64"), { target: { value: "2" } });
    fireEvent.change(screen.getByLabelText("Maximum capacity for e1-large-amd64"), { target: { value: "10" } });
    fireEvent.change(screen.getByLabelText("Maximum capacity for e1-large-arm64"), { target: { value: "3" } });

    expect(screen.getByLabelText("Warm capacity for e1-large-amd64")).toHaveValue(2);
    expect(screen.getByLabelText("Maximum capacity for e1-large-amd64")).toHaveValue(10);
    expect(screen.getByLabelText("Warm capacity for e1-large-arm64")).toHaveValue(0);
    expect(screen.getByLabelText("Maximum capacity for e1-large-arm64")).toHaveValue(3);
  });

  it("uses the reachable SuperPlane URL entered by the administrator", async () => {
    render(<FleetManagerSetup />);
    fireEvent.change(await screen.findByLabelText("SuperPlane URL"), {
      target: { value: "https://runners.example.com" },
    });

    const yaml = screen.getByText(/url: https:\/\/runners.example.com/).textContent ?? "";
    expect(yaml).toContain("url: https://runners.example.com");
    expect(yaml).not.toContain(window.location.origin);
  });

  it("hides incomplete YAML when the URL is invalid", async () => {
    saveFleetManagerDraft({
      provider: "aws",
      managerId: "terraform-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      superplaneUrl: "not-a-url",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
      fields: {},
    });

    render(<FleetManagerSetup />);

    expect((await screen.findAllByText("Enter a valid SuperPlane URL.")).length).toBeGreaterThan(0);
    expect(screen.queryByText(/existingSecret:/)).not.toBeInTheDocument();
    expect(screen.queryByText("values.yaml")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check installation" })).toBeDisabled();

    fireEvent.change(screen.getByLabelText("SuperPlane URL"), {
      target: { value: "https://superplane.example.com" },
    });

    expect(screen.getByText(/existingSecret: superplane-fleet-manager-token/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check installation" })).toBeEnabled();
  });

  it("hides incomplete YAML when no fleet is selected", async () => {
    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("checkbox", { name: /e1-large-amd64/ }));

    expect(screen.getAllByText("Select at least one fleet.").length).toBeGreaterThan(0);
    expect(screen.queryByText(/existingSecret:/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check installation" })).toBeDisabled();
  });

  it("clears incompatible fields when the provider changes", async () => {
    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "GCP" }));

    expect(screen.getByLabelText("GCP project")).toHaveValue("");
    expect(screen.queryByLabelText("AWS region")).not.toBeInTheDocument();
  });

  it("starts fleet verification only after Check installation", async () => {
    render(<FleetManagerSetup />);
    expect(await screen.findByRole("button", { name: "Check installation" })).toBeEnabled();
    expect(fleetRequestCount(vi.mocked(fetch))).toBe(0);

    fireEvent.click(screen.getByRole("button", { name: "Check installation" }));

    expect(await screen.findByText(/created every selected fleet/i)).toBeInTheDocument();
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).toBeNull();
    expect(screen.queryByRole("heading", { name: "Fleet Manager setup saved" })).not.toBeInTheDocument();
    expect(fleetRequestCount(vi.mocked(fetch))).toBe(1);
  });

  it("explains authentication failures and retries verification", async () => {
    let fleetRequests = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(metadata);
        if (url.endsWith("/admin/api/installation/fleets")) {
          fleetRequests += 1;
          if (fleetRequests === 1) return new Response("", { status: 401 });
          return jsonResponse({ fleets: [{ id: "e1-large-amd64", runnerVersion: "v0.1.0", spec: fleetSpec }] });
        }
        return new Response("", { status: 404 });
      }),
    );

    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "Check installation" }));

    expect(await screen.findByText(/sign in again and retry/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry verification" }));

    expect(await screen.findByText(/created every selected fleet/i)).toBeInTheDocument();
    expect(fleetRequests).toBe(2);
  });

  it("stops verification after the bounded timeout", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(metadata);
        if (url.endsWith("/admin/api/installation/fleets")) return jsonResponse({ fleets: [] });
        return new Response("", { status: 404 });
      }),
    );

    render(<FleetManagerSetup />);
    const now = vi
      .spyOn(Date, "now")
      .mockReturnValueOnce(0)
      .mockReturnValue(10 * 60_000);
    fireEvent.click(await screen.findByRole("button", { name: "Check installation" }));

    expect(await screen.findByText(/verification timed out/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry verification" })).toBeEnabled();
    now.mockRestore();
  });
});
