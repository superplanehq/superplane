import { useMemo, useState } from "react";

import {
  FleetManagerFleetSection,
  FleetManagerInfrastructureSection,
  FleetManagerProviderSection,
} from "@/components/fleet-manager-setup/FleetManagerSections";
import { FleetManagerVerificationStatus } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { Button } from "@/components/ui/button";
import { useFleetManagerSetup, useFleetManagerVerification } from "@/hooks/useFleetManagerSetup";
import { downloadFleetManagerValues } from "@/lib/downloadFleetManagerValues";
import { fleetManagerInstallCommand, type FleetManagerDraft, type FleetManagerProvider } from "@/lib/fleetManagerSetup";
import { CopyButton } from "@/ui/CopyButton";
import { Download, TriangleAlert } from "lucide-react";

const INSTALL_INSTRUCTION = "Run this command where kubectl can access your cluster.";

const providerNames: Record<FleetManagerProvider, string> = {
  aws: "AWS",
  gcp: "GCP",
  azure: "Azure",
};

function providerValues(values: string) {
  const marker = "\nfleets:\n";
  const index = values.indexOf(marker);
  return index === -1 ? values : values.slice(index + 1);
}

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

function ValuesBlock({
  values,
  command,
  provider,
}: {
  values: string;
  command: string;
  provider: FleetManagerProvider;
}) {
  const providerName = providerNames[provider];

  return (
    <div className="space-y-4" data-testid="fleet-manager-values">
      <div className="flex gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-950 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-100">
        <TriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
        <p>Replace each public VM image placeholder before production use.</p>
      </div>
      <p className="text-sm text-gray-600 dark:text-gray-300">The values file uses {providerName}.</p>
      <div className="space-y-2">
        <p className="text-xs font-medium text-gray-500 dark:text-gray-400">{providerName} values</p>
        <pre className="max-h-96 overflow-auto rounded-md border border-slate-200 bg-slate-50 p-4 text-left font-mono text-xs text-gray-800 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100">
          <code>{providerValues(values)}</code>
        </pre>
      </div>
      <div className="space-y-2">
        <p className="text-xs font-medium text-gray-500 dark:text-gray-400">values.yaml</p>
        <pre className="max-h-80 overflow-auto rounded-md border border-slate-200 bg-slate-50 p-4 text-left font-mono text-xs text-gray-800 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100">
          <code>{values}</code>
        </pre>
        <div className="flex flex-wrap gap-2">
          <CopyButton variant="button" text={values}>
            Copy values.yaml
          </CopyButton>
          <Button type="button" variant="outline" onClick={() => downloadFleetManagerValues(values)}>
            <Download className="size-4" aria-hidden />
            Download values.yaml
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <p className="text-sm text-gray-600 dark:text-gray-300">{INSTALL_INSTRUCTION}</p>
        <pre className="overflow-auto rounded-md border border-slate-200 bg-slate-50 p-4 text-left font-mono text-xs text-gray-800 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100">
          <code>{command}</code>
        </pre>
        <CopyButton variant="button" text={command}>
          Copy Helm command
        </CopyButton>
      </div>
    </div>
  );
}

export function FleetManagerSetup() {
  const setup = useFleetManagerSetup();
  const [submittedDraft, setSubmittedDraft] = useState<FleetManagerDraft | null>(null);
  const draftIsValid = setup.validationErrors.length === 0;
  const submittedDraftIsCurrent = submittedDraft !== null && setup.draft === submittedDraft;
  const verification = useFleetManagerVerification(submittedDraftIsCurrent, submittedDraft, setup.metadata);
  const command = useMemo(
    () =>
      setup.metadata && setup.draft
        ? fleetManagerInstallCommand(setup.metadata, setup.draft.namespace, "values.yaml")
        : "",
    [setup.draft, setup.metadata],
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
            title="Confirm cloud infrastructure"
            description="Review the values from Terraform and add missing account details."
          >
            <FleetManagerInfrastructureSection setup={setup} />
          </SectionCard>
          <SectionCard
            title="Install Fleet Manager"
            description={draftIsValid ? "Save this file before you install the standalone chart." : undefined}
          >
            <div className="space-y-5">
              {draftIsValid ? (
                <ValuesBlock values={setup.values} command={command} provider={setup.draft.provider} />
              ) : (
                <p role="alert" className="text-sm text-destructive">
                  {setup.validationErrors[0]}
                </p>
              )}
              <Button
                type="button"
                onClick={() => {
                  if (setup.draft && draftIsValid) setSubmittedDraft(setup.draft);
                }}
                disabled={!draftIsValid || submittedDraftIsCurrent}
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
