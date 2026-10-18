import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { Check, Loader2 } from "lucide-react";
import { useLocation } from "react-router";

import { IntakeSetupWizard } from "./IntakeSetupWizard";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { LinearProjectPicker } from "./LinearProjectPicker";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";
import { useLinearIntakeSetup, type LinearIntakeSetupModel } from "./useLinearIntakeSetup";

interface LinearIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: () => void;
  selectIntegrationId?: string;
}

export function LinearIntakeSetupDialog(props: LinearIntakeSetupDialogProps) {
  const location = useLocation();
  const setup = useLinearIntakeSetup(props.organizationId, props.factoryId, props.selectIntegrationId);
  const title =
    setup.step === "connection"
      ? LINEAR_INTAKE_SETUP_COPY.wizardStepConnect
      : LINEAR_INTAKE_SETUP_COPY.wizardStepProject;
  const helper =
    setup.step === "connection"
      ? LINEAR_INTAKE_SETUP_COPY.wizardStepConnectHelper
      : LINEAR_INTAKE_SETUP_COPY.wizardStepProjectHelper;
  const showConnectAction =
    setup.step === "connection" && !setup.connectedQuery.isLoading && setup.linearIntegrations.length === 0;

  return (
    <>
      <IntakeSetupWizard
        testId="linear-intake-setup"
        integrationName="Linear"
        step={setup.step}
        title={title}
        helper={helper}
        resourceStepLabel="Choose projects"
        resourceStepCaption="Awaiting projects"
        stepAction={
          showConnectAction ? (
            <Button
              type="button"
              disabled={setup.connecting}
              onClick={() => void setup.connectLinear()}
              data-testid="linear-setup-connect"
            >
              {setup.connecting ? LINEAR_INTAKE_SETUP_COPY.wizardConnecting : LINEAR_INTAKE_SETUP_COPY.wizardConnect}
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
        integrationDefinition={setup.linearDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Linear"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
        setupReturnTo={location.pathname}
      />
    </>
  );
}

function SetupStepBody({ setup }: { setup: LinearIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ConnectionStep
        integrations={setup.linearIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
      />
    );
  }

  return (
    <LinearProjectPicker
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
        {LINEAR_INTAKE_SETUP_COPY.wizardConnectionsLoading}
      </p>
    );
  }
  if (integrations.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{LINEAR_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`linear-connection-${id}`}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Linear"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function SetupFooter({ setup, onCreated }: { setup: LinearIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <div>
        <Button
          type="button"
          className="w-full"
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
          data-testid="linear-setup-continue"
        >
          {LINEAR_INTAKE_SETUP_COPY.wizardContinue}
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
            ? LINEAR_INTAKE_SETUP_COPY.importExistingHelperOff
            : LINEAR_INTAKE_SETUP_COPY.importExistingHelper
        }
        testId="linear-skip-initial-import"
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
        data-testid="linear-setup-finish"
      >
        {setup.createIntake.isPending
          ? LINEAR_INTAKE_SETUP_COPY.wizardFinishing
          : LINEAR_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
