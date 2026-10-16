import { describe, expect, it } from "bun:test";
import { load } from "js-yaml";

import {
  FLEET_MANAGER_DRAFT_KEY,
  fleetsMatchConfiguration,
  fleetManagerInstallCommand,
  generateFleetManagerValues,
  loadFleetManagerDraft,
  mergeTerraformInfrastructure,
  saveFleetManagerDraft,
  terraformInstallPlaceholders,
  validateFleetManagerConfiguration,
  validateFleetSelection,
  type FleetManagerSetupMetadata,
} from "./fleetManagerSetup";

const metadata: FleetManagerSetupMetadata = {
  defaultRunnerVersion: "v0.1.0",
  defaultFleetManagerVersion: "v0.1.0",
  defaultChartVersion: "0.1.0",
  runnerReleaseUrlTemplate: "https://example/releases/runner%2F{version}",
  chartRepository: "oci://ghcr.io/superplanehq/fleet-manager-chart",
  imageRepository: "ghcr.io/superplanehq/fleet-manager",
  runnerVersions: ["v0.1.0"],
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
      imageAmd64: "runner-image-amd64",
    },
  },
};

describe("generateFleetManagerValues", () => {
  it.each(["aws", "gcp", "azure"] as const)("generates complete %s values", (provider) => {
    const values = generateFleetManagerValues({
      metadata,
      installationAdminToken: "personal-token",
      provider,
      managerId: `test-${provider}`,
      superplaneUrl: "https://superplane.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 2 } },
      fields: { imageAmd64: "runner-image-amd64" },
    });

    expect(values).toContain(`provider: ${provider}`);
    expect(values).toContain("runner-image-amd64");
    expect(values).toContain("installationAdminToken: personal-token");
    expect(values).not.toContain("credentials:");
  });

  it("quotes the runner release URL so a shell paste keeps the version placeholder", () => {
    const values = generateFleetManagerValues({
      metadata,
      installationAdminToken: "personal-token",
      provider: "gcp",
      managerId: "test-gcp",
      superplaneUrl: "https://superplane.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
      fields: {},
    });

    expect(values).toContain('releaseBaseUrl: "https://example/releases/runner%2F{version}"');
    const parsed = load(values) as { runner: { releaseBaseUrl: string } };
    expect(parsed.runner.releaseBaseUrl).toBe(metadata.runnerReleaseUrlTemplate);
  });

  it("generates capacity for each selected fleet", () => {
    const values = generateFleetManagerValues({
      metadata: {
        ...metadata,
        fleetPresets: [...metadata.fleetPresets, { ...metadata.fleetPresets[0], id: "e1-tiny-amd64" }],
      },
      installationAdminToken: "personal-token",
      provider: "aws",
      managerId: "test-aws",
      superplaneUrl: "https://superplane.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64", "e1-tiny-amd64"],
      fleetCapacities: {
        "e1-large-amd64": { warmCapacity: 2, maxCapacity: 10 },
        "e1-tiny-amd64": { warmCapacity: 0, maxCapacity: 3 },
      },
      fields: {},
    });

    const configuration = load(values) as {
      fleets: Array<{ id: string; warmCapacity: number; maxCapacity: number }>;
    };
    expect(configuration.fleets).toEqual([
      expect.objectContaining({ id: "e1-large-amd64", warmCapacity: 2, maxCapacity: 10 }),
      expect.objectContaining({ id: "e1-tiny-amd64", warmCapacity: 0, maxCapacity: 3 }),
    ]);
  });

  it("uses a machine type for each selected fleet", () => {
    const values = generateFleetManagerValues({
      metadata: {
        ...metadata,
        fleetPresets: [
          ...metadata.fleetPresets,
          {
            ...metadata.fleetPresets[0],
            id: "e1-tiny-amd64",
            spec: { ...metadata.fleetPresets[0].spec, cpuMillicores: 250, memoryMb: 1024 },
          },
        ],
      },
      installationAdminToken: "personal-token",
      provider: "aws",
      managerId: "test-aws",
      superplaneUrl: "https://cluster.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64", "e1-tiny-amd64"],
      fleetCapacities: {
        "e1-large-amd64": { warmCapacity: 0, maxCapacity: 2 },
        "e1-tiny-amd64": { warmCapacity: 0, maxCapacity: 2 },
      },
      fields: {
        "machineType.e1-large-amd64": "m8a.large",
        "machineType.e1-tiny-amd64": "t3.small",
      },
      serviceAccountName: "custom-fleet-manager",
    });

    expect(values).toContain("instanceType: m8a.large");
    expect(values).toContain("instanceType: t3.small");
    expect(values).toContain("name: custom-fleet-manager");
    expect(values).toContain("url: https://cluster.example.com");
  });

  it("does not attach a GCP service account unless one is configured", () => {
    const values = generateFleetManagerValues({
      metadata,
      installationAdminToken: "personal-token",
      provider: "gcp",
      managerId: "test-gcp",
      superplaneUrl: "https://superplane.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
      fields: {},
    });

    expect(values).not.toContain("serviceAccountEmail");
  });
});

