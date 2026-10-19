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
  runnerVersions: string[];
  fleetPresets: Array<{ id: string; spec: FleetSpec }>;
  infrastructureDefaults: {
    source: "terraform" | "manual";
    provider: FleetManagerProvider | null;
    managerId: string;
    namespace: string;
    serviceAccountName: string;
    fields: Record<string, string | string[]>;
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
  fieldsByProvider?: Partial<Record<FleetManagerProvider, FleetManagerProviderFields>>;
};

export type InstalledFleet = {
  id?: string;
  runnerVersion?: string;
  spec?: Partial<FleetSpec>;
};

type GenerateValuesInput = Omit<FleetManagerDraft, "namespace" | "serviceAccountName"> & {
  metadata: FleetManagerSetupMetadata;
  installationAdminToken: string;
  serviceAccountName?: string;
};

export type FleetManagerConfigurationInput = Omit<FleetManagerDraft, "provider" | "fields">;
type FleetSelectionInput = Pick<FleetManagerDraft, "runnerVersion" | "selectedFleetIds" | "fleetCapacities"> & {
  runnerVersions?: readonly string[];
};
type FleetInfrastructureInput = Pick<
  FleetManagerDraft,
  "managerId" | "namespace" | "serviceAccountName" | "superplaneUrl"
>;

const LEGACY_FLEET_MANAGER_DRAFT_KEY = "superplane.fleet-manager-setup.v1";
export const FLEET_MANAGER_DRAFT_KEY = "superplane.fleet-manager-setup.v2";
export const DEFAULT_FLEET_CAPACITY: FleetCapacity = { warmCapacity: 1, maxCapacity: 10 };

const splitList = (value: string | undefined, fallback: string) =>
  (value || fallback)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

const machineType = (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields, fallback: string) =>
  fields[`machineType.${fleetId}`] ||
  fields[`machineType${spec.architecture === "amd64" ? "Amd64" : "Arm64"}`] ||
  fallback;

export const recommendedMachineType = (
  provider: FleetManagerProvider,
  architecture: FleetSpec["architecture"],
): string => {
  if (provider === "aws") return architecture === "amd64" ? "m8a.large" : "m8g.large";
  if (provider === "gcp") return architecture === "amd64" ? "e2-standard-4" : "t2a-standard-4";
  return architecture === "amd64" ? "Standard_D2ds_v4" : "Standard_D2ps_v5";
};

export const DEFAULT_SETUP_FLEET_ID = "e1-large-amd64";

export type CloudMachineType = {
  id: string;
  vcpu?: number;
  memoryGb?: number;
};

const CLOUD_MACHINE_TYPES: Record<FleetManagerProvider, CloudMachineType[]> = {
  aws: [
    { id: "m8a.large", vcpu: 2, memoryGb: 8 },
    { id: "m8a.xlarge", vcpu: 4, memoryGb: 16 },
    { id: "m7i.large", vcpu: 2, memoryGb: 8 },
    { id: "m7i.xlarge", vcpu: 4, memoryGb: 16 },
  ],
  gcp: [
    { id: "e2-standard-2", vcpu: 2, memoryGb: 8 },
    { id: "e2-standard-4", vcpu: 4, memoryGb: 16 },
    { id: "n2-standard-2", vcpu: 2, memoryGb: 8 },
    { id: "n2-standard-4", vcpu: 4, memoryGb: 16 },
  ],
  azure: [
    { id: "Standard_D2ds_v4", vcpu: 2, memoryGb: 8 },
    { id: "Standard_D4ds_v4", vcpu: 4, memoryGb: 16 },
    { id: "Standard_D2s_v5", vcpu: 2, memoryGb: 8 },
    { id: "Standard_D4s_v5", vcpu: 4, memoryGb: 16 },
  ],
};

export function formatMachineTypeOption(option: CloudMachineType): string {
  if (option.vcpu == null || option.memoryGb == null) return option.id;
  return `${option.id} · ${option.vcpu} vCPU · ${option.memoryGb} GB memory`;
}

export function machineTypeChoices(provider: FleetManagerProvider, current: string): CloudMachineType[] {
  const catalog = CLOUD_MACHINE_TYPES[provider];
  const value = current.trim();
  if (!value || catalog.some((option) => option.id === value)) return catalog;
  return [...catalog, { id: value }];
}

