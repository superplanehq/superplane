import { Button } from "@/components/ui/button";
import { integrationDetailPath } from "@/lib/integrationSettingsPaths";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";

import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { ProductiveConnectionStep, ProductiveProjectStep } from "./ProductiveIntakeSetupSteps";
import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";
import { type ProductiveIntakeSetupModel, useProductiveIntakeSetup } from "./useProductiveIntakeSetup";

interface ProductiveIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  /** Base path of the organization integrations settings pages. */
  integrationsBasePath: string;
  onClose: () => void;
  onCreated: () => void;
}

export function ProductiveIntakeSetupDialog(props: ProductiveIntakeSetupDialogProps) {
  const setup = useProductiveIntakeSetup(props.organizationId, props.factoryId);
  const title =
    setup.step === "connection"
      ? PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepConnect
      : PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProject;
  const helper =
    setup.step === "connection"
      ? PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepConnectHelper
      : PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProjectHelper;
  const onConnectionStep = setup.step === "connection" && !setup.connectedQuery.isLoading;
  const hasConnections = setup.productiveIntegrations.length > 0;
  // A broken account is only replaceable while Connect stays reachable, so the
  // action also shows next to an existing connection, not only on an empty list.
  const showConnectAction = onConnectionStep;
  const hideEmptyConnectionBody = onConnectionStep && !hasConnections;

  return (
    <>
      <IntakeSetupWizard
        testId="productive-intake-setup"
        integrationName="Productive.io"
        step={setup.step}
        title={title}
        helper={helper}
        stepAction={
          showConnectAction ? (
            <Button
              type="button"
              variant={hasConnections ? "outline" : "default"}
              size={hasConnections ? "sm" : "default"}
              onClick={() => setup.setConnectOpen(true)}
              data-testid="productive-setup-connect"
            >
              {hasConnections
                ? PRODUCTIVE_INTAKE_SETUP_COPY.wizardConnectAnother
                : PRODUCTIVE_INTAKE_SETUP_COPY.wizardConnect}
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
        {hideEmptyConnectionBody && !setup.error ? null : (
          <div>
            <SetupStepBody setup={setup} integrationsBasePath={props.integrationsBasePath} />
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
        integrationDefinition={setup.productiveDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Productive"
        existingIntegrationNames={setup.existingNames}
        hiddenFieldNames={["region"]}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function SetupStepBody({
  setup,
  integrationsBasePath,
}: {
  setup: ProductiveIntakeSetupModel;
  integrationsBasePath: string;
}) {
  if (setup.step === "connection") {
    return (
      <ProductiveConnectionStep
        integrations={setup.productiveIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
      />
    );
  }

  return (
    <ProductiveProjectStep
      projects={setup.projectsQuery.data ?? []}
      selectedId={setup.projectId}
      loading={setup.projectsQuery.isLoading}
      error={setup.projectsQuery.isError}
      repairHref={setup.integrationId ? integrationDetailPath(integrationsBasePath, setup.integrationId) : undefined}
      onSelect={setup.setProjectId}
      onRetry={() => void setup.projectsQuery.refetch()}
    />
  );
}

function SetupFooter({ setup, onCreated }: { setup: ProductiveIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <Button
        type="button"
        className="w-full"
        disabled={!setup.integrationId}
        onClick={() => setup.setStep("project")}
        data-testid="productive-setup-continue"
      >
        {PRODUCTIVE_INTAKE_SETUP_COPY.wizardContinue}
      </Button>
    );
  }

  return (
    <div className="space-y-3">
      <IntakeSkipInitialImportField
        checked={!setup.skipInitialImport}
        onCheckedChange={(importExisting) => setup.setSkipInitialImport(!importExisting)}
        helper={
          setup.skipInitialImport
            ? PRODUCTIVE_INTAKE_SETUP_COPY.importExistingHelperOff
            : PRODUCTIVE_INTAKE_SETUP_COPY.importExistingHelper
        }
        testId="productive-skip-initial-import"
      />
      <Button
        type="button"
        className="w-full"
        disabled={!setup.projectId || setup.createIntake.isPending}
        onClick={() => {
          void setup.createBoundIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="productive-setup-finish"
      >
        {setup.createIntake.isPending
          ? PRODUCTIVE_INTAKE_SETUP_COPY.wizardFinishing
          : PRODUCTIVE_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
