import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft } from "lucide-react";

import { JiraCompleteStep, JiraConnectionStep, JiraProjectStep } from "./JiraIntakeSetupSteps";
import { type JiraIntakeSetupModel, type JiraSetupStep, useJiraIntakeSetup } from "./useJiraIntakeSetup";

interface JiraIntakeSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  onClose: () => void;
}

export function JiraIntakeSetupDialog(props: JiraIntakeSetupDialogProps) {
  const setup = useJiraIntakeSetup(props.organizationId, props.factoryId, props.open);
  return (
    <>
      <JiraSetupView open={props.open && !setup.connectOpen} setup={setup} onClose={props.onClose} />
      <IntegrationCreateDialog
        open={setup.connectOpen}
        onOpenChange={setup.setConnectOpen}
        integrationDefinition={setup.jiraDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Jira"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function JiraSetupView({ open, setup, onClose }: { open: boolean; setup: JiraIntakeSetupModel; onClose: () => void }) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="gap-0 p-0 sm:max-w-xl" showCloseButton data-testid="jira-intake-setup">
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

function SetupHeader({ step, onBack }: { step: JiraSetupStep; onBack: () => void }) {
  return (
    <DialogHeader className="border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2">
        {step === "project" ? (
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={onBack}
            aria-label="Back to connection"
          >
            <ArrowLeft className="size-4" />
          </button>
        ) : null}
        <IntegrationIcon integrationName="jira" className="size-5" size={20} />
        <DialogTitle className="text-[15px] font-semibold">Add Jira intake</DialogTitle>
      </div>
      <DialogDescription className="sr-only">Connect Jira and choose a project for intake.</DialogDescription>
    </DialogHeader>
  );
}

function SetupStepContent({ setup }: { setup: JiraIntakeSetupModel }) {
  switch (setup.step) {
    case "connection":
      return (
        <JiraConnectionStep
          integrations={setup.jiraIntegrations}
          selectedId={setup.integrationId}
          loading={setup.connectedQuery.isLoading}
          onSelect={setup.setIntegrationId}
          onConnect={() => setup.setConnectOpen(true)}
        />
      );
    case "project":
      return (
        <JiraProjectStep
          projects={setup.projectsQuery.data ?? []}
          selectedId={setup.projectId}
          loading={setup.projectsQuery.isLoading}
          error={setup.projectsQuery.isError}
          onSelect={setup.setProjectId}
          onRetry={() => void setup.projectsQuery.refetch()}
        />
      );
    case "complete":
      return <JiraCompleteStep />;
    default:
      return null;
  }
}

function SetupFooter({ setup, onClose }: { setup: JiraIntakeSetupModel; onClose: () => void }) {
  if (setup.step === "complete") {
    return (
      <div className="border-t border-border px-5 py-4">
        <Button type="button" className="w-full" onClick={onClose}>
          Done
        </Button>
      </div>
    );
  }

  if (setup.step === "connection") {
    return (
      <div className="border-t border-border px-5 py-4">
        <Button
          type="button"
          className="w-full"
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
        >
          Continue
        </Button>
      </div>
    );
  }

  return (
    <div className="border-t border-border px-5 py-4">
      <Button
        type="button"
        className="w-full"
        disabled={!setup.projectId || setup.createIntake.isPending}
        onClick={() => void setup.createBoundIntake()}
      >
        {setup.createIntake.isPending ? "Creating intake…" : "Create intake"}
      </Button>
    </div>
  );
}