describe("validateFleetManagerConfiguration", () => {
  const validConfiguration = {
    managerId: "my-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    superplaneUrl: "https://superplane.example.com",
    runnerVersion: "v0.1.0",
    runnerVersions: ["v0.1.0"],
    selectedFleetIds: ["e1-large-amd64"],
    fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
  };

  it.each([
    [{ selectedFleetIds: [] }, "Select at least one fleet."],
    [
      { fleetCapacities: { "e1-large-amd64": { warmCapacity: -1, maxCapacity: 1 } } },
      "Warm capacity for e1-large-amd64 must be a non-negative integer.",
    ],
    [
      { fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1.5 } } },
      "Maximum capacity for e1-large-amd64 must be a non-negative integer.",
    ],
    [
      { fleetCapacities: { "e1-large-amd64": { warmCapacity: 2, maxCapacity: 1 } } },
      "Warm capacity for e1-large-amd64 cannot exceed maximum capacity.",
    ],
    [{ superplaneUrl: "not-a-url" }, "Enter a valid SuperPlane URL."],
  ])("rejects an invalid fleet configuration", (updates, expectedError) => {
    expect(validateFleetManagerConfiguration({ ...validConfiguration, ...updates })).toContain(expectedError);
  });

  it("rejects a runner version that is empty or not listed", () => {
    const listed = { ...validConfiguration, runnerVersions: ["v0.1.0"] };
    expect(validateFleetSelection({ ...listed, runnerVersion: "" })).toContain("Select a listed Runner version.");
    expect(validateFleetSelection({ ...listed, runnerVersion: "v9.9.9" })).toContain("Select a listed Runner version.");
    expect(validateFleetSelection(listed)).not.toContain("Select a listed Runner version.");
  });
});

it("generates a bounded Helm installation command with the complete values", () => {
  const install = fleetManagerInstallCommand(metadata, "superplane", "installationAdminToken: personal-token\n");

  expect(install).toContain("cat > fleet-manager-values.yaml <<'EOF'");
  expect(install).toContain("--version 0.1.0");
  expect(install).toContain("--timeout 10m");
  expect(install).toContain("-f fleet-manager-values.yaml");
  expect(install).not.toContain("--values -");
  expect(install).toContain("installationAdminToken: personal-token");
});

it("uses a custom namespace in generated commands", () => {
  const install = fleetManagerInstallCommand(metadata, "runners", "manager:\n  id: test\n");

  expect(install).toContain("--namespace runners");
});

it("matches installed fleets by version and specification", () => {
  const draft = {
    provider: "aws" as const,
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    superplaneUrl: "https://superplane.example.com",
    runnerVersion: "v0.1.0",
    selectedFleetIds: ["e1-large-amd64"],
    fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
    fields: {},
  };
  const installed = [{ id: "e1-large-amd64", runnerVersion: "v0.1.0", spec: metadata.fleetPresets[0].spec }];

  expect(fleetsMatchConfiguration(installed, draft, metadata.fleetPresets)).toBe(true);
  expect(fleetsMatchConfiguration([{ ...installed[0], runnerVersion: "v0.0.9" }], draft, metadata.fleetPresets)).toBe(
    false,
  );
  expect(fleetsMatchConfiguration(installed, { ...draft, selectedFleetIds: [] }, metadata.fleetPresets)).toBe(false);
});

