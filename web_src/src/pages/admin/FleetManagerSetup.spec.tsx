import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { FLEET_MANAGER_DRAFT_KEY } from "@/lib/fleetManagerSetup";
import { FleetManagerSetup } from "./FleetManagerSetup";

const { createToken } = vi.hoisted(() => ({ createToken: vi.fn() }));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Acme" }] }),
}));

vi.mock("@/hooks/useUserTokens", () => ({
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
}));

const fleetSpec = {
  operatingSystem: "linux",
  architecture: "amd64" as const,
  cpuMillicores: 2000,
  memoryMb: 8192,
  diskGb: 30,
  capabilities: ["docker"],
};

const metadata = {
  defaultRunnerVersion: "v0.2.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  runnerVersions: ["v0.2.0", "v0.1.0"],
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
      imageAmd64: "ami-amd64",
      imageArm64: "",
    },
  },
};

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

describe("FleetManagerSetup", () => {
  beforeEach(() => {
    window.localStorage.clear();
    createToken.mockReset();
    createToken.mockResolvedValue({ plaintext: "one-time-personal-token" });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(metadata);
        if (url.endsWith("/admin/api/installation/fleets")) {
          return jsonResponse({ fleets: [{ id: "e1-large-amd64", runnerVersion: "v0.2.0", spec: fleetSpec }] });
        }
        return new Response("", { status: 404 });
      }),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  it("uses the Provider, Fleets, and Install sequence", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "Choose a cloud provider" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Choose runner fleets" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Install Fleet Manager" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /cloud infrastructure/i })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Machine type for e1-large-amd64")).toHaveValue("m8a.large");
    expect(screen.getByRole("combobox", { name: "Runner version" })).toHaveTextContent("v0.2.0");
  });

  it("generates a command when no Terraform runner image is configured", async () => {
    const manual = {
      ...metadata,
      infrastructureDefaults: {
        source: "manual" as const,
        provider: null,
        managerId: "my-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        fields: {},
      },
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(manual);
        return new Response("", { status: 404 });
      }),
    );

    render(<FleetManagerSetup />);

    const amd64 = await screen.findByRole("checkbox", { name: /e1-large-amd64/ });
    expect(amd64).toBeEnabled();
    expect(screen.queryByText(/Add the amd64 runner image through Terraform/)).not.toBeInTheDocument();
    fireEvent.click(amd64);
    fireEvent.click(screen.getByRole("button", { name: "Generate install command" }));

    const command = await screen.findByTestId("fleet-manager-install-command");
    expect(command).toHaveTextContent("ami: <RUNNER_IMAGE>");
  });

  it("lets you clear a selected fleet when Terraform has no image", async () => {
    window.localStorage.setItem(
      FLEET_MANAGER_DRAFT_KEY,
      JSON.stringify({
        provider: "aws",
        managerId: "terraform-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: "https://superplane.example.com",
        runnerVersion: "v0.2.0",
        selectedFleetIds: ["e1-large-arm64"],
        fleetCapacities: { "e1-large-arm64": { warmCapacity: 0, maxCapacity: 1 } },
        fields: { region: "us-east-1" },
      }),
    );

    render(<FleetManagerSetup />);

    const arm64 = await screen.findByRole("checkbox", { name: /e1-large-arm64/ });
    expect(arm64).toBeEnabled();
    expect(arm64).toBeChecked();
    fireEvent.click(arm64);
    expect(arm64).not.toBeChecked();
  });

  it("prevents selection when Terraform has no image for the architecture", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("checkbox", { name: /e1-large-amd64/ })).toBeEnabled();
    expect(screen.getByRole("checkbox", { name: /e1-large-arm64/ })).toBeDisabled();
    expect(screen.getByText(/Add the arm64 runner image through Terraform/)).toBeInTheDocument();
  });

  it("creates one personal token and puts it only in the values file", async () => {
    render(<FleetManagerSetup />);

    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));

    const command = await screen.findByTestId("fleet-manager-install-command");
    expect(createToken).toHaveBeenCalledTimes(1);
    expect(createToken).toHaveBeenCalledWith({ name: "Fleet Manager" });
    expect(command).toHaveTextContent("cat > fleet-manager-values.yaml");
    expect(command).toHaveTextContent("-f fleet-manager-values.yaml");
    expect(command).toHaveTextContent("installationAdminToken: one-time-personal-token");
    expect(command).toHaveTextContent("ami: ami-amd64");
    expect(command).toHaveTextContent("runnerVersion: v0.2.0");
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("one-time-personal-token");

    fireEvent.change(screen.getByLabelText("Maximum capacity for e1-large-amd64"), { target: { value: "3" } });
    await waitFor(() => expect(command).toHaveTextContent("maxCapacity: 3"));
    expect(createToken).toHaveBeenCalledTimes(1);
  });

  it("creates a new token after the page is remounted", async () => {
    const first = render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));
    await screen.findByTestId("fleet-manager-install-command");
    first.unmount();

    render(<FleetManagerSetup />);
    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));
    await screen.findByTestId("fleet-manager-install-command");

    expect(createToken).toHaveBeenCalledTimes(2);
  });

  it("reports a personal-token creation failure", async () => {
    createToken.mockRejectedValueOnce(new Error("failed"));
    render(<FleetManagerSetup />);

    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(/could not create the Fleet Manager personal token/i);
    expect(screen.queryByTestId("fleet-manager-install-command")).not.toBeInTheDocument();
  });
});
