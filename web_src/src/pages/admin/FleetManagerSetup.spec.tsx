import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { FleetManagerWizardScreen } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { FLEET_MANAGER_DRAFT_KEY } from "@/lib/fleetManagerSetup";
import { FleetManagerSetup } from "./FleetManagerSetup";

const { createToken } = vi.hoisted(() => ({ createToken: vi.fn() }));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Acme" }] }),
}));

vi.mock("@/hooks/useUserTokens", () => ({
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
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
  stableRunnerVersion: "v0.2.0",
  installationType: "",
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

function stubSetup(body: unknown = metadata) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/admin/api/installation/fleet-manager/setup")) return jsonResponse(body);
      return new Response("", { status: 404 });
    }),
  );
}

function FleetWizard() {
  const setup = useFleetManagerSetup(true, "org-1");
  return (
    <FleetManagerWizardScreen
      step="fleets"
      setup={setup}
      onNext={() => undefined}
      onFinish={() => undefined}
      onFinishLater={() => undefined}
    />
  );
}

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

  it("starts a new fleet at warm 1 and max 10", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Warm capacity")).toHaveValue(1);
    expect(screen.getByLabelText("Maximum capacity")).toHaveValue(10);

    fireEvent.click(screen.getByRole("button", { name: "Generate install command" }));
    const command = await screen.findByTestId("fleet-manager-install-command");
    expect(command).toHaveTextContent("warmCapacity: 1");
    expect(command).toHaveTextContent("maxCapacity: 10");
  });

  it("keeps a saved warm 0 and max 1 capacity", async () => {
    window.localStorage.setItem(
      FLEET_MANAGER_DRAFT_KEY,
      JSON.stringify({
        provider: "aws",
        managerId: "terraform-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: "https://superplane.example.com",
        runnerVersion: "v0.2.0",
        selectedFleetIds: ["e1-large-amd64"],
        fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
        fields: { region: "us-east-1", imageAmd64: "ami-amd64" },
      }),
    );

    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Warm capacity")).toHaveValue(0);
    expect(screen.getByLabelText("Maximum capacity")).toHaveValue(1);
  });

  it("uses the Provider, Fleets, and Install sequence", async () => {
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("heading", { name: "Choose where runners run" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Configure the default fleet" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Install Fleet Manager" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /cloud infrastructure/i })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Fleet")).toHaveValue("e1-large-amd64");
    expect(screen.getByRole("combobox", { name: "Machine type" })).toHaveTextContent(
      "m8a.large · 2 vCPU · 8 GB memory",
    );
    expect(screen.getByText(/Idle machines kept ready before a task starts/)).toBeInTheDocument();
    expect(screen.getByText(/The most machines this fleet can run at one time/)).toBeInTheDocument();
    expect(screen.queryByText("e1-large-arm64")).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Runner version" })).not.toBeInTheDocument();
  });

  it("shows the single-VM runner command instead of the chart", async () => {
    const user = userEvent.setup();
    stubSetup({ ...metadata, installationType: "single-host" });
    render(<FleetManagerSetup />);

    await user.click(await screen.findByRole("button", { name: "This VM" }));
    expect(screen.queryByRole("heading", { name: "Configure the default fleet" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Generate install command" }));
    expect(await screen.findByText("./enable-runners.sh 'one-time-personal-token'")).toBeInTheDocument();
    expect(screen.queryByText(/helm upgrade --install fleet-manager/)).not.toBeInTheDocument();
  });

  it("switches the machine type list when the cloud changes", async () => {
    const user = userEvent.setup();
    render(<FleetManagerSetup />);

    await user.click(await screen.findByRole("button", { name: "GCP" }));
    expect(screen.getByRole("combobox", { name: "Machine type" })).toHaveTextContent(
      "e2-standard-4 · 4 vCPU · 16 GB memory",
    );

    await user.click(screen.getByRole("combobox", { name: "Machine type" }));
    expect(screen.getByRole("option", { name: "e2-standard-2 · 2 vCPU · 8 GB memory" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /m8a.large/ })).not.toBeInTheDocument();
  });

  it("pins the setup runner version and ignores a saved version choice", async () => {
    window.localStorage.setItem(
      FLEET_MANAGER_DRAFT_KEY,
      JSON.stringify({
        provider: "aws",
        managerId: "terraform-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: "https://superplane.example.com",
        runnerVersion: "v0.1.0",
        selectedFleetIds: ["e1-large-amd64"],
        fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
        fields: { region: "us-east-1", imageAmd64: "ami-amd64" },
      }),
    );

    render(<FleetManagerSetup />);

    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));
    expect(await screen.findByTestId("fleet-manager-install-command")).toHaveTextContent("runnerVersion: v0.2.0");
    expect(screen.queryByRole("combobox", { name: "Runner version" })).not.toBeInTheDocument();
  });

  it("blocks installation when no runner version is available", async () => {
    stubSetup({ ...metadata, defaultRunnerVersion: "", runnerVersions: [] });
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("button", { name: "Generate install command" })).toBeDisabled();
    expect(screen.getAllByRole("alert")[0]).toHaveTextContent("SuperPlane could not determine a Runner version.");
  });

  it("keeps the wizard install action disabled when no runner version is available", async () => {
    stubSetup({ ...metadata, defaultRunnerVersion: "", runnerVersions: [] });
    render(<FleetWizard />);

    expect(await screen.findByRole("button", { name: "Generate install command" })).toBeDisabled();
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

    expect(await screen.findByLabelText("Fleet")).toHaveValue("e1-large-amd64");
    expect(screen.queryByText(/Add the amd64 runner image through Terraform/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Generate install command" }));

    const command = await screen.findByTestId("fleet-manager-install-command");
    expect(command).toHaveTextContent("ami: <RUNNER_IMAGE>");
  });

  it("keeps a saved custom machine type and ignores other fleets", async () => {
    window.localStorage.setItem(
      FLEET_MANAGER_DRAFT_KEY,
      JSON.stringify({
        provider: "aws",
        managerId: "terraform-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: "https://superplane.example.com",
        runnerVersion: "v0.2.0",
        selectedFleetIds: ["e1-large-arm64", "e1-tiny-amd64"],
        fleetCapacities: { "e1-large-arm64": { warmCapacity: 0, maxCapacity: 1 } },
        fields: { region: "us-east-1", "machineType.e1-large-amd64": "t3.small" },
      }),
    );

    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Fleet")).toHaveValue("e1-large-amd64");
    expect(screen.queryByText("e1-large-arm64")).not.toBeInTheDocument();
    expect(screen.queryByText("e1-tiny-amd64")).not.toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Machine type" })).toHaveTextContent("t3.small");
  });

  it("keeps the listed machine type when the other type is cleared", async () => {
    const user = userEvent.setup();
    render(<FleetManagerSetup />);

    expect(await screen.findByRole("combobox", { name: "Machine type" })).toHaveTextContent(
      "m8a.large · 2 vCPU · 8 GB memory",
    );
    await user.click(screen.getByRole("combobox", { name: "Machine type" }));
    await user.click(screen.getByRole("option", { name: "Other machine type" }));

    const customType = screen.getByRole("textbox", { name: "Other machine type" });
    await user.type(customType, "c5.large");
    await user.clear(customType);
    fireEvent.blur(customType);

    expect(screen.getByRole("combobox", { name: "Machine type" })).toHaveTextContent(
      "m8a.large · 2 vCPU · 8 GB memory",
    );
  });

  it("keeps an entered machine type after a listed type is selected", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem(
      FLEET_MANAGER_DRAFT_KEY,
      JSON.stringify({
        provider: "aws",
        managerId: "terraform-cluster",
        namespace: "superplane",
        serviceAccountName: "superplane-fleet-manager",
        superplaneUrl: "https://superplane.example.com",
        runnerVersion: "v0.2.0",
        selectedFleetIds: ["e1-large-amd64"],
        fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
        fields: { region: "us-east-1", "machineType.e1-large-amd64": "t3.small" },
      }),
    );

    render(<FleetManagerSetup />);

    await user.click(await screen.findByRole("combobox", { name: "Machine type" }));
    expect(screen.getByRole("option", { name: "t3.small" })).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: "m8a.large · 2 vCPU · 8 GB memory" }));

    await user.click(screen.getByRole("combobox", { name: "Machine type" }));
    expect(screen.getByRole("option", { name: "t3.small" })).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: "Other machine type" }));

    const customType = screen.getByRole("textbox", { name: "Other machine type" });
    await user.type(customType, "c5.large");
    fireEvent.blur(customType);

    await user.click(screen.getByRole("combobox", { name: "Machine type" }));
    expect(screen.getByRole("option", { name: "c5.large" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "t3.small" })).toBeInTheDocument();
  });

  it("blocks install when Terraform has no amd64 runner image", async () => {
    stubSetup({
      ...metadata,
      infrastructureDefaults: {
        ...metadata.infrastructureDefaults,
        fields: { ...metadata.infrastructureDefaults.fields, imageAmd64: "" },
      },
    });
    render(<FleetManagerSetup />);

    expect(await screen.findByLabelText("Fleet")).toHaveValue("e1-large-amd64");
    expect(screen.getByText(/Add the amd64 runner image through Terraform to use this fleet/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Generate install command" })).toBeDisabled();
  });

  it("creates one personal token and puts it only in the values file", async () => {
    render(<FleetManagerSetup />);

    fireEvent.click(await screen.findByRole("button", { name: "Generate install command" }));

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    const command = await screen.findByTestId("fleet-manager-install-command");
    expect(createToken).toHaveBeenCalledTimes(1);
    expect(createToken).toHaveBeenCalledWith({ name: "Fleet Manager" });
    expect(command).toHaveTextContent("installationAdminToken: one-time-personal-token");
    expect(command).toHaveTextContent("ami: ami-amd64");
    expect(command).toHaveTextContent("runnerVersion: v0.2.0");
    expect(command).toHaveTextContent("cpuMillicores: 2000");
    expect(command).toHaveTextContent("-f fleet-manager-values.yaml");
    expect(command).not.toHaveTextContent("kubectl create secret");
    expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("one-time-personal-token");

    expect(command).toHaveTextContent("cat > fleet-manager-values.yaml <<'EOF'");
    fireEvent.click(screen.getByRole("button", { name: "Copy values command" }));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(expect.stringContaining("cat > fleet-manager-values.yaml <<'EOF'")),
    );
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining("installationAdminToken: one-time-personal-token"));
    fireEvent.click(screen.getByRole("button", { name: "Copy install command" }));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(expect.stringContaining("helm upgrade --install fleet-manager")),
    );
    expect(writeText.mock.calls.some(([text]) => String(text).includes("kubectl"))).toBe(false);

    fireEvent.change(screen.getByLabelText("Maximum capacity"), { target: { value: "3" } });
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
