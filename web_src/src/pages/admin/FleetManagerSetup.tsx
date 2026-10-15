import { useMemo, useState } from "react";

import {
  FleetManagerFleetSection,
  FleetManagerProviderSection,
} from "@/components/fleet-manager-setup/FleetManagerSections";
import { FleetManagerVerificationStatus } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { Button } from "@/components/ui/button";
import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useFleetManagerSetup, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import { fleetManagerInstallCommand, type FleetManagerDraft } from "@/lib/fleetManagerSetup";
import { CopyButton } from "@/ui/CopyButton";

const INSTALL_INSTRUCTION = "Run these commands where kubectl can access your cluster.";

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

function InstallCommand({ command }: { command: string }) {
  return (
    <div className="space-y-4" data-testid="fleet-manager-install-command">
      <div className="space-y-2">
        <p className="text-sm text-gray-600 dark:text-gray-300">{INSTALL_INSTRUCTION}</p>
        <pre className="max-h-96 overflow-auto rounded-md border border-slate-200 bg-slate-50 p-4 text-left font-mono text-xs text-gray-800 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100">
          <code>{command}</code>
        </pre>
        <CopyButton variant="button" buttonVariant="default" copiedLabel="Copied" text={command}>
          Copy commands
        </CopyButton>
      </div>
    </div>
  );
}

export function FleetManagerSetup() {
  const organizations = useAccountOrganizations();
  const setup = useFleetManagerSetup(true, organizations.data?.[0]?.id ?? "");
  const [submittedDraft, setSubmittedDraft] = useState<FleetManagerDraft | null>(null);
  const draftIsValid = setup.validationErrors.length === 0;
  const submittedDraftIsCurrent = submittedDraft !== null && setup.draft === submittedDraft;
  const verification = useFleetManagerVerification(submittedDraftIsCurrent, submittedDraft, setup.metadata);
  const command = useMemo(
    () =>
      setup.metadata && setup.draft
        ? fleetManagerInstallCommand(setup.metadata, setup.draft.namespace, setup.values)
        : "",
    [setup.draft, setup.metadata, setup.values],
  );

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
        <div className="space-y-5">
          <SectionCard
            title="Choose a cloud provider"
            description="Select the account that will run your fleet machines."
          >
            <FleetManagerProviderSection setup={setup} />
          </SectionCard>
          <SectionCard title="Choose runner fleets" description="Select fleet sizes and set their initial capacity.">
            <FleetManagerFleetSection setup={setup} />
          </SectionCard>
          <SectionCard
            title="Install Fleet Manager"
            description={
              draftIsValid ? "Generate one command with the complete Fleet Manager configuration." : undefined
            }
          >
            <div className="space-y-5">
              {!draftIsValid ? (
                <p role="alert" className="text-sm text-destructive">
                  {setup.validationErrors[0]}
                </p>
              ) : setup.values ? (
                <InstallCommand command={command} />
              ) : (
                <Button
                  type="button"
                  onClick={() => void setup.prepareInstallation()}
                  disabled={setup.isPreparingInstallation}
                >
                  Generate install command
                </Button>
              )}
              {setup.tokenError && (
                <p role="alert" className="text-sm text-destructive">
                  {setup.tokenError}
                </p>
              )}
              <Button
                type="button"
                onClick={() => {
                  if (setup.draft && draftIsValid) setSubmittedDraft(setup.draft);
                }}
                disabled={!draftIsValid || !setup.values || submittedDraftIsCurrent}
              >
                Check installation
              </Button>
              {submittedDraftIsCurrent && (
                <FleetManagerVerificationStatus status={verification.status} onRetry={verification.retry} />
              )}
            </div>
          </SectionCard>
        </div>
      )}
    </div>
  );
}
