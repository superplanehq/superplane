import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { Check, Loader2 } from "lucide-react";

import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";
import { DatadogServicePicker } from "./DatadogServicePicker";
import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { useDatadogIntakeSetup, type DatadogIntakeSetupModel } from "./useDatadogIntakeSetup";

interface DatadogIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: () => void;
}

export function DatadogIntakeSetupDialog(props: DatadogIntakeSetupDialogProps) {
  const setup = useDatadogIntakeSetup(props.organizationId, props.factoryId);
  const title =
    setup.step === "connection"
      ? DATADOG_INTAKE_SETUP_COPY.wizardStepConnect
      : DATADOG_INTAKE_SETUP_COPY.wizardStepService;
  const helper =
    setup.step === "connection"
      ? DATADOG_INTAKE_SETUP_COPY.wizardStepConnectHelper
      : DATADOG_INTAKE_SETUP_COPY.wizardStepServiceHelper;
  const showConnectAction =
    setup.step === "connection" && !setup.connectedQuery.isLoading && setup.datadogIntegrations.length === 0;

  return (
    <>
      <IntakeSetupWizard
        testId="datadog-intake-setup"
        integrationName="Datadog"
        step={setup.step}
        title={title}
        helper={helper}
        resourceStepLabel="Choose service"
        resourceStepCaption="Awaiting service"
        stepAction={
          showConnectAction ? (
            <Button type="button" onClick={() => setup.setConnectOpen(true)} data-testid="datadog-setup-connect">
              {DATADOG_INTAKE_SETUP_COPY.wizardConnect}
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
        integrationDefinition={setup.datadogDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Datadog"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function SetupStepBody({ setup }: { setup: DatadogIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ConnectionStep
        integrations={setup.datadogIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
      />
    );
  }

  return (
    <DatadogServicePicker
      services={setup.servicesQuery.data ?? []}
      serviceName={setup.serviceName}
      loading={setup.servicesQuery.isLoading}
      error={setup.servicesQuery.isError}
      onChange={setup.setServiceName}
      onRetry={() => void setup.servicesQuery.refetch()}
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
        {DATADOG_INTAKE_SETUP_COPY.wizardConnectionsLoading}
      </p>
    );
  }
  if (integrations.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{DATADOG_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`datadog-connection-${id}`}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Datadog"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function SetupFooter({ setup, onCreated }: { setup: DatadogIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <div>
        <Button
          type="button"
          className="w-full"
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
          data-testid="datadog-setup-continue"
        >
          {DATADOG_INTAKE_SETUP_COPY.wizardContinue}
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
            ? DATADOG_INTAKE_SETUP_COPY.importExistingHelperOff
            : DATADOG_INTAKE_SETUP_COPY.importExistingHelper
        }
        testId="datadog-skip-initial-import"
      />
      <Button
        type="button"
        className="w-full"
        disabled={!setup.serviceName.trim() || setup.createIntake.isPending}
        onClick={() => {
          void setup.createBoundIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="datadog-setup-finish"
      >
        {setup.createIntake.isPending
          ? DATADOG_INTAKE_SETUP_COPY.wizardFinishing
          : DATADOG_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
