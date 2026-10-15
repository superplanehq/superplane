import { useMemo } from "react";

import {
  FleetManagerFleetSection,
  FleetManagerInfrastructureSection,
  FleetManagerProviderSection,
  type FleetManagerSetupState,
} from "@/components/fleet-manager-setup/FleetManagerSections";
import { downloadFleetManagerValues } from "@/lib/downloadFleetManagerValues";
import { Button } from "@/components/ui/button";
import { type FleetManagerVerificationStatus, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import { fleetManagerInstallCommand } from "@/lib/fleetManagerSetup";
import type { FleetManagerWizardStep } from "@/lib/fleetManagerWizard";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { CopyButton } from "@/ui/CopyButton";
import { CheckCircle2, Download, LoaderCircle, TriangleAlert } from "lucide-react";

type WizardProps = {
  step: FleetManagerWizardStep;
  setup: FleetManagerSetupState;
  onNext: () => void;
  onBack?: () => void;
  onFinish: () => void;
  onFinishLater: () => void;
  showBackButton?: boolean;
};

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

function ProviderStep({ setup, actions }: { setup: FleetManagerSetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div data-testid="fleet-manager-provider-step">
      <FirstRunHeading headline="Choose a cloud provider" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Select the account that will run your fleet machines.
        </p>
      </FirstRunHeading>
      <div className="mt-7">
        <FleetManagerProviderSection setup={setup} />
      </div>
      {actions}
    </div>
  );
}

function FleetsStep({ setup, actions }: { setup: FleetManagerSetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div data-testid="fleet-manager-fleets-step">
      <FirstRunHeading headline="Choose runner fleets" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Select fleet sizes and set their initial capacity.
        </p>
      </FirstRunHeading>
      <div className="mt-6">
        <FleetManagerFleetSection setup={setup} contained />
      </div>
      {actions}
    </div>
  );
}

function InfrastructureStep({ setup, actions }: { setup: FleetManagerSetupState; actions: React.ReactNode }) {
  if (!setup.metadata || !setup.draft) return null;

  return (
    <div data-testid="fleet-manager-infrastructure-step">
      <FirstRunHeading headline="Confirm cloud infrastructure" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Review the values from Terraform and add missing account details.
        </p>
      </FirstRunHeading>
      <div className="mt-6">
        <FleetManagerInfrastructureSection setup={setup} contained />
      </div>
      {actions}
    </div>
  );
}

function ReviewStep({ setup, actions }: { setup: FleetManagerSetupState; actions: React.ReactNode }) {
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
        <Button type="button" variant="outline" onClick={() => downloadFleetManagerValues(setup.values)}>
          <Download className="size-4" aria-hidden />
          Download values.yaml
        </Button>
      </div>
      {actions}
    </div>
  );
}

const fleetManagerVerificationCopy: Record<FleetManagerVerificationStatus, string> = {
  idle: "Verification has not started.",
  checking: "Checking Fleet Manager installation...",
  waiting: "Waiting for the selected fleets...",
  verified: "Fleet Manager created every selected fleet with the requested configuration.",
  unauthorized: "SuperPlane could not verify the fleets. Sign in again and retry.",
  error: "SuperPlane could not verify the fleets. Check the installation and retry.",
  timeout: "Verification timed out. Check the Helm command and retry.",
};

export function FleetManagerVerificationStatus({
  status,
  onRetry,
}: {
  status: FleetManagerVerificationStatus;
  onRetry: () => void;
}) {
  const failed = ["unauthorized", "error", "timeout"].includes(status);

  return (
    <div className="flex items-start gap-3 rounded-md border border-border p-4 text-sm" role="status">
      {status === "verified" ? (
        <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-emerald-400" aria-hidden />
      ) : failed ? (
        <TriangleAlert className="mt-0.5 size-5 shrink-0 text-amber-400" aria-hidden />
      ) : (
        <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin" aria-hidden />
      )}
      <div>
        <p>{fleetManagerVerificationCopy[status]}</p>
        {failed && (
          <Button type="button" variant="link" className="h-auto p-0" onClick={onRetry}>
            Retry verification
          </Button>
        )}
      </div>
    </div>
  );
}

function InstallStep({ setup, onFinish, onFinishLater }: WizardProps) {
  const verification = useFleetManagerVerification(true, setup.draft, setup.metadata);
  const command = useMemo(
    () =>
      setup.metadata && setup.draft
        ? fleetManagerInstallCommand(setup.metadata, setup.draft.namespace, "values.yaml")
        : "",
    [setup.draft, setup.metadata],
  );

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
      <div className="mt-5">
        <FleetManagerVerificationStatus status={verification.status} onRetry={verification.retry} />
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
  const stepErrors =
    props.step === "provider"
      ? []
      : props.step === "fleets"
        ? props.setup.fleetErrors
        : props.step === "review"
          ? props.setup.validationErrors
          : props.setup.infrastructureErrors;
  const continueActions = (
    <WizardActions
      primary={props.step === "review" ? "Continue to install" : "Continue"}
      onPrimary={props.onNext}
      primaryDisabled={stepErrors.length > 0}
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
