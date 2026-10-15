import { useMemo } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { type FleetManagerVerificationStatus, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import type { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { fleetManagerInstallCommand, type FleetManagerProvider } from "@/lib/fleetManagerSetup";
import type { FleetManagerWizardStep } from "@/lib/fleetManagerWizard";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { CopyButton } from "@/ui/CopyButton";
import { CheckCircle2, Cloud, Download, LoaderCircle, TriangleAlert } from "lucide-react";

type SetupState = ReturnType<typeof useFleetManagerSetup>;

type WizardProps = {
  step: FleetManagerWizardStep;
  setup: SetupState;
  onNext: () => void;
  onBack?: () => void;
  onFinish: () => void;
  onFinishLater: () => void;
  showBackButton?: boolean;
};

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

function WizardActions({
  primary,
  onPrimary,
  primaryDisabled = false,
  secondary,
  onSecondary,
  showBack,
  onBack,
}: {
  primary: string;
  onPrimary: () => void;
  primaryDisabled?: boolean;
  secondary?: string;
  onSecondary?: () => void;
  showBack?: boolean;
  onBack?: () => void;
}) {
  return (
    <div className="mt-7 flex flex-wrap items-center gap-3">
      {showBack && onBack && (
        <Button type="button" variant="outline" onClick={onBack}>
          Back
        </Button>
      )}
      <Button type="button" onClick={onPrimary} disabled={primaryDisabled}>
        {primary}
      </Button>
      {secondary && onSecondary && (
        <Button type="button" variant="ghost" onClick={onSecondary}>
          {secondary}
        </Button>
      )}
    </div>
  );
}

function ProviderStep({ setup, actions }: { setup: SetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;
  const terraformProvider = setup.metadata.infrastructureDefaults.provider;

  return (
    <div data-testid="fleet-manager-provider-step">
      <FirstRunHeading headline="Choose a cloud provider" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Select the account that will run your fleet machines.
        </p>
      </FirstRunHeading>
      <div className="mt-7 grid gap-3 sm:grid-cols-3">
        {(["aws", "gcp", "azure"] as const).map((provider) => (
          <Button
            key={provider}
            type="button"
            variant={setup.draft?.provider === provider ? "default" : "outline"}
            className="h-auto min-h-20 flex-col gap-1"
            onClick={() => setup.setProvider(provider)}
          >
            <Cloud className="size-4" aria-hidden />
            <span>{provider.toUpperCase()}</span>
            {terraformProvider === provider && <span className="text-[10px] normal-case">Loaded from Terraform</span>}
          </Button>
        ))}
      </div>
      {actions}
    </div>
  );
}

function FleetsStep({ setup, actions }: { setup: SetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div data-testid="fleet-manager-fleets-step">
      <FirstRunHeading headline="Choose runner fleets" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Select fleet sizes and set their initial capacity.
        </p>
      </FirstRunHeading>
      <div className="mt-6 max-h-52 space-y-2 overflow-y-auto pr-1">
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
                  <Field
                    id={`machine-type-${preset.id}`}
                    label={`Machine type for ${preset.id}`}
                    value={setup.draft.fields[`machineType.${preset.id}`] || ""}
                    placeholder={machineTypePlaceholder(setup.draft.provider, preset.spec.architecture)}
                    onChange={(event) => setup.setField(`machineType.${preset.id}`, event.target.value)}
                  />
                )}
              </div>
            </div>
          );
        })}
      </div>
      <div className="mt-5 grid gap-4 sm:grid-cols-3">
        <Field
          id="runner-version"
          label="Runner version"
          value={setup.draft.runnerVersion}
          onChange={(event) => setup.setRunnerVersion(event.target.value)}
        />
        <Field
          id="warm-capacity"
          label="Warm capacity"
          type="number"
          min={0}
          value={setup.draft.warmCapacity}
          onChange={(event) => setup.setWarmCapacity(Number(event.target.value))}
        />
        <Field
          id="max-capacity"
          label="Maximum capacity"
          type="number"
          min={0}
          value={setup.draft.maxCapacity}
          onChange={(event) => setup.setMaxCapacity(Number(event.target.value))}
        />
      </div>
      {setup.validationErrors.length > 0 && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {setup.validationErrors[0]}
        </p>
      )}
      {actions}
    </div>
  );
}

