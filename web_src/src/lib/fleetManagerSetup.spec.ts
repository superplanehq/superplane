import { describe, expect, it } from "bun:test";

import {
  fleetManagerInstallCommand,
  fleetManagerSecretCommand,
  generateFleetManagerValues,
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

it("keeps the token out of generated commands", () => {
  const secret = fleetManagerSecretCommand("superplane", "superplane-fleet-manager-token");
  const install = fleetManagerInstallCommand(metadata, "superplane", "values.yaml");

  expect(secret).toContain("read -rsp");
  expect(secret).not.toContain("<TOKEN>");
  expect(install).toContain("--version 0.1.0");
  expect(install).not.toContain("token");
});
