import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft } from "lucide-react";

import { NotionCompleteStep, NotionConnectionStep, NotionDatabaseStep } from "./NotionIntakeSetupSteps";
import { type NotionIntakeSetupModel, type NotionSetupStep, useNotionIntakeSetup } from "./useNotionIntakeSetup";

interface NotionIntakeSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  onClose: () => void;
}

export function NotionIntakeSetupDialog(props: NotionIntakeSetupDialogProps) {
  const setup = useNotionIntakeSetup(props.organizationId, props.factoryId, props.open);
  return (
    <>
      <NotionSetupView open={props.open && !setup.connectOpen} setup={setup} onClose={props.onClose} />
      <IntegrationCreateDialog
        open={setup.connectOpen}
        onOpenChange={setup.setConnectOpen}
        integrationDefinition={setup.notionDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Notion"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function NotionSetupView({
  open,
  setup,
  onClose,
}: {
  open: boolean;
  setup: NotionIntakeSetupModel;
  onClose: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="gap-0 p-0 sm:max-w-xl" showCloseButton data-testid="notion-intake-setup">
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

function SetupHeader({ step, onBack }: { step: NotionSetupStep; onBack: () => void }) {
  return (
    <DialogHeader className="border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2">
        {step === "database" ? (
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={onBack}
            aria-label="Go back"
          >
            <ArrowLeft className="size-4" />
          </button>
        ) : null}
        <IntegrationIcon integrationName="notion" className="size-6" size={24} />
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

function SetupStepContent({ setup }: { setup: NotionIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <NotionConnectionStep
        integrations={setup.notionIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
        onConnect={() => setup.setConnectOpen(true)}
      />
    );
  }
  if (setup.step === "database") {
    return (
      <NotionDatabaseStep
        databases={setup.databasesQuery.data ?? []}
        selectedId={setup.databaseId}
        loading={setup.databasesQuery.isLoading}
        error={setup.databasesQuery.isError}
        onSelect={setup.setDatabaseId}
        onRetry={() => void setup.databasesQuery.refetch()}
      />
    );
  }
  return <NotionCompleteStep />;
}

function SetupFooter({ setup, onClose }: { setup: NotionIntakeSetupModel; onClose: () => void }) {
  return (
    <footer className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
      <span className="text-[12px] text-muted-foreground">{stepProgress(setup.step)}</span>
      <div className="flex items-center gap-2">
        {setup.step === "connection" ? (
          <Button type="button" disabled={!setup.integrationId} onClick={() => setup.setStep("database")}>
            Select database
          </Button>
        ) : null}
        {setup.step === "database" ? (
          <Button
            type="button"
            disabled={!setup.databaseId || setup.createIntake.isPending}
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

function setupTitle(step: NotionSetupStep): string {
  if (step === "connection") return "Connect Notion";
  if (step === "database") return "Choose a database";
  return "Setup complete";
}

function setupDescription(step: NotionSetupStep): string {
  if (step === "connection") return "Choose the Notion workspace that SuperPlane will monitor.";
  if (step === "database") return "SuperPlane adds the newest pages of this database to the Backlog.";
  return "The intake now listens for new Notion pages.";
}

function stepProgress(step: NotionSetupStep): string {
  if (step === "complete") return "Complete";
  return `Step ${step === "connection" ? 1 : 2} of 2`;
}