it("replaces saved infrastructure with the current Terraform values", () => {
  const draft = {
    provider: "aws" as const,
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    superplaneUrl: "https://old.example.com",
    runnerVersion: "v0.1.0",
    selectedFleetIds: [] as string[],
    fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
    fields: { region: "eu-west-1", imageAmd64: "ami-old", "machineType.e1-large-amd64": "t3.small" },
    fieldsByProvider: {
      aws: { region: "eu-west-1", imageAmd64: "ami-old", "machineType.e1-large-amd64": "t3.small" },
    },
  };

  const merged = mergeTerraformInfrastructure(draft, metadata);
  const values = generateFleetManagerValues({
    metadata,
    ...merged,
    installationAdminToken: "personal-token",
    selectedFleetIds: ["e1-large-amd64"],
  });

  expect(values).toContain("region: us-east-1");
  expect(values).toContain("ami: runner-image-amd64");
  expect(values).toContain("instanceType: t3.small");
  expect(values).toContain("url: https://superplane.example.com");
  expect(merged.fieldsByProvider?.aws?.region).toBe("us-east-1");
  expect(mergeTerraformInfrastructure({ ...draft, provider: "gcp" }, metadata).fields.region).toBe("eu-west-1");
  expect(
    mergeTerraformInfrastructure(draft, {
      ...metadata,
      infrastructureDefaults: { ...metadata.infrastructureDefaults, source: "manual" },
    }).fields.region,
  ).toBe("eu-west-1");
});

it.each([
  [
    "aws",
    {
      superplaneUrl: "https://superplane.example.com",
      region: "us-east-1",
      subnetIds: ["subnet-a", "subnet-b"],
      securityGroupIds: ["sg-runner"],
      imageAmd64: "ami-amd64",
      imageArm64: "",
      fleetManagerIdentity: "arn:aws:iam::123456789012:role/fleet-manager",
      runnerIdentity: "runner-profile",
    },
  ],
  [
    "gcp",
    {
      superplaneUrl: "https://superplane.fezta.app",
      projectId: "self-hosted-1",
      zones: ["us-central1-a"],
      subnetwork: "projects/self-hosted-1/regions/us-central1/subnetworks/superplane-runners",
      networkTags: ["superplane-runner"],
      imageAmd64: "projects/self-hosted-1/global/images/family/superplane-runner-amd64",
      imageArm64: "",
      fleetManagerIdentity: "superplane-fleet-mgr@self-hosted-1.iam.gserviceaccount.com",
    },
  ],
  [
    "azure",
    {
      superplaneUrl: "https://superplane.example.com",
      subscriptionId: "00000000-0000-0000-0000-000000000000",
      resourceGroup: "superplane-runners",
      location: "eastus",
      zones: ["1"],
      subnetId:
        "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/providers/Microsoft.Network/virtualNetworks/runners/subnets/runners",
      securityGroupIds:
        "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/providers/Microsoft.Network/networkSecurityGroups/runners",
      imageAmd64:
        "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/providers/Microsoft.Compute/galleries/runners/images/amd64",
      imageArm64: "",
      runnerIdentity:
        "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/providers/Microsoft.ManagedIdentity/userAssignedIdentities/runner",
      fleetManagerIdentity: "11111111-1111-1111-1111-111111111111",
    },
  ],
] as const)("writes a %s Terraform install with no placeholders", (provider, fields) => {
  const terraformMetadata: FleetManagerSetupMetadata = {
    ...metadata,
    infrastructureDefaults: {
      source: "terraform",
      provider,
      managerId: "superplane",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      fields: Object.fromEntries(
        Object.entries(fields).map(([key, value]) => [key, Array.isArray(value) ? [...value] : value]),
      ),
    },
  };
  const draft = mergeTerraformInfrastructure(
    {
      provider,
      managerId: "my-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      superplaneUrl: "http://localhost:8000",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 10 } },
      fields: {},
    },
    terraformMetadata,
  );
  const values = generateFleetManagerValues({
    metadata: terraformMetadata,
    ...draft,
    installationAdminToken: "personal-token",
  });

  expect(values).not.toMatch(/<[A-Z0-9_]+>/);
  expect(terraformInstallPlaceholders(draft, terraformMetadata)).toEqual([]);
});

