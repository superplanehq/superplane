import { dump } from "js-yaml";

export type FleetManagerProvider = "aws" | "gcp" | "azure";

export type FleetSpec = {
  operatingSystem: string;
  architecture: "amd64" | "arm64";
  cpuMillicores: number;
  memoryMb: number;
  diskGb: number;
  capabilities: string[];
};

export type FleetManagerSetupMetadata = {
  defaultRunnerVersion: string;
  defaultFleetManagerVersion: string;
  defaultChartVersion: string;
  runnerReleaseUrlTemplate: string;
  chartRepository: string;
  imageRepository: string;
  fleetPresets: Array<{ id: string; spec: FleetSpec }>;
  infrastructureDefaults: {
    source: "terraform" | "manual";
    provider: FleetManagerProvider | null;
    managerId: string;
    namespace: string;
    serviceAccountName: string;
    fields: Record<string, string | string[]>;
  };
  credentials: {
    existingSecret: string;
    tokenKey: string;
    autoProvisioned: boolean;
  };
};

export type FleetManagerProviderFields = Record<string, string>;

export type FleetCapacity = {
  warmCapacity: number;
  maxCapacity: number;
};

export type FleetManagerDraft = {
  provider: FleetManagerProvider;
  managerId: string;
  namespace: string;
  serviceAccountName: string;
  superplaneUrl: string;
  runnerVersion: string;
  selectedFleetIds: string[];
  fleetCapacities: Record<string, FleetCapacity>;
  fields: FleetManagerProviderFields;
};

export type InstalledFleet = {
  id?: string;
  runnerVersion?: string;
  spec?: Partial<FleetSpec>;
};

type GenerateValuesInput = Omit<FleetManagerDraft, "namespace" | "serviceAccountName"> & {
  metadata: FleetManagerSetupMetadata;
  serviceAccountName?: string;
};

export type FleetManagerConfigurationInput = Omit<FleetManagerDraft, "provider" | "fields">;
type FleetSelectionInput = Pick<FleetManagerDraft, "runnerVersion" | "selectedFleetIds" | "fleetCapacities">;
type FleetInfrastructureInput = Pick<
  FleetManagerDraft,
  "managerId" | "namespace" | "serviceAccountName" | "superplaneUrl"
>;

const LEGACY_FLEET_MANAGER_DRAFT_KEY = "superplane.fleet-manager-setup.v1";
export const FLEET_MANAGER_DRAFT_KEY = "superplane.fleet-manager-setup.v2";
export const DEFAULT_FLEET_CAPACITY: FleetCapacity = { warmCapacity: 0, maxCapacity: 1 };

const splitList = (value: string | undefined, fallback: string) =>
  (value || fallback)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

const machineType = (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields, fallback: string) =>
  fields[`machineType.${fleetId}`] ||
  fields[`machineType${spec.architecture === "amd64" ? "Amd64" : "Arm64"}`] ||
  fallback;

const providerConfiguration = {
  aws: (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    aws: {
      region: fields.region || "<AWS_REGION>",
      ami: "<PUBLIC_AMI_ID_FOR_REGION_AND_ARCH>",
      instanceType: machineType(fleetId, spec, fields, "<INSTANCE_TYPE>"),
      architecture: spec.architecture,
      subnetIds: splitList(fields.subnetIds, "<SUBNET_ID>"),
      securityGroupIds: splitList(fields.securityGroupIds, "<SECURITY_GROUP_ID>"),
      iamInstanceProfile: fields.runnerIdentity || "<INSTANCE_PROFILE>",
      volumeSizeGb: spec.diskGb,
    },
  }),
  gcp: (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    gcp: {
      projectId: fields.projectId || "<GCP_PROJECT_ID>",
      zones: splitList(fields.zones, "<GCP_ZONE>"),
      machineType: machineType(fleetId, spec, fields, "<MACHINE_TYPE>"),
      image: "<PUBLIC_GCP_IMAGE_OR_FAMILY>",
      architecture: spec.architecture,
      subnetwork: fields.subnetwork || "<RUNNER_SUBNETWORK>",
      ...(fields.runnerIdentity ? { serviceAccountEmail: fields.runnerIdentity } : {}),
      networkTags: splitList(fields.networkTags, "<RUNNER_NETWORK_TAG>"),
      diskSizeGb: spec.diskGb,
    },
  }),
  azure: (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    azure: {
      subscriptionId: fields.subscriptionId || "<AZURE_SUBSCRIPTION_ID>",
      resourceGroup: fields.resourceGroup || "<RESOURCE_GROUP>",
      location: fields.location || "<AZURE_LOCATION>",
      imageId: "<PUBLIC_AZURE_IMAGE_VERSION_ID>",
      vmSize: machineType(fleetId, spec, fields, "<VM_SIZE>"),
      architecture: spec.architecture,
      subnetId: fields.subnetId || "<RUNNER_SUBNET_ID>",
      networkSecurityGroupId: fields.securityGroupIds || "<NETWORK_SECURITY_GROUP_ID>",
      identityId: fields.runnerIdentity || "<RUNNER_IDENTITY_ID>",
      zones: splitList(fields.zones, "<AZURE_ZONE>"),
      diskSizeGb: spec.diskGb,
    },
  }),
};

