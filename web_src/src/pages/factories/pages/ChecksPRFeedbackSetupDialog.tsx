import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { sortConnectedIntegrationsByType } from "@/lib/sortConnectedIntegrations";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft } from "lucide-react";

import { CHECKS_HANDLER_SKIP_INTEGRATIONS } from "./checksPRFeedbackSetup";
import { StatusCheckPicker } from "./StatusCheckPicker";
import {
  integrationDefinitionLabel,
  useChecksPRFeedbackSetup,
  type ChecksPRFeedbackSetupModel,
} from "./useChecksPRFeedbackSetup";
import { PR_FEEDBACK_SETTINGS_COPY, type PRFeedbackSource } from "./prFeedbackSettingsModel";

interface ChecksPRFeedbackSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}

export function ChecksPRFeedbackSetupDialog(props: ChecksPRFeedbackSetupDialogProps) {
  const setup = useChecksPRFeedbackSetup(props.organizationId, props.factoryId, props.repository, props.open);
  return (
    <>
      <ChecksSetupView
        open={props.open && !setup.connectName}
        setup={setup}
        source={props.source}
        onClose={props.onClose}
        onCreated={props.onCreated}
      />
      <IntegrationCreateDialog
        open={Boolean(setup.connectName)}
        onOpenChange={(next) => {
          if (!next) {
            setup.setConnectName(null);
          }
        }}
        integrationDefinition={setup.connectDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName={integrationDefinitionLabel(setup.connectDefinition)}
        existingIntegrationNames={setup.existingNames}
        onCreated={(integrationId) => {
          setup.setRunnerIntegrationIds((current) =>
            current.includes(integrationId) ? current : [...current, integrationId],
          );
          setup.setConnectName(null);
          void setup.connectedQuery?.refetch?.();
        }}
      />
    </>
  );
}

function ChecksSetupView({
  open,
  setup,
  source,
  onClose,
  onCreated,
}: {
  open: boolean;
  setup: ChecksPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="gap-0 p-0 sm:max-w-xl" showCloseButton data-testid="checks-pr-feedback-setup">
        <SetupHeader step={setup.step} onBack={() => setup.setStep("checks")} />
        <div className="max-h-[min(32rem,65vh)] overflow-y-auto px-5 py-5">
          {setup.step === "checks" ? (
            <StatusCheckPicker
              names={setup.checkNames}
              catalog={setup.catalog}
              loading={setup.catalogLoading}
              loadError={setup.catalogQuery.isError}
              onToggle={setup.toggleCheckName}
            />
          ) : (
            <ToolsStep setup={setup} />
          )}
          {setup.error ? (
            <p className="workspace-body-text mt-4 text-destructive" role="alert">
              {setup.error}
            </p>
          ) : null}
        </div>
        <SetupFooter setup={setup} source={source} onClose={onClose} onCreated={onCreated} />
      </DialogContent>
    </Dialog>
  );
}

function SetupHeader({ step, onBack }: { step: "checks" | "tools"; onBack: () => void }) {
  return (
    <DialogHeader className="border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2">
        {step === "tools" ? (
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={onBack}
            aria-label="Go back"
          >
            <ArrowLeft className="size-4" />
          </button>
        ) : null}
        <IntegrationIcon integrationName="github" className="size-6" size={24} />
        <div>
          <DialogTitle className="text-[15px] font-semibold">{PR_FEEDBACK_SETTINGS_COPY.wizardTitle}</DialogTitle>
          <DialogDescription className="workspace-body-text mt-1 text-muted-foreground">
            {step === "checks"
              ? PR_FEEDBACK_SETTINGS_COPY.wizardChecksDescription
              : PR_FEEDBACK_SETTINGS_COPY.wizardToolsDescription}
          </DialogDescription>
        </div>
      </div>
    </DialogHeader>
  );
}

function ToolsStep({ setup }: { setup: ChecksPRFeedbackSetupModel }) {
  const ready = sortConnectedIntegrationsByType(
    (setup.connected ?? []).filter(
      (integration) =>
        integration.status?.state === "ready" &&
        integration.metadata?.id &&
        !CHECKS_HANDLER_SKIP_INTEGRATIONS.has(integration.metadata.integrationName?.toLowerCase() ?? ""),
    ),
  );
  const suggested = setup.available.filter((integration) => setup.suggestedNames.includes(integration.name ?? ""));
  const others = setup.available.filter((integration) => !setup.suggestedNames.includes(integration.name ?? ""));

  return (
    <div className="space-y-5">
      {setup.suggestedNames.length === 0 ? (
        <p className="workspace-body-text text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardToolsUnknown}</p>
      ) : null}
      {suggested.length > 0 ? (
        <IntegrationChoiceList
          title={PR_FEEDBACK_SETTINGS_COPY.wizardSuggested}
          definitions={suggested}
          ready={ready}
          selectedIds={setup.runnerIntegrationIds}
          onToggle={setup.setRunnerIntegrationIds}
          onConnect={setup.setConnectName}
        />
      ) : null}
      {others.length > 0 ? (
        <IntegrationChoiceList
          title={
            setup.suggestedNames.length > 0
              ? PR_FEEDBACK_SETTINGS_COPY.wizardOtherTools
              : PR_FEEDBACK_SETTINGS_COPY.integrationsLabel
          }
          definitions={others}
          ready={ready}
          selectedIds={setup.runnerIntegrationIds}
          onToggle={setup.setRunnerIntegrationIds}
          onConnect={setup.setConnectName}
        />
      ) : null}
    </div>
  );
}

function IntegrationChoiceList({
  title,
  definitions,
  ready,
  selectedIds,
  onToggle,
  onConnect,
}: {
  title: string;
  definitions: Array<{ name?: string; label?: string }>;
  ready: Array<{ metadata?: { id?: string; name?: string; integrationName?: string } }>;
  selectedIds: string[];
  onToggle: (ids: string[]) => void;
  onConnect: (name: string) => void;
}) {
  return (
    <section>
      <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">{title}</h3>
      <ul className="mt-3 flex flex-col gap-2">
        {definitions.map((definition) => {
          const type = definition.name ?? "";
          const instances = ready.filter((item) => item.metadata?.integrationName === type);
          return (
            <li key={type} className="rounded-md border border-border px-3 py-2.5">
              <div className="flex items-center gap-2">
                <IntegrationIcon integrationName={type} className="size-4 shrink-0" size={16} />
                <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
                  {getIntegrationTypeDisplayName(definition.label, type)}
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => onConnect(type)}
                  data-testid={`checks-setup-connect-${type}`}
                >
                  {PR_FEEDBACK_SETTINGS_COPY.wizardConnect}
                </Button>
              </div>
              {instances.map((instance) => {
                const id = instance.metadata?.id ?? "";
                const name = instance.metadata?.name || type;
                const checked = selectedIds.includes(id);
                return (
                  <div key={id} className="mt-2 flex items-center gap-2 pl-6">
                    <Checkbox
                      id={`checks-setup-integration-${id}`}
                      className="cursor-pointer"
                      checked={checked}
                      onChange={(event) => {
                        if (event.currentTarget.checked) {
                          onToggle([...selectedIds, id]);
                          return;
                        }
                        onToggle(selectedIds.filter((item) => item !== id));
                      }}
                      data-testid={`checks-setup-integration-${id}`}
                    />
                    <Label htmlFor={`checks-setup-integration-${id}`} className="cursor-pointer">
                      {name}
                    </Label>
                  </div>
                );
              })}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function SetupFooter({
  setup,
  source,
  onClose,
  onCreated,
}: {
  setup: ChecksPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  return (
    <footer className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
      <span className="text-[12px] text-muted-foreground">
        {setup.step === "checks"
          ? PR_FEEDBACK_SETTINGS_COPY.wizardStepChecks
          : PR_FEEDBACK_SETTINGS_COPY.wizardStepTools}
      </span>
      <div className="flex items-center gap-2">
        {setup.step === "checks" ? (
          <Button
            type="button"
            disabled={!setup.canContinue}
            onClick={() => setup.setStep("tools")}
            data-testid="checks-setup-continue"
          >
            {PR_FEEDBACK_SETTINGS_COPY.wizardContinue}
          </Button>
        ) : (
          <Button
            type="button"
            disabled={setup.createHandler.isPending}
            onClick={() => {
              void setup.finish(source).then((handler) => {
                if (handler?.id) {
                  onCreated(handler.id);
                  onClose();
                }
              });
            }}
            data-testid="checks-setup-finish"
          >
            {setup.createHandler.isPending
              ? PR_FEEDBACK_SETTINGS_COPY.wizardFinishing
              : PR_FEEDBACK_SETTINGS_COPY.wizardFinish}
          </Button>
        )}
      </div>
    </footer>
  );
}