export function resolveSetupMachineType(provider: FleetManagerProvider, current: string | undefined): string {
  const value = current?.trim() ?? "";
  if (!value) return recommendedMachineType(provider, "amd64");
  if (CLOUD_MACHINE_TYPES[provider].some((option) => option.id === value)) return value;
  const belongsToAnotherCloud = (["aws", "gcp", "azure"] as const).some(
    (other) => other !== provider && CLOUD_MACHINE_TYPES[other].some((option) => option.id === value),
  );
  if (belongsToAnotherCloud) return recommendedMachineType(provider, "amd64");
  return value;
}

export const fleetImage = (fields: FleetManagerProviderFields, architecture: FleetSpec["architecture"]): string =>
  fields[architecture === "amd64" ? "imageAmd64" : "imageArm64"]?.trim() ?? "";

const configuredRunnerImage = (fields: FleetManagerProviderFields, architecture: FleetSpec["architecture"]): string =>
  fleetImage(fields, architecture) || "<RUNNER_IMAGE>";

const providerConfiguration = {
  aws: (fleetId: string, spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    aws: {
      region: fields.region || "<AWS_REGION>",
      ami: configuredRunnerImage(fields, spec.architecture),
      instanceType: machineType(fleetId, spec, fields, recommendedMachineType("aws", spec.architecture)),
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
      machineType: machineType(fleetId, spec, fields, recommendedMachineType("gcp", spec.architecture)),
      image: configuredRunnerImage(fields, spec.architecture),
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
      imageId: configuredRunnerImage(fields, spec.architecture),
      vmSize: machineType(fleetId, spec, fields, recommendedMachineType("azure", spec.architecture)),
      architecture: spec.architecture,
      subnetId: fields.subnetId || "<RUNNER_SUBNET_ID>",
      networkSecurityGroupId: fields.securityGroupIds || "<NETWORK_SECURITY_GROUP_ID>",
      identityId: fields.runnerIdentity || "<RUNNER_IDENTITY_ID>",
      zones: fields.zones?.trim() ? splitList(fields.zones, "<AZURE_ZONE>") : [],
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

  return quoteReleaseBaseUrl(
    dump(
      {
        manager: { id: input.managerId },
        superplane: { url: input.superplaneUrl },
        installationAdminToken: input.installationAdminToken,
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
    ),
  );
}

// An unquoted https value is a shell URL. zsh url-quote-magic then inserts
// a backslash before { and }. Fleet Manager replaces only the exact text {version}.
function quoteReleaseBaseUrl(values: string): string {
  return values.replace(/^(\s*releaseBaseUrl:\s+)(.+?)\s*$/m, (_match, prefix: string, value: string) => {
    const trimmed = value.trim();
    if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
      return `${prefix}${trimmed}`;
    }
    return `${prefix}"${trimmed.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
  });
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

export function installationRunnerVersion(
  metadata: Pick<FleetManagerSetupMetadata, "defaultRunnerVersion" | "runnerVersions">,
): string {
  const listed = metadata.runnerVersions ?? [];
  const preferred = metadata.defaultRunnerVersion?.trim() ?? "";
  if (preferred && listed.includes(preferred)) return preferred;
  return listed[0] ?? "";
}

export function validateFleetSelection(input: FleetSelectionInput): string[] {
  const errors: string[] = [];
  const runnerVersions = input.runnerVersions ?? [];
  if (!input.runnerVersion || !runnerVersions.includes(input.runnerVersion)) {
    errors.push("SuperPlane could not determine a Runner version.");
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

export function validateFleetManagerConfiguration(
  input: FleetManagerConfigurationInput & { runnerVersions?: readonly string[] },
): string[] {
  return [...validateFleetSelection(input), ...validateFleetInfrastructure(input)];
}

const INSTALL_PLACEHOLDER = /<[A-Z0-9_]+>/;

export function terraformInstallPlaceholders(draft: FleetManagerDraft, metadata: FleetManagerSetupMetadata): string[] {
  const defaults = metadata.infrastructureDefaults;
  if (defaults.source !== "terraform" || defaults.provider !== draft.provider) return [];

  const values = generateFleetManagerValues({
    metadata,
    ...draft,
    installationAdminToken: "token",
  });
  if (!INSTALL_PLACEHOLDER.test(values)) return [];
  return ["Apply Terraform again before you install Fleet Manager. A required value is missing."];
}

export const FLEET_MANAGER_VALUES_FILE = "fleet-manager-values.yaml";

export function fleetManagerValuesFileCommand(values: string): string {
  return `cat > ${FLEET_MANAGER_VALUES_FILE} <<'EOF'\n${values.trimEnd()}\nEOF\n`;
}

export function fleetManagerHelmCommand(metadata: FleetManagerSetupMetadata, namespace: string): string {
  return [
    "helm upgrade --install fleet-manager",
    metadata.chartRepository,
    `--version ${metadata.defaultChartVersion}`,
    `--namespace ${namespace}`,
    "--create-namespace",
    "--wait",
    "--timeout 10m",
    `-f ${FLEET_MANAGER_VALUES_FILE}`,
  ].join(" \\\n  ");
}

export function fleetManagerInstallCommand(
  metadata: FleetManagerSetupMetadata,
  namespace: string,
  values: string,
): string {
  return `${fleetManagerValuesFileCommand(values)}\n${fleetManagerHelmCommand(metadata, namespace)}\n`;
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

  if (draft.selectedFleetIds.length === 0) return false;

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

const FLEET_MANAGER_PROVIDERS: FleetManagerProvider[] = ["aws", "gcp", "azure"];

export function mergeTerraformInfrastructure(
  draft: FleetManagerDraft,
  metadata: FleetManagerSetupMetadata,
): FleetManagerDraft {
  const defaults = metadata.infrastructureDefaults;
  if (defaults.source !== "terraform" || defaults.provider !== draft.provider) return draft;

  const { superplaneUrl, ...infrastructureFields } = normalizeInfrastructureFields(defaults.fields);
  const machineTypes = Object.fromEntries(
    Object.entries(draft.fields).filter(([key]) => key.startsWith("machineType.")),
  );
  const fields = { ...infrastructureFields, ...machineTypes };
  return {
    ...draft,
    superplaneUrl: superplaneUrl || draft.superplaneUrl,
    fields,
    fieldsByProvider: {
      ...draft.fieldsByProvider,
      [draft.provider]: fields,
    },
  };
}

export function rememberProviderFields(draft: FleetManagerDraft): FleetManagerDraft {
  return {
    ...draft,
    fieldsByProvider: {
      ...draft.fieldsByProvider,
      [draft.provider]: draft.fields,
    },
  };
}

export function saveFleetManagerDraft(draft: FleetManagerDraft): void {
  window.localStorage.removeItem(LEGACY_FLEET_MANAGER_DRAFT_KEY);
  window.localStorage.setItem(FLEET_MANAGER_DRAFT_KEY, JSON.stringify(rememberProviderFields(draft)));
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
    if (!draft.provider || !FLEET_MANAGER_PROVIDERS.includes(draft.provider)) return null;
    if (!hasRequiredDraftStrings(draft)) return null;
    if (!Array.isArray(draft.selectedFleetIds) || !draft.selectedFleetIds.every((id) => typeof id === "string"))
      return null;
    if (typeof draft.fields !== "object" || draft.fields === null) return null;
    const fields = Object.fromEntries(
      Object.entries(draft.fields).filter((entry): entry is [string, string] => typeof entry[1] === "string"),
    );
    const fleetCapacities = storedFleetCapacities(draft);
    const fieldsByProvider = {
      ...storedFieldsByProvider(draft.fieldsByProvider),
      [draft.provider]: fields,
    };
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
      fieldsByProvider,
    };
  } catch {
    return null;
  }
}

function storedFieldsByProvider(value: unknown): Partial<Record<FleetManagerProvider, FleetManagerProviderFields>> {
  if (typeof value !== "object" || value === null) return {};

  return Object.fromEntries(
    FLEET_MANAGER_PROVIDERS.flatMap((provider) => {
      const fields = (value as Record<string, unknown>)[provider];
      if (typeof fields !== "object" || fields === null) return [];
      return [
        [
          provider,
          Object.fromEntries(
            Object.entries(fields).filter((entry): entry is [string, string] => typeof entry[1] === "string"),
          ),
        ],
      ];
    }),
  );
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