export function generateFleetManagerValues(input: GenerateValuesInput): string {
  const selected = new Set(input.selectedFleetIds);
  const fleets = input.metadata.fleetPresets
    .filter((preset) => selected.has(preset.id))
    .map((preset) => {
      const capacity = input.fleetCapacities[preset.id] ?? DEFAULT_FLEET_CAPACITY;
      return {
        id: preset.id,
        runnerVersion: input.runnerVersion,
        warmCapacity: capacity.warmCapacity,
        maxCapacity: capacity.maxCapacity,
        spec: preset.spec,
        provider: input.provider,
        ...providerConfiguration[input.provider](preset.id, preset.spec, input.fields),
      };
    });

  const workloadIdentity =
    input.provider === "aws"
      ? {
          annotations: {
            "eks.amazonaws.com/role-arn": input.fields.fleetManagerIdentity || "<FLEET_MANAGER_ROLE_ARN>",
          },
        }
      : input.provider === "gcp"
        ? {
            annotations: {
              "iam.gke.io/gcp-service-account":
                input.fields.fleetManagerIdentity || "<FLEET_MANAGER_SERVICE_ACCOUNT_EMAIL>",
            },
          }
        : {
            annotations: {
              "azure.workload.identity/client-id": input.fields.fleetManagerIdentity || "<FLEET_MANAGER_CLIENT_ID>",
            },
          };

  return dump(
    {
      manager: { id: input.managerId },
      superplane: { url: input.superplaneUrl },
      credentials: {
        existingSecret: input.metadata.credentials.existingSecret,
        tokenKey: input.metadata.credentials.tokenKey,
      },
      runner: { releaseBaseUrl: input.metadata.runnerReleaseUrlTemplate },
      image: {
        repository: input.metadata.imageRepository,
        tag: input.metadata.defaultFleetManagerVersion,
        pullPolicy: "IfNotPresent",
      },
      serviceAccount: {
        create: true,
        name: input.serviceAccountName || "superplane-fleet-manager",
        ...workloadIdentity,
      },
      podLabels: input.provider === "azure" ? { "azure.workload.identity/use": "true" } : {},
      fleets,
    },
    { noRefs: true, lineWidth: 120, quotingType: '"' },
  );
}

export function validateFleetInfrastructure(input: FleetInfrastructureInput): string[] {
  const errors: string[] = [];
  if (!/^[a-z0-9][a-z0-9_-]{0,62}$/.test(input.managerId)) {
    errors.push("Enter a valid Fleet Manager ID.");
  }
  if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(input.namespace)) {
    errors.push("Enter a valid Kubernetes namespace.");
  }
  if (!/^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$/.test(input.serviceAccountName)) {
    errors.push("Enter a valid service account name.");
  }
  try {
    const url = new URL(input.superplaneUrl);
    if (url.protocol !== "https:" && url.protocol !== "http:") errors.push("Enter a valid SuperPlane URL.");
  } catch {
    errors.push("Enter a valid SuperPlane URL.");
  }
  return errors;
}

export function validateFleetSelection(input: FleetSelectionInput): string[] {
  const errors: string[] = [];
  if (!/^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(input.runnerVersion)) {
    errors.push("Enter a valid immutable Runner version.");
  }
  if (input.selectedFleetIds.length === 0) {
    errors.push("Select at least one fleet.");
  }
  for (const fleetId of input.selectedFleetIds) {
    const capacity = input.fleetCapacities[fleetId] ?? DEFAULT_FLEET_CAPACITY;
    if (!Number.isInteger(capacity.warmCapacity) || capacity.warmCapacity < 0) {
      errors.push(`Warm capacity for ${fleetId} must be a non-negative integer.`);
    }
    if (!Number.isInteger(capacity.maxCapacity) || capacity.maxCapacity < 0) {
      errors.push(`Maximum capacity for ${fleetId} must be a non-negative integer.`);
    }
    if (capacity.warmCapacity > capacity.maxCapacity) {
      errors.push(`Warm capacity for ${fleetId} cannot exceed maximum capacity.`);
    }
  }
  return errors;
}

export function validateFleetManagerConfiguration(input: FleetManagerConfigurationInput): string[] {
  return [...validateFleetSelection(input), ...validateFleetInfrastructure(input)];
}

export function fleetManagerInstallCommand(
  metadata: FleetManagerSetupMetadata,
  namespace: string,
  valuesFile: string,
): string {
  return `helm upgrade --install fleet-manager ${metadata.chartRepository} --version ${metadata.defaultChartVersion} --namespace ${namespace} --create-namespace --values ${valuesFile} --wait --timeout 10m`;
}