function InfrastructureStep({ setup, actions }: { setup: SetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;
  const namespaceIsManaged = setup.metadata.credentials.autoProvisioned;

  return (
    <div data-testid="fleet-manager-infrastructure-step">
      <FirstRunHeading headline="Confirm cloud infrastructure" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Review the values from Terraform and add missing account details.
        </p>
      </FirstRunHeading>
      <div className="mt-6 max-h-[50vh] space-y-4 overflow-y-auto pr-2">
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
      </div>
      {actions}
    </div>
  );
}

function downloadValues(values: string) {
  const url = URL.createObjectURL(new Blob([values], { type: "application/yaml;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = "values.yaml";
  link.click();
  URL.revokeObjectURL(url);
}

function ReviewStep({ setup, actions }: { setup: SetupState; actions: React.ReactNode }) {
  return (
    <div data-testid="fleet-manager-review-step">
      <FirstRunHeading headline="Review Fleet Manager values" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Save this file before you install the standalone chart.
        </p>
      </FirstRunHeading>
      <div className="mt-5 flex gap-2 rounded-md border border-amber-700/60 bg-amber-950/30 p-3 text-sm text-amber-100">
        <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
        <p>Replace each public VM image placeholder before production use.</p>
      </div>
      <pre className="mt-4 max-h-72 overflow-auto rounded-md border border-border bg-muted p-4 text-left font-mono text-xs">
        <code>{setup.values}</code>
      </pre>
      <div className="mt-3 flex flex-wrap gap-2">
        <CopyButton variant="button" text={setup.values}>
          Copy values.yaml
        </CopyButton>
        <Button type="button" variant="outline" onClick={() => downloadValues(setup.values)}>
          <Download className="size-4" aria-hidden />
          Download values.yaml
        </Button>
      </div>
      {actions}
    </div>
  );
}

const verificationCopy: Record<FleetManagerVerificationStatus, string> = {
  idle: "Verification has not started.",
  checking: "Checking Fleet Manager installation...",
  waiting: "Waiting for the selected fleets...",
  verified: "Fleet Manager created every selected fleet with the requested configuration.",
  unauthorized: "SuperPlane could not verify the fleets. Sign in again and retry.",
  error: "SuperPlane could not verify the fleets. Check the installation and retry.",
  timeout: "Verification timed out. Check the Helm command and retry.",
};

function InstallStep({ setup, onFinish, onFinishLater }: WizardProps) {
  const verification = useFleetManagerVerification(true, setup.draft, setup.metadata);
  const command = useMemo(
    () =>
      setup.metadata && setup.draft
        ? fleetManagerInstallCommand(setup.metadata, setup.draft.namespace, "values.yaml")
        : "",
    [setup.draft, setup.metadata],
  );
  const failed = ["unauthorized", "error", "timeout"].includes(verification.status);

  return (
    <div data-testid="fleet-manager-install-step">
      <FirstRunHeading headline="Install Fleet Manager" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Run this command where kubectl can access your cluster.
        </p>
      </FirstRunHeading>
      <pre className="mt-5 max-h-40 overflow-auto rounded-md border border-border bg-muted p-4 text-left font-mono text-xs">
        <code>{command}</code>
      </pre>
      <div className="mt-3">
        <CopyButton variant="button" text={command}>
          Copy Helm command
        </CopyButton>
      </div>
      <div className="mt-5 flex items-start gap-3 rounded-md border border-border p-4 text-sm" role="status">
        {verification.status === "verified" ? (
          <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-emerald-400" aria-hidden />
        ) : failed ? (
          <TriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-400" aria-hidden />
        ) : (
          <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin" aria-hidden />
        )}
        <div>
          <p>{verificationCopy[verification.status]}</p>
          {failed && (
            <Button type="button" variant="link" className="h-auto p-0" onClick={verification.retry}>
              Retry verification
            </Button>
          )}
        </div>
      </div>
      <div className="mt-7 flex flex-wrap gap-3">
        {verification.status === "verified" && (
          <Button type="button" onClick={onFinish}>
            Finish installation
          </Button>
        )}
        <Button type="button" variant="ghost" onClick={onFinishLater}>
          Finish later
        </Button>
      </div>
    </div>
  );
}

export function FleetManagerWizardScreen(props: WizardProps) {
  const back = props.showBackButton ? props.onBack : undefined;
  const continueActions = (
    <WizardActions
      primary={props.step === "review" ? "Continue to install" : "Continue"}
      onPrimary={props.onNext}
      primaryDisabled={props.setup.validationErrors.length > 0}
      showBack={props.showBackButton}
      onBack={back}
    />
  );

  if (props.setup.loadError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        SuperPlane could not load Fleet Manager setup. Refresh the page and try again.
      </p>
    );
  }
  if (!props.setup.metadata || !props.setup.draft) {
    return <p className="text-sm text-muted-foreground">Loading Fleet Manager setup...</p>;
  }

  if (props.step === "provider") return <ProviderStep setup={props.setup} actions={continueActions} />;
  if (props.step === "fleets") return <FleetsStep setup={props.setup} actions={continueActions} />;
  if (props.step === "infrastructure") return <InfrastructureStep setup={props.setup} actions={continueActions} />;
  if (props.step === "review") return <ReviewStep setup={props.setup} actions={continueActions} />;
  return <InstallStep {...props} />;
}
