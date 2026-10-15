import { describe, expect, it } from "bun:test";
import { load } from "js-yaml";

import {
  FLEET_MANAGER_DRAFT_KEY,
  fleetsMatchConfiguration,
  fleetManagerInstallCommand,
  generateFleetManagerValues,
  loadFleetManagerDraft,
  saveFleetManagerDraft,
  validateFleetManagerConfiguration,
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
});

it("generates a bounded Helm installation command with the complete values", () => {
  const install = fleetManagerInstallCommand(metadata, "superplane", "installationAdminToken: personal-token\n");

  expect(install).toContain("--version 0.1.0");
  expect(install).toContain("--wait --timeout 10m");
  expect(install).toContain("--values - --wait --timeout 10m <<'FLEET_MANAGER_VALUES'");
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
