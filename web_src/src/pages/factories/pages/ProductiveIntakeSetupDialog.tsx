import { Button } from "@/components/ui/button";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";

import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { ProductiveConnectionStep, ProductiveProjectStep } from "./ProductiveIntakeSetupSteps";
import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";
import { type ProductiveIntakeSetupModel, useProductiveIntakeSetup } from "./useProductiveIntakeSetup";

interface ProductiveIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
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
  const showConnectAction =
    setup.step === "connection" && !setup.connectedQuery.isLoading && setup.productiveIntegrations.length === 0;

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
            <Button type="button" onClick={() => setup.setConnectOpen(true)} data-testid="productive-setup-connect">
              {PRODUCTIVE_INTAKE_SETUP_COPY.wizardConnect}
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

function SetupStepBody({ setup }: { setup: ProductiveIntakeSetupModel }) {
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
  );
}
