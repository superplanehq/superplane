import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import type { FleetManagerProvider } from "@/lib/fleetManagerSetup";
import { Cloud } from "lucide-react";

export type FleetManagerSetupState = ReturnType<typeof useFleetManagerSetup>;

type ProviderField = { key: string; label: string; placeholder: string };

const providerFields: Record<FleetManagerProvider, ProviderField[]> = {
  aws: [
    { key: "region", label: "AWS region", placeholder: "us-east-1" },
    { key: "subnetIds", label: "Private subnet IDs", placeholder: "subnet-a, subnet-b" },
    { key: "securityGroupIds", label: "Runner security group IDs", placeholder: "sg-0123456789" },
    { key: "runnerIdentity", label: "Runner instance profile", placeholder: "superplane-runner" },
    {
      key: "fleetManagerIdentity",
      label: "Fleet Manager role ARN",
      placeholder: "arn:aws:iam::123456789012:role/fleet-manager",
    },
  ],
  gcp: [
    { key: "projectId", label: "GCP project", placeholder: "my-project" },
    { key: "zones", label: "Zones", placeholder: "us-central1-a, us-central1-b" },
    { key: "subnetwork", label: "Runner subnetwork", placeholder: "projects/.../subnetworks/runners" },
    { key: "networkTags", label: "Runner network tags", placeholder: "superplane-runner" },
    {
      key: "runnerIdentity",
      label: "Runner service account",
      placeholder: "runner@project.iam.gserviceaccount.com",
    },
    {
      key: "fleetManagerIdentity",
      label: "Fleet Manager service account",
      placeholder: "fleet-manager@project.iam.gserviceaccount.com",
    },
  ],
  azure: [
    { key: "subscriptionId", label: "Azure subscription ID", placeholder: "00000000-0000-0000-0000-000000000000" },
    { key: "resourceGroup", label: "Resource group", placeholder: "superplane" },
    { key: "location", label: "Location", placeholder: "eastus" },
    { key: "zones", label: "Zones", placeholder: "1, 2" },
    { key: "subnetId", label: "Runner subnet ID", placeholder: "/subscriptions/.../subnets/runners" },
    {
      key: "securityGroupIds",
      label: "Network security group ID",
      placeholder: "/subscriptions/.../networkSecurityGroups/runners",
    },
    {
      key: "runnerIdentity",
      label: "Runner identity ID",
      placeholder: "/subscriptions/.../userAssignedIdentities/runner",
    },
    {
      key: "fleetManagerIdentity",
      label: "Fleet Manager client ID",
      placeholder: "00000000-0000-0000-0000-000000000000",
    },
  ],
};

const machineTypePlaceholder = (provider: FleetManagerProvider, architecture: "amd64" | "arm64") => {
  if (provider === "aws") return architecture === "amd64" ? "m8a.large" : "m8g.large";
  if (provider === "gcp") return architecture === "amd64" ? "e2-standard-4" : "t2a-standard-4";
  return architecture === "amd64" ? "Standard_D2ds_v4" : "Standard_D2ps_v5";
};

const Field = ({
  id,
  label,
  imported = false,
  ...props
}: { id: string; label: string; imported?: boolean } & React.ComponentProps<typeof Input>) => (
  <div className="space-y-1.5 text-sm">
    <div className="flex items-center justify-between gap-3">
      <Label htmlFor={id}>{label}</Label>
      {imported && <span className="text-[11px] text-emerald-400">Loaded from Terraform</span>}
    </div>
    <Input id={id} {...props} />
  </div>
);

export function FleetManagerProviderSection({ setup }: { setup: FleetManagerSetupState }) {
  if (!setup.metadata || !setup.draft) return null;
  const terraformProvider = setup.metadata.infrastructureDefaults.provider;

  return (
    <div className="grid gap-3 sm:grid-cols-3" role="radiogroup" aria-label="Cloud provider">
      {(["aws", "gcp", "azure"] as const).map((provider) => {
        const selected = setup.draft?.provider === provider;
        return (
          <Button
            key={provider}
            type="button"
            variant={selected ? "default" : "outline"}
            className="h-auto min-h-20 flex-col gap-1"
            aria-pressed={selected}
            onClick={() => setup.setProvider(provider)}
          >
            <Cloud className="size-4" aria-hidden />
            <span>{provider.toUpperCase()}</span>
            {terraformProvider === provider && <span className="text-[10px] normal-case">Loaded from Terraform</span>}
          </Button>
        );
      })}
    </div>
  );
}

