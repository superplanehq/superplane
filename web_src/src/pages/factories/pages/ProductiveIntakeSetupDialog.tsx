import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft } from "lucide-react";

import { ProductiveCompleteStep, ProductiveConnectionStep, ProductiveProjectStep } from "./ProductiveIntakeSetupSteps";
import {
  type ProductiveIntakeSetupModel,
  type ProductiveSetupStep,
  useProductiveIntakeSetup,
} from "./useProductiveIntakeSetup";

interface ProductiveIntakeSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  onClose: () => void;
}

export function ProductiveIntakeSetupDialog(props: ProductiveIntakeSetupDialogProps) {
  const setup = useProductiveIntakeSetup(props.organizationId, props.factoryId, props.open);
  return (
    <>
      <ProductiveSetupView open={props.open && !setup.connectOpen} setup={setup} onClose={props.onClose} />
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

function ProductiveSetupView({
  open,
  setup,
  onClose,
}: {
  open: boolean;
  setup: ProductiveIntakeSetupModel;
  onClose: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="gap-0 p-0 sm:max-w-xl" showCloseButton data-testid="productive-intake-setup">
        <SetupHeader step={setup.step} onBack={() => setup.setStep("connection")} />
        <div className="max-h-[min(32rem,65vh)] overflow-y-auto px-5 py-5">
          <SetupStepContent setup={setup} />
          {setup.error ? (
            <p className="workspace-body-text mt-4 text-destructive" role="alert">
              {setup.error}
            </p>
          ) : null}
        </div>
        <SetupFooter setup={setup} onClose={onClose} />
      </DialogContent>
    </Dialog>
  );
}

function SetupHeader({ step, onBack }: { step: ProductiveSetupStep; onBack: () => void }) {
  return (
    <DialogHeader className="border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2">
        {step === "project" ? (
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={onBack}
            aria-label="Go back"
          >
            <ArrowLeft className="size-4" />
          </button>
        ) : null}
        <IntegrationIcon integrationName="productive" className="size-6" size={24} />
        <div>
          <DialogTitle className="text-[15px] font-semibold">{setupTitle(step)}</DialogTitle>
          <DialogDescription className="workspace-body-text mt-1 text-muted-foreground">
            {setupDescription(step)}
          </DialogDescription>
        </div>
      </div>
    </DialogHeader>
  );
}

function SetupStepContent({ setup }: { setup: ProductiveIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ProductiveConnectionStep
        integrations={setup.productiveIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
        onConnect={() => setup.setConnectOpen(true)}
      />
    );
  }
  if (setup.step === "project") {
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
  return <ProductiveCompleteStep />;
}

function SetupFooter({ setup, onClose }: { setup: ProductiveIntakeSetupModel; onClose: () => void }) {
  return (
    <footer className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
      <span className="text-[12px] text-muted-foreground">{stepProgress(setup.step)}</span>
      <div className="flex items-center gap-2">
        {setup.step === "connection" ? (
          <Button type="button" disabled={!setup.integrationId} onClick={() => setup.setStep("project")}>
            Select project
          </Button>
        ) : null}
        {setup.step === "project" ? (
          <Button
            type="button"
            disabled={!setup.projectId || setup.createIntake.isPending}
            onClick={setup.createBoundIntake}
          >
            {setup.createIntake.isPending ? "Creating intake…" : "Finish setup"}
          </Button>
        ) : null}
        {setup.step === "complete" ? (
          <Button type="button" onClick={onClose}>
            View backlog
          </Button>
        ) : null}
      </div>
    </footer>
  );
}

function setupTitle(step: ProductiveSetupStep): string {
  if (step === "connection") return "Connect Productive.io";
  if (step === "project") return "Choose a project";
  return "Setup complete";
}

function setupDescription(step: ProductiveSetupStep): string {
  if (step === "connection") return "Choose the Productive.io account that SuperPlane will monitor.";
  if (step === "project") return "SuperPlane adds the newest open tasks of this project to the Backlog.";
  return "The intake now listens for new Productive.io tasks.";
}

function stepProgress(step: ProductiveSetupStep): string {
  if (step === "complete") return "Complete";
  return `Step ${step === "connection" ? 1 : 2} of 2`;
}