export function normalizeInfrastructureFields(fields: Record<string, string | string[]>): FleetManagerProviderFields {
  return Object.fromEntries(
    Object.entries(fields).map(([key, value]) => [key, Array.isArray(value) ? value.join(", ") : value]),
  );
}

export function fleetsMatchConfiguration(
  installed: InstalledFleet[],
  draft: FleetManagerDraft,
  presets: FleetManagerSetupMetadata["fleetPresets"],
): boolean {
  const byID = new Map(installed.map((fleet) => [fleet.id, fleet]));
  const presetByID = new Map(presets.map((preset) => [preset.id, preset]));

  return draft.selectedFleetIds.every((id) => {
    const fleet = byID.get(id);
    const preset = presetByID.get(id);
    if (!fleet || !preset || fleet.runnerVersion !== draft.runnerVersion) return false;

    return (
      fleet.spec?.operatingSystem === preset.spec.operatingSystem &&
      fleet.spec.architecture === preset.spec.architecture &&
      fleet.spec.cpuMillicores === preset.spec.cpuMillicores &&
      fleet.spec.memoryMb === preset.spec.memoryMb &&
      fleet.spec.diskGb === preset.spec.diskGb &&
      JSON.stringify(fleet.spec.capabilities ?? []) === JSON.stringify(preset.spec.capabilities)
    );
  });
}

export function saveFleetManagerDraft(draft: FleetManagerDraft): void {
  window.localStorage.removeItem(LEGACY_FLEET_MANAGER_DRAFT_KEY);
  window.localStorage.setItem(FLEET_MANAGER_DRAFT_KEY, JSON.stringify(draft));
}

type RequiredDraftStrings = Pick<
  FleetManagerDraft,
  "managerId" | "namespace" | "serviceAccountName" | "superplaneUrl" | "runnerVersion"
>;

function hasRequiredDraftStrings(
  draft: Partial<FleetManagerDraft>,
): draft is Partial<FleetManagerDraft> & RequiredDraftStrings {
  return [draft.managerId, draft.namespace, draft.serviceAccountName, draft.superplaneUrl, draft.runnerVersion].every(
    (value) => typeof value === "string" && value.length > 0,
  );
}

export function loadFleetManagerDraft(): FleetManagerDraft | null {
  const stored = window.localStorage.getItem(FLEET_MANAGER_DRAFT_KEY);
  if (!stored) return null;

  try {
    const draft = JSON.parse(stored) as Partial<FleetManagerDraft> & {
      warmCapacity?: unknown;
      maxCapacity?: unknown;
    };
    if (!draft.provider || !["aws", "gcp", "azure"].includes(draft.provider)) return null;
    if (!hasRequiredDraftStrings(draft)) return null;
    if (!Array.isArray(draft.selectedFleetIds) || !draft.selectedFleetIds.every((id) => typeof id === "string"))
      return null;
    if (typeof draft.fields !== "object" || draft.fields === null) return null;
    const fields = Object.fromEntries(
      Object.entries(draft.fields).filter((entry): entry is [string, string] => typeof entry[1] === "string"),
    );
    const fleetCapacities = storedFleetCapacities(draft);
    return {
      provider: draft.provider,
      managerId: draft.managerId,
      namespace: draft.namespace,
      serviceAccountName: draft.serviceAccountName,
      superplaneUrl: draft.superplaneUrl,
      runnerVersion: draft.runnerVersion,
      selectedFleetIds: draft.selectedFleetIds,
      fleetCapacities,
      fields,
    };
  } catch {
    return null;
  }
}

function storedFleetCapacities(
  draft: Partial<FleetManagerDraft> & { warmCapacity?: unknown; maxCapacity?: unknown },
): Record<string, FleetCapacity> {
  const capacities = Object.fromEntries(
    Object.entries(draft.fleetCapacities ?? {}).filter(
      (entry): entry is [string, FleetCapacity] =>
        typeof entry[1] === "object" &&
        entry[1] !== null &&
        typeof entry[1].warmCapacity === "number" &&
        typeof entry[1].maxCapacity === "number",
    ),
  );
  const legacyCapacity =
    typeof draft.warmCapacity === "number" && typeof draft.maxCapacity === "number"
      ? { warmCapacity: draft.warmCapacity, maxCapacity: draft.maxCapacity }
      : DEFAULT_FLEET_CAPACITY;

  for (const fleetId of draft.selectedFleetIds ?? []) {
    capacities[fleetId] ??= { ...legacyCapacity };
  }
  return capacities;
}

export function clearFleetManagerDraft(): void {
  window.localStorage.removeItem(LEGACY_FLEET_MANAGER_DRAFT_KEY);
  window.localStorage.removeItem(FLEET_MANAGER_DRAFT_KEY);
}