export function FleetManagerFleetSection({
  setup,
  contained = false,
}: {
  setup: FleetManagerSetupState;
  contained?: boolean;
}) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div>
      <div className={contained ? "max-h-52 space-y-2 overflow-y-auto pr-1" : "space-y-2"}>
        {setup.metadata.fleetPresets.map((preset) => {
          const id = `fleet-${preset.id}`;
          return (
            <div key={preset.id} className="flex items-start gap-3 rounded-md border border-border p-3">
              <Checkbox
                id={id}
                checked={setup.draft?.selectedFleetIds.includes(preset.id)}
                onChange={(event) => setup.toggleFleet(preset.id, event.target.checked)}
              />
              <div className="min-w-0 flex-1 space-y-3">
                <Label htmlFor={id} className="text-sm">
                  <span className="block font-medium">{preset.id}</span>
                  <span className="block text-xs text-muted-foreground">
                    {preset.spec.architecture} · {preset.spec.cpuMillicores}m CPU · {preset.spec.memoryMb} MB memory
                  </span>
                </Label>
                {setup.draft?.selectedFleetIds.includes(preset.id) && (
                  <div className="grid gap-3 sm:grid-cols-3">
                    <Field
                      id={`machine-type-${preset.id}`}
                      label={`Machine type for ${preset.id}`}
                      value={setup.draft.fields[`machineType.${preset.id}`] || ""}
                      placeholder={machineTypePlaceholder(setup.draft.provider, preset.spec.architecture)}
                      onChange={(event) => setup.setField(`machineType.${preset.id}`, event.target.value)}
                    />
                    <Field
                      id={`warm-capacity-${preset.id}`}
                      label={`Warm capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={setup.draft.fleetCapacities[preset.id]?.warmCapacity ?? 0}
                      onChange={(event) =>
                        setup.setFleetCapacity(preset.id, { warmCapacity: Number(event.target.value) })
                      }
                    />
                    <Field
                      id={`max-capacity-${preset.id}`}
                      label={`Maximum capacity for ${preset.id}`}
                      type="number"
                      min={0}
                      value={setup.draft.fleetCapacities[preset.id]?.maxCapacity ?? 1}
                      onChange={(event) =>
                        setup.setFleetCapacity(preset.id, { maxCapacity: Number(event.target.value) })
                      }
                    />
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
      <div className="mt-5 max-w-xs">
        <Field
          id="runner-version"
          label="Runner version"
          value={setup.draft.runnerVersion}
          onChange={(event) => setup.setRunnerVersion(event.target.value)}
        />
      </div>
      {setup.fleetErrors.length > 0 && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {setup.fleetErrors[0]}
        </p>
      )}
    </div>
  );
}

export function FleetManagerInfrastructureSection({
  setup,
  contained = false,
}: {
  setup: FleetManagerSetupState;
  contained?: boolean;
}) {
  if (!setup.metadata || !setup.draft) return null;
  const namespaceIsManaged = setup.metadata.credentials.autoProvisioned;

  return (
    <div className={contained ? "max-h-[50vh] space-y-4 overflow-y-auto pr-2" : "space-y-4"}>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="manager-id"
          label="Fleet Manager ID"
          imported={setup.metadata.infrastructureDefaults.managerId === setup.draft.managerId}
          value={setup.draft.managerId}
          onChange={(event) => setup.setManagerId(event.target.value)}
        />
        <Field
          id="namespace"
          label="Kubernetes namespace"
          imported={setup.metadata.infrastructureDefaults.namespace === setup.draft.namespace}
          value={setup.draft.namespace}
          disabled={namespaceIsManaged}
          onChange={(event) => setup.setNamespace(event.target.value)}
        />
        <Field
          id="service-account-name"
          label="Fleet Manager service account"
          imported={setup.metadata.infrastructureDefaults.serviceAccountName === setup.draft.serviceAccountName}
          value={setup.draft.serviceAccountName}
          onChange={(event) => setup.setServiceAccountName(event.target.value)}
        />
        <Field
          id="superplane-url"
          label="SuperPlane URL"
          imported={setup.metadata.infrastructureDefaults.fields.superplaneUrl === setup.draft.superplaneUrl}
          value={setup.draft.superplaneUrl}
          placeholder="https://superplane.example.com"
          onChange={(event) => setup.setSuperplaneUrl(event.target.value)}
        />
        {providerFields[setup.draft.provider].map((field) => (
          <Field
            key={field.key}
            id={`provider-${field.key}`}
            label={field.label}
            imported={setup.importedFieldKeys.has(field.key)}
            value={setup.draft?.fields[field.key] || ""}
            placeholder={field.placeholder}
            onChange={(event) => setup.setField(field.key, event.target.value)}
          />
        ))}
      </div>
      {namespaceIsManaged && (
        <p className="text-xs text-muted-foreground">
          The generated credential requires Fleet Manager in the SuperPlane namespace.
        </p>
      )}
      {setup.infrastructureErrors.length > 0 && (
        <p role="alert" className="text-sm text-destructive">
          {setup.infrastructureErrors[0]}
        </p>
      )}
    </div>
  );
}
