import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { FLEET_MANAGER_DRAFT_KEY } from "@/lib/fleetManagerSetup";
import FleetManagerSetup from "./FleetManagerSetup";

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
  fleetPresets: [{ id: "e1-large-amd64", spec: fleetSpec }],
  infrastructureDefaults: {
    source: "terraform" as const,
    provider: "aws" as const,
    managerId: "terraform-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    fields: {
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

async function continueTo(stepName: string) {
  while (!screen.queryByRole("heading", { name: stepName })) {
    fireEvent.click(await screen.findByRole("button", { name: /continue/i }));
  }
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

  it("uses Terraform defaults without creating or displaying a token", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();
    expect(screen.getAllByText("Loaded from Terraform").length).toBeGreaterThan(0);
    expect(screen.queryByText(/create fleet manager token/i)).not.toBeInTheDocument();

    await continueTo("Confirm cloud infrastructure");
    expect(screen.getByLabelText("AWS region")).toHaveValue("us-east-1");
    expect(screen.getByLabelText("Private subnet IDs")).toHaveValue("subnet-a, subnet-b");
  });

  it("generates values with a Secret reference and no credential value", async () => {
    render(<FleetManagerSetup />);
    await continueTo("Review Fleet Manager values");

    const yaml = screen.getByText(/existingSecret: superplane-fleet-manager-token/).textContent ?? "";
    expect(yaml).toContain("tokenKey: INSTALLATION_ADMIN_TOKEN");
    expect(yaml).not.toContain("kubernetes-api-token");
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("INSTALLATION_ADMIN_TOKEN");
  });

  it("clears incompatible fields when the provider changes", async () => {
    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "GCP" }));
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByLabelText("GCP project")).toHaveValue("");
    expect(screen.queryByLabelText("AWS region")).not.toBeInTheDocument();
  });

  it("verifies every selected fleet after the Helm step", async () => {
    render(<FleetManagerSetup />);
    await continueTo("Review Fleet Manager values");
    fireEvent.click(screen.getByRole("button", { name: "Continue to install" }));

    expect(await screen.findByText(/created every selected fleet/i)).toBeInTheDocument();
    expect(screen.getByText(/--wait --timeout 10m/)).toBeInTheDocument();
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Finish installation" }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Fleet Manager setup saved" })).toBeInTheDocument());
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
    await continueTo("Review Fleet Manager values");
    fireEvent.click(screen.getByRole("button", { name: "Continue to install" }));

    expect(await screen.findByText(/sign in again and retry/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry verification" }));

    expect(await screen.findByText(/created every selected fleet/i)).toBeInTheDocument();
    expect(fleetRequests).toBe(2);
  });

  it("stops verification after the bounded timeout", async () => {
    const now = vi
      .spyOn(Date, "now")
      .mockReturnValueOnce(0)
      .mockReturnValue(10 * 60_000);
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
    await continueTo("Review Fleet Manager values");
    fireEvent.click(screen.getByRole("button", { name: "Continue to install" }));

    expect(await screen.findByText(/verification timed out/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry verification" })).toBeEnabled();
    now.mockRestore();
  });
});
