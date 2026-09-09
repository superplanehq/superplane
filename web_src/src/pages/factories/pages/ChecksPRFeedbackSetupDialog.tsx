import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { cn } from "@/lib/utils";
import { sortConnectedIntegrationsByType } from "@/lib/sortConnectedIntegrations";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft, Check, TriangleAlert } from "lucide-react";

import {
  checksHandlerIntegrationRows,
  hasSelectedSuggestedIntegration,
  isChecksHandlerCIIntegration,
  type ChecksHandlerIntegrationRow,
} from "./checksPRFeedbackSetup";
import { StatusCheckPicker } from "./StatusCheckPicker";
import {
  integrationDefinitionLabel,
  useChecksPRFeedbackSetup,
  type ChecksPRFeedbackSetupModel,
} from "./useChecksPRFeedbackSetup";
import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackSource } from "./prFeedbackSettingsModel";

interface ChecksPRFeedbackSetupDialogProps {
  open: boolean;
  organizationId: string;
  factoryId: string;
  repository: string;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
  /** Dialog overlay (default) or inline card for a dedicated setup page. */
  presentation?: "dialog" | "page";
}

export function ChecksPRFeedbackSetupDialog(props: ChecksPRFeedbackSetupDialogProps) {
  const presentation = props.presentation ?? "dialog";
  const setup = useChecksPRFeedbackSetup(
    props.organizationId,
    props.factoryId,
    props.repository,
    props.open || presentation === "page",
  );
  return (
    <>
      <ChecksSetupView
        open={(props.open || presentation === "page") && !setup.connectName}
        presentation={presentation}
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
  presentation,
  setup,
  source,
  onClose,
  onCreated,
}: {
  open: boolean;
  presentation: "dialog" | "page";
  setup: ChecksPRFeedbackSetupModel;
  source: PRFeedbackSource;
  onClose: () => void;
  onCreated: (handlerId: string) => void;
}) {
  const body = (
    <>
      <SetupHeader step={setup.step} onBack={() => setup.setStep("checks")} />
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5">
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
      <div className="shrink-0 border-t border-border">
        {setup.step === "tools" ? <ToolsAccessWarning warning={toolsAccessWarning(setup)} /> : null}
        <SetupFooter setup={setup} source={source} onClose={onClose} onCreated={onCreated} />
      </div>
    </>
  );

  if (presentation === "page") {
    if (!open) {
      return null;
    }
    return (
      <div
        className="grid h-[min(36rem,80vh)] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden rounded-lg border border-border bg-background"
        data-testid="checks-pr-feedback-setup"
      >
        {body}
      </div>
    );
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent
        className="grid h-[min(36rem,80vh)] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0 sm:max-w-xl"
        showCloseButton
        data-testid="checks-pr-feedback-setup"
      >
        {body}
      </DialogContent>
    </Dialog>
  );
}

function SetupHeader({ step, onBack }: { step: "checks" | "tools"; onBack: () => void }) {
  return (
    <header className="shrink-0 border-b border-border px-5 py-4 text-left">
      <div className="flex items-center gap-2.5">
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
        <IntegrationIcon integrationName="github" className="size-5 shrink-0" size={20} />
        <h2 className="min-w-0 text-[15px] font-semibold leading-5">{PR_FEEDBACK_SETTINGS_COPY.wizardTitle}</h2>
      </div>
      {step === "tools" ? (
        <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardToolsDescription}</p>
      ) : (
        <p className="sr-only">{PR_FEEDBACK_SETTINGS_COPY.wizardChecksDescription}</p>
      )}
    </header>
  );
}

