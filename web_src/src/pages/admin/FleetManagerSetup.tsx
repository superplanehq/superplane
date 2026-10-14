import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import {
  fleetManagerInstallCommand,
  fleetManagerRestartCommand,
  fleetManagerSecretCommand,
  type FleetManagerProvider,
  type FleetManagerSetupMetadata,
} from "@/lib/fleetManagerSetup";
import { CopyButton } from "@/ui/CopyButton";

type ProviderField = { key: string; label: string; placeholder: string };
const providerFields: Record<FleetManagerProvider, ProviderField[]> = {
  aws: [
    { key: "region", label: "AWS region", placeholder: "us-east-1" },
    { key: "machineTypeAmd64", label: "EC2 instance type for AMD64", placeholder: "m8a.large" },
    { key: "machineTypeArm64", label: "EC2 instance type for ARM64", placeholder: "m8g.large" },
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
    { key: "machineTypeAmd64", label: "Compute Engine machine type for AMD64", placeholder: "e2-standard-4" },
    { key: "machineTypeArm64", label: "Compute Engine machine type for ARM64", placeholder: "t2a-standard-4" },
    { key: "subnetwork", label: "Runner subnetwork", placeholder: "projects/.../subnetworks/runners" },
    { key: "networkTags", label: "Runner network tags", placeholder: "superplane-runner" },
    {
      key: "runnerIdentity",
      label: "Runner service account (optional)",
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
    { key: "machineTypeAmd64", label: "Azure VM size for AMD64", placeholder: "Standard_D2ds_v4" },
    { key: "machineTypeArm64", label: "Azure VM size for ARM64", placeholder: "Standard_D2ps_v5" },
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

const Section = ({ title, children }: { title: string; children: React.ReactNode }) => (
  <section className="space-y-4 rounded-lg border border-slate-200 bg-white p-5 dark:border-gray-700 dark:bg-gray-900">
    <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
    {children}
  </section>
);

type SelectionProps = {
  metadata: FleetManagerSetupMetadata;
  provider: FleetManagerProvider;
  selected: string[];
  onProvider: (provider: FleetManagerProvider) => void;
  onToggle: (id: string, checked: boolean) => void;
};

function FleetSelection({ metadata, provider, selected, onProvider, onToggle }: SelectionProps) {
  return (
    <Section title="1. Select a provider and fleets">
      <div className="flex gap-2">
        {(["aws", "gcp", "azure"] as const).map((item) => (
          <Button
            key={item}
            type="button"
            variant={provider === item ? "default" : "outline"}
            onClick={() => onProvider(item)}
          >
            {item.toUpperCase()}
          </Button>
        ))}
      </div>
      <div className="grid gap-2 sm:grid-cols-2">
        {metadata.fleetPresets.map((preset) => {
          const inputId = `fleet-${preset.id}`;
          return (
            <div
              key={preset.id}
              className="flex items-start gap-2 rounded border border-slate-200 p-3 dark:border-gray-700"
            >
              <Checkbox
                id={inputId}
                checked={selected.includes(preset.id)}
                onChange={(event) => onToggle(preset.id, event.target.checked)}
              />
              <Label htmlFor={inputId} className="text-sm">
                <span className="block font-medium">{preset.id}</span>
                <span className="text-gray-500">
                  {preset.spec.architecture}, {preset.spec.cpuMillicores}m CPU, {preset.spec.memoryMb} MB memory
                </span>
              </Label>
            </div>
          );
        })}
      </div>
    </Section>
  );
}

type InfrastructureProps = {
  provider: FleetManagerProvider;
  managerId: string;
  namespace: string;
  serviceAccountName: string;
  runnerVersion: string;
  warmCapacity: number;
  maxCapacity: number;
  fields: Record<string, string>;
  setManagerId: (value: string) => void;
  setNamespace: (value: string) => void;
  setServiceAccountName: (value: string) => void;
  setRunnerVersion: (value: string) => void;
  setWarmCapacity: (value: number) => void;
  setMaxCapacity: (value: number) => void;
  setField: (key: string, value: string) => void;
};

const Field = ({ id, label, ...props }: { id: string; label: string } & React.ComponentProps<typeof Input>) => (
  <div className="space-y-1 text-sm">
    <Label htmlFor={id}>{label}</Label>
    <Input id={id} {...props} />
  </div>
);

function InfrastructureFields(props: InfrastructureProps) {
  return (
    <Section title="2. Configure capacity and infrastructure">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          id="manager-id"
          label="Fleet Manager ID"
          value={props.managerId}
          onChange={(event) => props.setManagerId(event.target.value)}
        />
        <Field
          id="runner-version"
          label="Immutable Runner version"
          value={props.runnerVersion}
          onChange={(event) => props.setRunnerVersion(event.target.value)}
        />
        <Field
          id="namespace"
          label="Kubernetes namespace"
          value={props.namespace}
          onChange={(event) => props.setNamespace(event.target.value)}
        />
        <Field
          id="service-account-name"
          label="Fleet Manager service account"
          value={props.serviceAccountName}
          onChange={(event) => props.setServiceAccountName(event.target.value)}
        />
        <Field
          id="warm-capacity"
          label="Warm capacity"
          type="number"
          min={0}
          value={props.warmCapacity}
          onChange={(event) => props.setWarmCapacity(Number(event.target.value))}
        />
        <Field
          id="max-capacity"
          label="Maximum capacity"
          type="number"
          min={0}
          value={props.maxCapacity}
          onChange={(event) => props.setMaxCapacity(Number(event.target.value))}
        />
        {providerFields[props.provider].map((field) => (
          <Field
            key={field.key}
            id={`provider-${field.key}`}
            label={field.label}
            value={props.fields[field.key] || ""}
            placeholder={field.placeholder}
            onChange={(event) => props.setField(field.key, event.target.value)}
          />
        ))}
      </div>
    </Section>
  );
}

type InstructionsProps = {
  metadata: FleetManagerSetupMetadata;
  values: string;
  organizationId: string;
  namespace: string;
  validationErrors: string[];
  tokenPending: boolean;
  tokenError: string | null;
  plaintextToken: string | null;
  onCreateToken: () => void;
};

function InstallationInstructions(props: InstructionsProps) {
  const secretCommand = fleetManagerSecretCommand(props.namespace, "superplane-fleet-manager-token");
  const installCommand = fleetManagerInstallCommand(props.metadata, props.namespace, "values.yaml");
  const configurationIsValid = props.validationErrors.length === 0;
  return (
    <>
      <Section title="3. Review values.yaml">
        <div className="rounded border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-100">
          Public VM images are not available yet. Replace every public image placeholder before production use.
        </div>
        <Textarea
          readOnly
          value={props.values}
          className="min-h-96 font-mono text-xs"
          aria-label="Fleet Manager values"
        />
        {props.validationErrors.length > 0 && (
          <div role="alert" className="text-sm text-red-600 dark:text-red-400">
            <p>Fix these values before you copy the configuration:</p>
            <ul className="list-disc pl-5">
              {props.validationErrors.map((error) => (
                <li key={error}>{error}</li>
              ))}
            </ul>
          </div>
        )}
        {configurationIsValid ? (
          <CopyButton variant="button" text={props.values}>
            Copy values.yaml
          </CopyButton>
        ) : (
          <Button type="button" variant="outline" disabled>
            Copy values.yaml
          </Button>
        )}
      </Section>
      <Section title="4. Create the token Secret">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Create a personal token only when you are ready to install. SuperPlane shows this token once. The token is not
          part of values.yaml or the Helm release.
        </p>
        <Button type="button" disabled={!props.organizationId || props.tokenPending} onClick={props.onCreateToken}>
          {props.tokenPending ? "Creating token..." : "Create Fleet Manager token"}
        </Button>
        {props.tokenError && (
          <p role="alert" className="text-sm text-red-600 dark:text-red-400">
            {props.tokenError}
          </p>
        )}
        {props.plaintextToken && (
          <div className="space-y-2">
            <Textarea
              readOnly
              value={props.plaintextToken}
              className="font-mono text-xs"
              aria-label="Fleet Manager token"
            />
            <CopyButton variant="button" text={props.plaintextToken}>
              Copy token
            </CopyButton>
          </div>
        )}
        <Textarea
          readOnly
          value={secretCommand}
          className="min-h-32 font-mono text-xs"
          aria-label="Create token Secret command"
        />
        <CopyButton variant="button" text={secretCommand}>
          Copy Secret command
        </CopyButton>
      </Section>
      <Section title="5. Install Fleet Manager">
        <Textarea
          readOnly
          value={installCommand}
          className="font-mono text-xs"
          aria-label="Install Fleet Manager command"
        />
        <CopyButton variant="button" text={installCommand}>
          Copy Helm command
        </CopyButton>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          After token rotation, run <code>{fleetManagerRestartCommand(props.namespace)}</code>. Verify the new token,
          then revoke the old token.
        </p>
      </Section>
      <Section title="Fleet removal">
        <ol className="list-decimal space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-400">
          <li>Disable the fleet in SuperPlane.</li>
          <li>Keep it in values.yaml until Fleet Manager removes its runner resources.</li>
          <li>Remove it from values.yaml and upgrade the Helm release.</li>
        </ol>
      </Section>
    </>
  );
}

export default function FleetManagerSetup() {
  const setup = useFleetManagerSetup();

  if (setup.loadError)
    return <p className="text-sm text-red-600">Could not load Fleet Manager setup. Refresh the page and try again.</p>;
  if (!setup.metadata) return <p className="text-sm text-gray-500">Loading Fleet Manager setup...</p>;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Set up Fleet Manager</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Generate standalone Helm values for the runner infrastructure that Terraform created.
        </p>
      </header>
      <FleetSelection
        metadata={setup.metadata}
        provider={setup.provider}
        selected={setup.selectedFleetIds}
        onProvider={setup.setProvider}
        onToggle={setup.toggleFleet}
      />
      <InfrastructureFields
        provider={setup.provider}
        managerId={setup.managerId}
        namespace={setup.namespace}
        serviceAccountName={setup.serviceAccountName}
        runnerVersion={setup.runnerVersion}
        warmCapacity={setup.warmCapacity}
        maxCapacity={setup.maxCapacity}
        fields={setup.fields}
        setManagerId={setup.setManagerId}
        setNamespace={setup.setNamespace}
        setServiceAccountName={setup.setServiceAccountName}
        setRunnerVersion={setup.setRunnerVersion}
        setWarmCapacity={setup.setWarmCapacity}
        setMaxCapacity={setup.setMaxCapacity}
        setField={setup.setField}
      />
      <InstallationInstructions
        metadata={setup.metadata}
        values={setup.values}
        organizationId={setup.organizationId}
        namespace={setup.namespace}
        validationErrors={setup.validationErrors}
        tokenPending={setup.tokenPending}
        tokenError={setup.tokenError}
        plaintextToken={setup.plaintextToken}
        onCreateToken={() => void setup.generateToken()}
      />
    </div>
  );
}
