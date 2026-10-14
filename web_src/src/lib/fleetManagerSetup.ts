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
};

export type FleetManagerProviderFields = Record<string, string>;

type GenerateValuesInput = {
  metadata: FleetManagerSetupMetadata;
  provider: FleetManagerProvider;
  managerId: string;
  runnerVersion: string;
  selectedFleetIds: string[];
  warmCapacity: number;
  maxCapacity: number;
  fields: FleetManagerProviderFields;
};

const splitList = (value: string | undefined, fallback: string) =>
  (value || fallback)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

const providerConfiguration = {
  aws: (spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    aws: {
      region: fields.region || "<AWS_REGION>",
      ami: "<PUBLIC_AMI_ID_FOR_REGION_AND_ARCH>",
      instanceType: fields.machineType || "<INSTANCE_TYPE>",
      architecture: spec.architecture,
      subnetIds: splitList(fields.subnetIds, "<SUBNET_ID>"),
      securityGroupIds: splitList(fields.securityGroupIds, "<SECURITY_GROUP_ID>"),
      iamInstanceProfile: fields.runnerIdentity || "<INSTANCE_PROFILE>",
      volumeSizeGb: spec.diskGb,
    },
  }),
  gcp: (spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    gcp: {
      projectId: fields.projectId || "<GCP_PROJECT_ID>",
      zones: splitList(fields.zones, "<GCP_ZONE>"),
      machineType: fields.machineType || "<MACHINE_TYPE>",
      image: "<PUBLIC_GCP_IMAGE_OR_FAMILY>",
      architecture: spec.architecture,
      subnetwork: fields.subnetwork || "<RUNNER_SUBNETWORK>",
      ...(fields.runnerIdentity ? { serviceAccountEmail: fields.runnerIdentity } : {}),
      networkTags: splitList(fields.networkTags, "<RUNNER_NETWORK_TAG>"),
      diskSizeGb: spec.diskGb,
    },
  }),
  azure: (spec: FleetSpec, fields: FleetManagerProviderFields) => ({
    azure: {
      subscriptionId: fields.subscriptionId || "<AZURE_SUBSCRIPTION_ID>",
      resourceGroup: fields.resourceGroup || "<RESOURCE_GROUP>",
      location: fields.location || "<AZURE_LOCATION>",
      imageId: "<PUBLIC_AZURE_IMAGE_VERSION_ID>",
      vmSize: fields.machineType || "<VM_SIZE>",
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
    .map((preset) => ({
      id: preset.id,
      runnerVersion: input.runnerVersion,
      warmCapacity: input.warmCapacity,
      maxCapacity: input.maxCapacity,
      spec: preset.spec,
      provider: input.provider,
      ...providerConfiguration[input.provider](preset.spec, input.fields),
    }));

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
      superplane: { url: window.location.origin },
      credentials: {
        existingSecret: "superplane-fleet-manager-token",
        tokenKey: "INSTALLATION_ADMIN_TOKEN",
      },
      runner: { releaseBaseUrl: input.metadata.runnerReleaseUrlTemplate },
      image: {
        repository: input.metadata.imageRepository,
        tag: input.metadata.defaultFleetManagerVersion,
        pullPolicy: "IfNotPresent",
      },
      serviceAccount: {
        create: true,
        name: "superplane-fleet-manager",
        ...workloadIdentity,
      },
      podLabels: input.provider === "azure" ? { "azure.workload.identity/use": "true" } : {},
      fleets,
    },
    { noRefs: true, lineWidth: 120, quotingType: '"' },
  );
}

export function fleetManagerSecretCommand(namespace: string, secretName: string): string {
  const createSecret = [
    `kubectl --namespace ${namespace} create secret generic ${secretName}`,
    '  --from-literal=INSTALLATION_ADMIN_TOKEN="$FLEET_MANAGER_TOKEN"',
    "  --dry-run=client --output=yaml | kubectl apply -f -",
  ].join(" \\\n");
  return [
    "read -rsp 'Fleet Manager token: ' FLEET_MANAGER_TOKEN; echo",
    createSecret,
    "unset FLEET_MANAGER_TOKEN",
  ].join("\n");
}

export function fleetManagerInstallCommand(
  metadata: FleetManagerSetupMetadata,
  namespace: string,
  valuesFile: string,
): string {
  return `helm upgrade --install fleet-manager ${metadata.chartRepository} --version ${metadata.defaultChartVersion} --namespace ${namespace} --create-namespace --values ${valuesFile}`;
}

export const fleetManagerRestartCommand = (namespace: string) =>
  `kubectl --namespace ${namespace} rollout restart deployment/fleet-manager-fleet-manager`;
