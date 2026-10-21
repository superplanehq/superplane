import { useEffect, useMemo, useRef } from "react";

import { FleetManagerCommandBlock } from "@/components/fleet-manager-setup/FleetManagerInstallCommands";
import {
  FleetManagerFleetSection,
  FleetManagerProviderSection,
  type FleetManagerSetupState,
} from "@/components/fleet-manager-setup/FleetManagerSections";
import { Button } from "@/components/ui/button";
import { type FleetManagerVerificationStatus, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import {
  FLEET_MANAGER_VALUES_FILE,
  fleetManagerDockerCommand,
  fleetManagerHelmCommand,
  fleetManagerValuesFileCommand,
} from "@/lib/fleetManagerSetup";
import {
  fleetProviderChoiceCopy,
  omittedFleetWizardStep,
  singleHostInstall,
  type FleetManagerWizardStep,
} from "@/lib/fleetManagerWizard";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { CheckCircle2, LoaderCircle, TriangleAlert } from "lucide-react";

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

  const choice = fleetProviderChoiceCopy(singleHostInstall(setup.metadata.installationType));

  return (
    <div data-testid="fleet-manager-provider-step">
      <FirstRunHeading headline={choice.title} size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">{choice.description}</p>
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
      <FirstRunHeading headline="Configure the default fleet" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Set the cloud machine type and how many machines to keep ready.
        </p>
      </FirstRunHeading>
      <div className="mt-6">
        <FleetManagerFleetSection setup={setup} />
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
  timeout: "Verification timed out. Check the install command and retry.",
};

export function FleetManagerVerificationStatus({
  status,
  onRetry,
  copy,
}: {
  status: FleetManagerVerificationStatus;
  onRetry: () => void;
  copy?: Partial<Record<FleetManagerVerificationStatus, string>>;
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
        <p>{copy?.[status] ?? fleetManagerVerificationCopy[status]}</p>
        {failed && (
          <Button type="button" variant="link" className="h-auto p-0" onClick={onRetry}>
            Retry verification
          </Button>
        )}
      </div>
    </div>
  );
}

function ValuesStep({ setup, actions }: { setup: FleetManagerSetupState; actions: React.ReactNode }) {
  const valuesCommand = useMemo(
    () => (setup.values ? fleetManagerValuesFileCommand(setup.values) : null),
    [setup.values],
  );

  return (
    <div data-testid="fleet-manager-values-step">
      <FirstRunHeading headline="Create the chart values" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Run this command. It writes {FLEET_MANAGER_VALUES_FILE}.
        </p>
      </FirstRunHeading>
      {valuesCommand && (
        <div className="mt-5">
          <FleetManagerCommandBlock code={valuesCommand} language="shell" copyLabel="Copy values command" />
        </div>
      )}
      {actions}
    </div>
  );
}

function DockerStep({ setup, onFinish, onFinishLater }: WizardProps) {
  const verification = useFleetManagerVerification(Boolean(setup.installationToken), setup.draft, setup.metadata);
  const command = setup.installationToken ? fleetManagerDockerCommand(setup.installationToken) : null;

  return (
    <div data-testid="fleet-manager-docker-step">
      <FirstRunHeading headline="Start runners on this VM" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Run this command in the SuperPlane install directory. It starts Fleet Manager on this machine.
        </p>
      </FirstRunHeading>
      {command ? (
        <div className="mt-5">
          <FleetManagerCommandBlock code={command} language="shell" copyLabel="Copy runner command" />
        </div>
      ) : (
        <p className="mt-5 text-sm text-muted-foreground">Preparing the runner command...</p>
      )}
      {setup.tokenError && (
        <div className="mt-5">
          <p role="alert" className="text-sm text-destructive">
            {setup.tokenError}
          </p>
          <Button type="button" className="mt-3" onClick={() => void setup.prepareInstallation()}>
            Try again
          </Button>
        </div>
      )}
      <div className="mt-5">
        <FleetManagerVerificationStatus
          status={verification.status}
          onRetry={verification.retry}
          copy={{
            waiting: "Waiting for Fleet Manager on this VM...",
            timeout: "Verification timed out. Check the command in the install directory and retry.",
          }}
        />
      </div>
      <div className="mt-7 flex flex-wrap gap-3">
        {verification.status === "verified" && (
          <Button type="button" onClick={onFinish}>
            Finish installation
          </Button>
        )}
        <FinishLaterButton onFinishLater={onFinishLater} />
      </div>
    </div>
  );
}

function InstallStep({ setup, onFinish, onFinishLater }: WizardProps) {
  const verification = useFleetManagerVerification(true, setup.draft, setup.metadata);
  const helmCommand = useMemo(() => {
    if (!setup.metadata || !setup.draft) return null;
    return fleetManagerHelmCommand(setup.metadata, setup.draft.namespace);
  }, [setup.draft, setup.metadata]);

  return (
    <div data-testid="fleet-manager-install-step">
      <FirstRunHeading headline="Install the chart" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Run this command in the same directory. It reads {FLEET_MANAGER_VALUES_FILE}.
        </p>
      </FirstRunHeading>
      {helmCommand && (
        <div className="mt-5">
          <FleetManagerCommandBlock code={helmCommand} language="shell" copyLabel="Copy install command" />
        </div>
      )}
      <div className="mt-5">
        <FleetManagerVerificationStatus status={verification.status} onRetry={verification.retry} />
      </div>
      <div className="mt-7 flex flex-wrap gap-3">
        {verification.status === "verified" && (
          <Button type="button" onClick={onFinish}>
            Finish installation
          </Button>
        )}
        <FinishLaterButton onFinishLater={onFinishLater} />
      </div>
    </div>
  );
}

function FinishLaterButton({ onFinishLater }: { onFinishLater: () => void }) {
  return (
    <Button type="button" variant="ghost" onClick={onFinishLater}>
      Finish later
    </Button>
  );
}

function ValuesCommandPending({
  setup,
  onRetry,
  onFinishLater,
}: {
  setup: FleetManagerSetupState;
  onRetry: () => void;
  onFinishLater: () => void;
}) {
  return (
    <div data-testid="fleet-manager-values-step">
      <FirstRunHeading headline="Create the chart values" size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">
          Run this command. It writes {FLEET_MANAGER_VALUES_FILE}.
        </p>
      </FirstRunHeading>
      {setup.tokenError ? (
        <div className="mt-5">
          <p role="alert" className="text-sm text-destructive">
            {setup.tokenError}
          </p>
          <div className="mt-3 flex flex-wrap gap-3">
            <Button type="button" onClick={onRetry}>
              Try again
            </Button>
            <FinishLaterButton onFinishLater={onFinishLater} />
          </div>
        </div>
      ) : (
        <>
          <p className="mt-5 text-sm text-muted-foreground">Preparing the values command...</p>
          <div className="mt-7">
            <FinishLaterButton onFinishLater={onFinishLater} />
          </div>
        </>
      )}
    </div>
  );
}

function usePrepareValuesCommand(step: FleetManagerWizardStep, setup: FleetManagerSetupState) {
  const started = useRef(false);

  useEffect(() => {
    const preparingDocker = step === "docker" && !setup.installationToken;
    const preparingValues = step === "values" && !setup.values;
    if ((!preparingDocker && !preparingValues) || started.current) return;
    started.current = true;
    void setup.prepareInstallation();
  }, [setup, step]);
}

export function FleetManagerWizardScreen(props: WizardProps) {
  const omittedStep = props.setup.metadata
    ? omittedFleetWizardStep(props.step, props.setup.metadata.installationType)
    : null;
  usePrepareValuesCommand(omittedStep ?? props.step, props.setup);

  if (props.setup.loadError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        SuperPlane could not load Fleet Manager setup. Refresh the page and try again.
      </p>
    );
  }
  if (!props.setup.metadata || !props.setup.draft || omittedStep) {
    return <p className="text-sm text-muted-foreground">Loading Fleet Manager setup...</p>;
  }

  return <FleetManagerWizardSteps {...props} />;
}

