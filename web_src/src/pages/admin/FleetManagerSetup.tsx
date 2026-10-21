import { useMemo, useState } from "react";

import {
  FleetManagerCommandBlock,
  FleetManagerInstallCommands,
} from "@/components/fleet-manager-setup/FleetManagerInstallCommands";
import {
  FleetManagerFleetSection,
  FleetManagerProviderSection,
  type FleetManagerSetupState,
} from "@/components/fleet-manager-setup/FleetManagerSections";
import { FleetManagerVerificationStatus } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { Button } from "@/components/ui/button";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useFleetManagerSetup, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import {
  fleetManagerDockerCommand,
  fleetManagerHelmCommand,
  fleetManagerValuesFileCommand,
  type FleetManagerDraft,
} from "@/lib/fleetManagerSetup";
import { fleetProviderChoiceCopy, singleHostInstall } from "@/lib/fleetManagerWizard";

function SectionCard({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="space-y-1">
        <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
        {description && <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>}
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function InstallActions({
  draftIsValid,
  error,
  commands,
  isPreparingInstallation,
  onGenerate,
}: {
  draftIsValid: boolean;
  error?: string;
  commands: { valuesCommand: string; helmCommand: string } | null;
  isPreparingInstallation: boolean;
  onGenerate: () => void;
}) {
  if (!draftIsValid) {
    return (
      <>
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
        <Button type="button" disabled>
          Generate install command
        </Button>
      </>
    );
  }
  if (commands) {
    return <FleetManagerInstallCommands valuesCommand={commands.valuesCommand} helmCommand={commands.helmCommand} />;
  }
  return (
    <Button type="button" onClick={onGenerate} disabled={isPreparingInstallation}>
      Generate install command
    </Button>
  );
}

function installSectionCopy(isDocker: boolean, draftIsValid: boolean) {
  if (!draftIsValid) {
    return { title: isDocker ? "Start runners on this VM" : "Install Fleet Manager", description: undefined };
  }
  if (isDocker) {
    return {
      title: "Start runners on this VM",
      description: "Run this command in the SuperPlane install directory.",
    };
  }
  return {
    title: "Install Fleet Manager",
    description:
      "Run the first command to write fleet-manager-values.yaml. Run the second command to install the chart.",
  };
}

function ProviderChoice({ setup }: { setup: FleetManagerSetupState }) {
  const copy = fleetProviderChoiceCopy(singleHostInstall(setup.metadata?.installationType));
  return (
    <SectionCard title={copy.title} description={copy.description}>
      <FleetManagerProviderSection setup={setup} />
    </SectionCard>
  );
}

function FleetManagerSetupForm({ setup }: { setup: FleetManagerSetupState }) {
  const [submittedDraft, setSubmittedDraft] = useState<FleetManagerDraft | null>(null);
  const draft = setup.draft;
  const isDocker = draft?.provider === "docker";
  const dockerCommand = isDocker && setup.installationToken ? fleetManagerDockerCommand(setup.installationToken) : "";
  const draftIsValid = setup.validationErrors.length === 0;
  const submittedDraftIsCurrent = submittedDraft !== null && draft === submittedDraft;
  const verification = useFleetManagerVerification(submittedDraftIsCurrent, submittedDraft, setup.metadata);
  const commands = useMemo(() => {
    if (!setup.metadata || !draft || !setup.values) return null;
    return {
      valuesCommand: fleetManagerValuesFileCommand(setup.values),
      helmCommand: fleetManagerHelmCommand(setup.metadata, draft.namespace),
    };
  }, [draft, setup.metadata, setup.values]);
  const installCopy = installSectionCopy(isDocker, draftIsValid);
  const canCheck =
    draftIsValid && !submittedDraftIsCurrent && (isDocker ? Boolean(dockerCommand) : Boolean(setup.values));

  return (
    <div className="space-y-5">
      <ProviderChoice setup={setup} />
      {isDocker ? null : (
        <SectionCard
          title="Configure the default fleet"
          description="Set the cloud machine type and how many machines to keep ready."
        >
          <FleetManagerFleetSection setup={setup} />
        </SectionCard>
      )}
      <SectionCard title={installCopy.title} description={installCopy.description}>
        <div className="space-y-5">
          {isDocker && dockerCommand ? (
            <FleetManagerCommandBlock code={dockerCommand} language="shell" copyLabel="Copy runner command" />
          ) : (
            <InstallActions
              draftIsValid={draftIsValid}
              error={setup.validationErrors[0]}
              commands={
                commands && setup.values
                  ? { valuesCommand: commands.valuesCommand, helmCommand: commands.helmCommand }
                  : null
              }
              isPreparingInstallation={setup.isPreparingInstallation}
              onGenerate={() => void setup.prepareInstallation()}
            />
          )}
          {setup.tokenError ? (
            <p role="alert" className="text-sm text-destructive">
              {setup.tokenError}
            </p>
          ) : null}
          <Button type="button" onClick={() => draft && setSubmittedDraft(draft)} disabled={!canCheck}>
            Check installation
          </Button>
          {submittedDraftIsCurrent ? (
            <FleetManagerVerificationStatus status={verification.status} onRetry={verification.retry} />
          ) : null}
        </div>
      </SectionCard>
    </div>
  );
}

export function FleetManagerSetup() {
  const organizations = useAccountOrganizations();
  const setup = useFleetManagerSetup(true, organizations.data?.[0]?.id ?? "");

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Set up Fleet Manager</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Generate standalone chart values for your runner infrastructure.
        </p>
      </header>

      {setup.loadError ? (
        <p role="alert" className="text-sm text-destructive">
          SuperPlane could not load Fleet Manager setup. Refresh the page and try again.
        </p>
      ) : !setup.metadata || !setup.draft ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading Fleet Manager setup...</p>
      ) : (
        <FleetManagerSetupForm setup={setup} />
      )}
    </div>
  );
}
