import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { Check, Loader2 } from "lucide-react";

import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";
import { SentryProjectPicker } from "./SentryProjectPicker";
import { useSentryIntakeSetup, type SentryIntakeSetupModel } from "./useSentryIntakeSetup";

interface SentryIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: () => void;
}

export function SentryIntakeSetupDialog(props: SentryIntakeSetupDialogProps) {
  const setup = useSentryIntakeSetup(props.organizationId, props.factoryId);
  const title =
    setup.step === "connection"
      ? SENTRY_INTAKE_SETUP_COPY.wizardStepConnect
      : SENTRY_INTAKE_SETUP_COPY.wizardStepProject;
  const helper =
    setup.step === "connection"
      ? SENTRY_INTAKE_SETUP_COPY.wizardStepConnectHelper
      : SENTRY_INTAKE_SETUP_COPY.wizardStepProjectHelper;
  const showConnectAction =
    setup.step === "connection" && !setup.connectedQuery.isLoading && setup.sentryIntegrations.length === 0;

  return (
    <>
      <IntakeSetupWizard
        testId="sentry-intake-setup"
        integrationName="Sentry"
        step={setup.step}
        title={title}
        helper={helper}
        stepAction={
          showConnectAction ? (
            <Button
              type="button"
              disabled={setup.connecting}
              onClick={() => void setup.connectSentry()}
              data-testid="sentry-setup-connect"
            >
              {setup.connecting ? "Connecting..." : SENTRY_INTAKE_SETUP_COPY.wizardConnect}
            </Button>
          ) : undefined
        }
        footer={<SetupFooter setup={setup} onCreated={props.onCreated} />}
        onBack={() => {
          if (setup.step === "project") {
            setup.returnToConnection();
            return;
          }
          props.onClose();
        }}
      >
        {showConnectAction && !setup.error ? null : (
          <div>
            <SetupStepBody setup={setup} />
            {setup.error ? (
              <p className="workspace-body-text mt-4 text-destructive" role="alert">
                {setup.error}
              </p>
            ) : null}
          </div>
        )}
      </IntakeSetupWizard>
      <IntegrationCreateDialog
        open={setup.connectOpen}
        onOpenChange={setup.setConnectOpen}
        integrationDefinition={setup.sentryDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Sentry"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function SetupStepBody({ setup }: { setup: SentryIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ConnectionStep
        integrations={setup.sentryIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
      />
    );
  }

  return (
    <SentryProjectPicker
      projects={setup.projectsQuery.data ?? []}
      selectedIds={setup.projectIds}
      loading={setup.projectsQuery.isLoading}
      error={setup.projectsQuery.isError}
      onToggle={setup.toggleProject}
      onRetry={() => void setup.projectsQuery.refetch()}
    />
  );
}

function ConnectionStep({
  integrations,
  selectedId,
  loading,
  onSelect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  onSelect: (id: string) => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading Sentry connections...
      </p>
    );
  }
  if (integrations.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{SENTRY_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`sentry-connection-${id}`}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Sentry"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function SetupFooter({ setup, onCreated }: { setup: SentryIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <div>
        <Button
          type="button"
          className="w-full"
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
          data-testid="sentry-setup-continue"
        >
          {SENTRY_INTAKE_SETUP_COPY.wizardContinue}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <IntakeSkipInitialImportField
        checked={!setup.skipInitialImport}
        onCheckedChange={(importExisting) => setup.setSkipInitialImport(!importExisting)}
        helper={
          setup.skipInitialImport
            ? SENTRY_INTAKE_SETUP_COPY.importExistingHelperOff
            : SENTRY_INTAKE_SETUP_COPY.importExistingHelper
        }
        testId="sentry-skip-initial-import"
      />
      <Button
        type="button"
        className="w-full"
        disabled={setup.projectIds.length === 0 || setup.createIntake.isPending}
        onClick={() => {
          void setup.createBoundIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="sentry-setup-finish"
      >
        {setup.createIntake.isPending
          ? SENTRY_INTAKE_SETUP_COPY.wizardFinishing
          : SENTRY_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