function FleetManagerWizardSteps(props: WizardProps) {
  const back = props.showBackButton ? props.onBack : undefined;
  const stepErrors = props.step === "fleets" ? props.setup.validationErrors : [];
  const continueToNextStep = async () => {
    const createToken =
      props.step === "fleets" || (props.step === "provider" && props.setup.draft?.provider === "docker");
    if (createToken && !(await props.setup.prepareInstallation())) return;
    props.onNext();
  };
  const continueActions = (
    <>
      <WizardActions
        primary={props.step === "fleets" ? "Generate install command" : "Continue"}
        onPrimary={() => void continueToNextStep()}
        primaryDisabled={stepErrors.length > 0 || props.setup.isPreparingInstallation}
        showBack={props.showBackButton}
        onBack={back}
      />
      {props.setup.tokenError && (
        <p role="alert" className="mt-3 text-sm text-destructive">
          {props.setup.tokenError}
        </p>
      )}
    </>
  );

  if (props.step === "docker") return <DockerStep {...props} />;
  if (props.step === "provider") return <ProviderStep setup={props.setup} actions={continueActions} />;
  if (props.step === "fleets") return <FleetsStep setup={props.setup} actions={continueActions} />;
  if (props.step === "values") {
    if (!props.setup.values) {
      return (
        <ValuesCommandPending
          setup={props.setup}
          onRetry={() => void props.setup.prepareInstallation()}
          onFinishLater={props.onFinishLater}
        />
      );
    }
    return <ValuesStep setup={props.setup} actions={continueActions} />;
  }
  return <InstallStep {...props} />;
}
