import { describe, expect, it } from "bun:test";

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
    fields: { region: "us-east-1", subnetIds: ["subnet-a"] },
  },
  credentials: {
    existingSecret: "superplane-fleet-manager-token",
    tokenKey: "INSTALLATION_ADMIN_TOKEN",
    autoProvisioned: true,
  },
};

describe("generateFleetManagerValues", () => {
  it.each([
    ["aws", "<PUBLIC_AMI_ID_FOR_REGION_AND_ARCH>"],
    ["gcp", "<PUBLIC_GCP_IMAGE_OR_FAMILY>"],
    ["azure", "<PUBLIC_AZURE_IMAGE_VERSION_ID>"],
  ] as const)("generates %s values with a public image placeholder", (provider, placeholder) => {
    const values = generateFleetManagerValues({
      metadata,
      provider,
      managerId: `test-${provider}`,
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      warmCapacity: 0,
      maxCapacity: 2,
      fields: {},
    });

    expect(values).toContain(`provider: ${provider}`);
    expect(values).toContain(placeholder);
    expect(values).not.toContain("INSTALLATION_ADMIN_TOKEN:");
    expect(values).not.toContain("token:");
  });

  it("generates more than one selected fleet", () => {
    const values = generateFleetManagerValues({
      metadata: {
        ...metadata,
        fleetPresets: [...metadata.fleetPresets, { ...metadata.fleetPresets[0], id: "e1-tiny-amd64" }],
      },
      provider: "aws",
      managerId: "test-aws",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64", "e1-tiny-amd64"],
      warmCapacity: 0,
      maxCapacity: 2,
      fields: {},
    });

    expect(values).toContain("id: e1-large-amd64");
    expect(values).toContain("id: e1-tiny-amd64");
  });

  it("uses an architecture-specific machine type for mixed fleets", () => {
    const values = generateFleetManagerValues({
      metadata: {
        ...metadata,
        fleetPresets: [
          ...metadata.fleetPresets,
          {
            ...metadata.fleetPresets[0],
            id: "e1-large-arm64",
            spec: { ...metadata.fleetPresets[0].spec, architecture: "arm64" },
          },
        ],
      },
      provider: "aws",
      managerId: "test-aws",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64", "e1-large-arm64"],
      warmCapacity: 0,
      maxCapacity: 2,
      fields: { machineTypeAmd64: "m8a.large", machineTypeArm64: "m8g.large" },
      serviceAccountName: "custom-fleet-manager",
    });

    expect(values).toContain("instanceType: m8a.large");
    expect(values).toContain("instanceType: m8g.large");
    expect(values).toContain("name: custom-fleet-manager");
  });

  it("does not attach a GCP service account unless one is configured", () => {
    const values = generateFleetManagerValues({
      metadata,
      provider: "gcp",
      managerId: "test-gcp",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      warmCapacity: 0,
      maxCapacity: 1,
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
    runnerVersion: "v0.1.0",
    selectedFleetIds: ["e1-large-amd64"],
    warmCapacity: 0,
    maxCapacity: 1,
  };

  it.each([
    [{ selectedFleetIds: [] }, "Select at least one fleet."],
    [{ warmCapacity: -1 }, "Warm capacity must be a non-negative integer."],
    [{ maxCapacity: 1.5 }, "Maximum capacity must be a non-negative integer."],
    [{ warmCapacity: 2, maxCapacity: 1 }, "Warm capacity cannot exceed maximum capacity."],
  ])("rejects an invalid fleet configuration", (updates, expectedError) => {
    expect(validateFleetManagerConfiguration({ ...validConfiguration, ...updates })).toContain(expectedError);
  });
});

it("generates a bounded Helm installation command", () => {
  const install = fleetManagerInstallCommand(metadata, "superplane", "values.yaml");

  expect(install).toContain("--version 0.1.0");
  expect(install).toContain("--wait --timeout 10m");
  expect(install).not.toContain("token");
});

it("uses a custom namespace in generated commands", () => {
  const install = fleetManagerInstallCommand(metadata, "runners", "values.yaml");

  expect(install).toContain("--namespace runners");
});

it("matches installed fleets by version and specification", () => {
  const draft = {
    provider: "aws" as const,
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    runnerVersion: "v0.1.0",
    selectedFleetIds: ["e1-large-amd64"],
    warmCapacity: 0,
    maxCapacity: 1,
    fields: {},
  };
  const installed = [{ id: "e1-large-amd64", runnerVersion: "v0.1.0", spec: metadata.fleetPresets[0].spec }];

  expect(fleetsMatchConfiguration(installed, draft, metadata.fleetPresets)).toBe(true);
  expect(fleetsMatchConfiguration([{ ...installed[0], runnerVersion: "v0.0.9" }], draft, metadata.fleetPresets)).toBe(
    false,
  );
});

it("uses a browser key that stores only the non-secret draft", () => {
  const draft = {
    provider: "aws" as const,
    managerId: "test-cluster",
    namespace: "superplane",
    serviceAccountName: "superplane-fleet-manager",
    runnerVersion: "v0.1.0",
    selectedFleetIds: ["e1-large-amd64"],
    warmCapacity: 0,
    maxCapacity: 1,
    fields: { region: "us-east-1" },
  };

  saveFleetManagerDraft(draft);

  expect(loadFleetManagerDraft()).toEqual(draft);
  expect(window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY)).not.toContain("INSTALLATION_ADMIN_TOKEN");
});

it("removes unknown values from a stored Fleet Manager draft", () => {
  window.localStorage.setItem(
    FLEET_MANAGER_DRAFT_KEY,
    JSON.stringify({
      provider: "aws",
      managerId: "test-cluster",
      namespace: "superplane",
      serviceAccountName: "superplane-fleet-manager",
      runnerVersion: "v0.1.0",
      selectedFleetIds: ["e1-large-amd64"],
      warmCapacity: 0,
      maxCapacity: 1,
      fields: { region: "us-east-1" },
      plaintextToken: "must-not-survive",
    }),
  );

  expect(loadFleetManagerDraft()).not.toHaveProperty("plaintextToken");
});