function ToolsStep({ setup }: { setup: ChecksPRFeedbackSetupModel }) {
  const ready = sortConnectedIntegrationsByType(
    (setup.connected ?? []).filter(
      (integration) =>
        integration.status?.state === "ready" &&
        integration.metadata?.id &&
        isChecksHandlerCIIntegration(integration.metadata.integrationName),
    ),
  );
  const suggested = setup.available.filter((integration) => setup.suggestedNames.includes(integration.name ?? ""));
  const others = setup.available.filter((integration) => !setup.suggestedNames.includes(integration.name ?? ""));

  return (
    <div className="space-y-5">
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

type ToolsAccessWarningContent = { testId: string; lines: string[] };

function toolsAccessWarning(setup: ChecksPRFeedbackSetupModel): ToolsAccessWarningContent | null {
  if (setup.suggestedNames.length === 0) {
    return { testId: "checks-setup-tools-unknown", lines: [PR_FEEDBACK_SETTINGS_COPY.wizardToolsUnknown] };
  }
  const ready = (setup.connected ?? []).filter(
    (integration) =>
      integration.status?.state === "ready" &&
      integration.metadata?.id &&
      isChecksHandlerCIIntegration(integration.metadata.integrationName),
  );
  if (hasSelectedSuggestedIntegration(setup.suggestedNames, setup.runnerIntegrationIds, ready)) {
    return null;
  }
  return {
    testId: "checks-setup-tools-unselected",
    lines: [PR_FEEDBACK_SETTINGS_COPY.wizardToolsUnselected, PR_FEEDBACK_SETTINGS_COPY.wizardToolsUnselectedDetail],
  };
}

function ToolsAccessWarning({ warning }: { warning: ToolsAccessWarningContent | null }) {
  if (!warning) {
    return null;
  }
  return (
    <div
      role="status"
      className="flex items-start gap-2 bg-amber-50/70 px-5 py-2.5 dark:bg-amber-950/25"
      data-testid={warning.testId}
    >
      <TriangleAlert className="mt-0.5 size-3.5 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
      <div className="workspace-body-text space-y-1 text-muted-foreground">
        {warning.lines.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
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
  const labeled = definitions.map((definition) => ({
    name: definition.name,
    label: getIntegrationTypeDisplayName(definition.label, definition.name ?? ""),
  }));
  const rows = checksHandlerIntegrationRows(labeled, ready);

  return (
    <section>
      <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">{title}</h3>
      <div className="mt-2 rounded-lg border border-border">
        <ul className="divide-y divide-border">
          {rows.map((row) => (
            <IntegrationChoiceRow
              key={row.instanceId ?? row.type}
              row={row}
              selected={Boolean(row.instanceId && selectedIds.includes(row.instanceId))}
              onToggle={() => {
                if (!row.instanceId) {
                  return;
                }
                onToggle(toggleUniqueString(selectedIds, row.instanceId));
              }}
              onConnect={() => onConnect(row.type)}
            />
          ))}
        </ul>
      </div>
    </section>
  );
}

function IntegrationChoiceRow({
  row,
  selected,
  onToggle,
  onConnect,
}: {
  row: ChecksHandlerIntegrationRow;
  selected: boolean;
  onToggle: () => void;
  onConnect: () => void;
}) {
  const selectable = Boolean(row.instanceId);

  return (
    <li>
      <div className={cn("flex w-full items-center gap-2 pr-3", selected ? "bg-accent/50" : "hover:bg-accent/30")}>
        <button
          type="button"
          role={selectable ? "option" : undefined}
          aria-selected={selectable ? selected : undefined}
          disabled={!selectable}
          onClick={onToggle}
          className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left disabled:cursor-default disabled:opacity-100"
          data-testid={row.instanceId ? `checks-setup-integration-${row.instanceId}` : `checks-setup-type-${row.type}`}
        >
          <IntegrationIcon integrationName={row.type} className="size-4 shrink-0" size={16} />
          <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{row.displayName}</span>
        </button>
        {selectable ? null : (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onConnect}
            data-testid={`checks-setup-connect-${row.type}`}
          >
            {PR_FEEDBACK_SETTINGS_COPY.wizardConnect}
          </Button>
        )}
        <span className="flex size-3.5 shrink-0 items-center justify-center">
          {selected ? <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
        </span>
      </div>
    </li>
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
    <footer className="flex items-center justify-between gap-3 px-5 py-3">
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