it("writes an empty Azure zone list when Terraform sets no zones", () => {
  const terraformMetadata: FleetManagerSetupMetadata = {
    ...metadata,
    infrastructureDefaults: {
      source: "terraform",
      provider: "azure",
      managerId: "superplane",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      fields: {
        superplaneUrl: "https://superplane.example.com",
        subscriptionId: "00000000-0000-0000-0000-000000000000",
        resourceGroup: "superplane-runners",
        location: "eastus",
        zones: [],
        subnetId: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/subnets/runners",
        securityGroupIds:
          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/networkSecurityGroups/runners",
        imageAmd64: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/images/amd64",
        imageArm64: "",
        runnerIdentity:
          "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/runners/userAssignedIdentities/runner",
        fleetManagerIdentity: "11111111-1111-1111-1111-111111111111",
      },
    },
  };
  const draft = mergeTerraformInfrastructure(
    {
      provider: "azure",
      managerId: "my-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      superplaneUrl: "http://localhost:8000",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 10 } },
      fields: {},
    },
    terraformMetadata,
  );
  const values = generateFleetManagerValues({
    metadata: terraformMetadata,
    ...draft,
    installationAdminToken: "personal-token",
  });

  expect(load(values)).toMatchObject({ fleets: [{ azure: { zones: [] } }] });
  expect(terraformInstallPlaceholders(draft, terraformMetadata)).toEqual([]);
});

it("blocks a Terraform install that would still contain a placeholder", () => {
  const draft = mergeTerraformInfrastructure(
    {
      provider: "aws",
      managerId: "test-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      superplaneUrl: "https://old.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
      fields: {},
    },
    metadata,
  );

  expect(terraformInstallPlaceholders(draft, metadata)).toEqual([
    "Apply Terraform again before you install Fleet Manager. A required value is missing.",
  ]);
  expect(
    terraformInstallPlaceholders(draft, {
      ...metadata,
      infrastructureDefaults: { ...metadata.infrastructureDefaults, source: "manual" },
    }),
  ).toEqual([]);
});

it("uses a browser key that stores only the non-secret draft", () => {
  const draft = {
    provider: "aws" as const,
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    superplaneUrl: "https://superplane.example.com",
    runnerVersion: "v0.1.0",
    selectedFleetIds: ["e1-large-amd64"],
    fleetCapacities: { "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } },
    fields: { region: "us-east-1" },
  };

  saveFleetManagerDraft(draft);

  expect(loadFleetManagerDraft()).toEqual({
    ...draft,
    fieldsByProvider: { aws: draft.fields },
  });
  expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("INSTALLATION_ADMIN_TOKEN");
});

it("migrates global capacities and removes unknown values from a stored draft", () => {
  window.localStorage.setItem(
    FLEET_MANAGER_DRAFT_KEY,
    JSON.stringify({
      provider: "aws",
      managerId: "test-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      superplaneUrl: "https://superplane.example.com",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      warmCapacity: 0,
      maxCapacity: 1,
      fields: { region: "us-east-1" },
      plaintextToken: "must-not-survive",
    }),
  );

  const draft = loadFleetManagerDraft();
  expect(draft).not.toHaveProperty("plaintextToken");
  expect(draft?.fleetCapacities).toEqual({ "e1-large-amd64": { warmCapacity: 0, maxCapacity: 1 } });
});
