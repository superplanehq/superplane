import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { Plus, Settings, XIcon } from "lucide-react";
import { useCallback, useState } from "react";

import { AddIntakePicker } from "./AddIntakePicker";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { intakeSettingsToApi, type IntakeAutomationRun, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeDrawerPopupsProps, LineIntakeDrawerProps } from "./lineIntakeDrawerTypes";
import {
  intakeAutomationFixture,
  intakeTicketAnalysisFixture,
  LINE_INTAKE_COPY,
  type AddIntakeTemplate,
  type ConfiguredLineIntakeSource,
  type LineIntakeAnalyzingTicket,
} from "./lineIntakeModel";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";
import { useIntakeAutomationRuns } from "./useIntakeAutomationRuns";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";

function useLineIntakeDrawerState({
  initialIntakeId,
  initialSettingsOpen,
  configuredSources,
  onOpenTicket,
  onSelectIntakeTemplate,
}: {
  initialIntakeId?: string;
  initialSettingsOpen: boolean;
  configuredSources: ConfiguredLineIntakeSource[];
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  onSelectIntakeTemplate?: (template: AddIntakeTemplate) => void;
}) {
  const [openTicket, setOpenTicket] = useState<LineIntakeAnalyzingTicket | null>(null);
  const [previewIntakeId, setPreviewIntakeId] = useState<string | null>(null);
  const [settingsIntakeId, setSettingsIntakeId] = useState<string | null>(
    initialSettingsOpen ? (initialIntakeId ?? configuredSources[0]?.intakeId ?? null) : null,
  );
  const [pickerOpen, setPickerOpen] = useState(false);

  function openIntakeRun(run: IntakeAutomationRun) {
    const ticket = { id: run.id, title: run.title, appId: run.appId, runId: run.runId };
    setOpenTicket(ticket);
    onOpenTicket?.(ticket);
  }

  function openIntakeGear(intake: ConfiguredLineIntakeSource) {
    setOpenTicket(null);
    setPreviewIntakeId(null);
    setSettingsIntakeId(intake.intakeId);
  }

  function selectIntakeTemplate(template: AddIntakeTemplate) {
    setPickerOpen(false);
    onSelectIntakeTemplate?.(template);
  }

  return {
    openTicket,
    pickerOpen,
    settingsIntake: configuredSources.find((intake) => intake.intakeId === settingsIntakeId),
    previewIntake: configuredSources.find((intake) => intake.intakeId === previewIntakeId),
    closePreview: () => setPreviewIntakeId(null),
    closeOpenTicket: () => setOpenTicket(null),
    closePicker: () => setPickerOpen(false),
    closeSettings: () => setSettingsIntakeId(null),
    openIntakeRun,
    openPicker: () => setPickerOpen(true),
    openIntakeGear,
    selectIntakeTemplate,
  };
}

export function LineIntakeDrawer({
  onClose,
  initialIntakeId,
  initialSettingsOpen = false,
  initialSettingsTab = "general",
  configuredSources = [],
  onOpenTicket,
  onSelectIntakeTemplate,
  organizationId,
  factoryId,
  factoryKey,
  editAutomationHref,
  editAutomationHrefFor,
  previewSource,
  onSettingsSaved,
  showAddIntakeControl = false,
}: LineIntakeDrawerProps) {
  const drawer = useLineIntakeDrawerState({
    initialIntakeId,
    initialSettingsOpen,
    configuredSources,
    onOpenTicket,
    onSelectIntakeTemplate,
  });

  return (
    <>
      <aside
        className="flex h-full min-h-0 w-[26rem] shrink-0 flex-col border-r border-border bg-slate-200 dark:bg-slate-800"
        data-testid="line-intake-drawer"
        aria-label="Intake"
      >
        <header className="flex shrink-0 items-center justify-between gap-2 px-3 pb-2 pt-3">
          <h2 className="workspace-section-title">Intake</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close Intake"
            title="Close Intake"
            data-testid="line-intake-close"
            className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <XIcon className="size-3.5" aria-hidden />
          </button>
        </header>
        <p className="workspace-body-text shrink-0 px-3 pb-3 text-muted-foreground">
          Automations that listen, evaluate, and create backlog work orders.
        </p>
        <LineIntakeList
          intakes={configuredSources}
          onOpenGear={drawer.openIntakeGear}
          onOpenPicker={drawer.openPicker}
          showAddIntakeControl={showAddIntakeControl}
        />
      </aside>

      <LineIntakeDrawerPopups
        pickerOpen={drawer.pickerOpen}
        initialSettingsTab={initialSettingsTab}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        settingsIntake={drawer.settingsIntake}
        editAutomationHref={
          (drawer.settingsIntake ? editAutomationHrefFor?.(drawer.settingsIntake) : undefined) ?? editAutomationHref
        }
        previewSource={drawer.previewIntake?.source ?? previewSource}
        previewAppId={drawer.previewIntake?.appId}
        openTicket={drawer.openTicket}
        onClosePicker={drawer.closePicker}
        onSelectTemplate={drawer.selectIntakeTemplate}
        onOpenRun={drawer.openIntakeRun}
        onSettingsSaved={onSettingsSaved}
        onCloseSettings={drawer.closeSettings}
        onClosePreview={drawer.closePreview}
        onCloseOpenTicket={drawer.closeOpenTicket}
      />
    </>
  );
}

function LineIntakeList({
  intakes,
  onOpenGear,
  onOpenPicker,
  showAddIntakeControl,
}: {
  intakes: ConfiguredLineIntakeSource[];
  onOpenGear: (intake: ConfiguredLineIntakeSource) => void;
  onOpenPicker: () => void;
  showAddIntakeControl: boolean;
}) {
  return (
    <ul className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-2 [scrollbar-width:thin]">
      {intakes.map((intake) => (
        <li key={intake.intakeId}>
          <IntakeCard intake={intake} onOpenGear={() => onOpenGear(intake)} />
        </li>
      ))}
      {showAddIntakeControl ? (
        <li>
          <button
            type="button"
            onClick={onOpenPicker}
            data-testid="line-intake-add"
            className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border bg-muted/60 px-3 py-3 text-[13px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:border-foreground/20 hover:bg-muted hover:text-foreground"
          >
            <Plus className="size-3.5 shrink-0" aria-hidden />
            Add intake
          </button>
        </li>
      ) : null}
    </ul>
  );
}

function LineIntakeDrawerPopups({
  pickerOpen,
  initialSettingsTab,
  organizationId,
  factoryId,
  factoryKey,
  settingsIntake,
  editAutomationHref,
  previewSource,
  previewAppId,
  openTicket,
  onClosePicker,
  onSelectTemplate,
  onOpenRun,
  onSettingsSaved,
  onCloseSettings,
  onClosePreview,
  onCloseOpenTicket,
}: LineIntakeDrawerPopupsProps) {
  const automation = useIntakeAutomationCanvas(organizationId, settingsIntake?.appId);
  const runs = useIntakeAutomationRuns(organizationId, factoryId, settingsIntake);
  const updateIntake = useUpdateFactoryIntake(organizationId ?? "", factoryId ?? "");

  const saveSettings = useCallback(
    async (next: IntakeSourceSettings) => {
      if (!settingsIntake) {
        throw new Error("The intake automation is not available.");
      }
      await updateIntake.mutateAsync({
        intakeId: settingsIntake.intakeId,
        name: next.name,
        settings: intakeSettingsToApi(next),
      });
      await automation.refetch();
      onSettingsSaved?.();
    },
    [automation, onSettingsSaved, settingsIntake, updateIntake],
  );

  return (
    <>
      <AddIntakePicker open={pickerOpen} onClose={onClosePicker} onSelect={onSelectTemplate} />
      {settingsIntake ? (
        <IntakeSourceSettingsPopup
          key={settingsIntake.intakeId}
          settings={settingsIntake.settings}
          sourceId={settingsIntake.source.id}
          automationGraph={automation.graph}
          automationLoading={automation.isLoading}
          automationError={automation.isError}
          onRetryAutomation={automation.refetch}
          runs={runs.runs}
          runsLoading={runs.isLoading}
          runsError={runs.isError}
          onRetryRuns={runs.retry}
          onSave={saveSettings}
          savePending={updateIntake.isPending || automation.isLoading}
          saveError={
            updateIntake.error
              ? getApiErrorMessage(updateIntake.error, "SuperPlane could not save the intake settings. Try again.")
              : undefined
          }
          onOpenRun={onOpenRun}
          editAutomationHref={editAutomationHref}
          onClose={onCloseSettings}
          initialTab={initialSettingsTab}
          fixed
        />
      ) : null}
      {previewSource ? (
        <WorkOrderSplitRunPopup
          key={previewSource.id}
          organizationId={organizationId}
          factoryKey={factoryKey}
          fixture={intakeAutomationFixture(previewSource, previewAppId)}
          onClose={onClosePreview}
          fixed
        />
      ) : null}
      {openTicket ? (
        <WorkOrderSplitRunPopup
          key={openTicket.id}
          organizationId={organizationId}
          factoryKey={factoryKey}
          fixture={intakeTicketAnalysisFixture(openTicket)}
          onClose={onCloseOpenTicket}
          fixed
        />
      ) : null}
    </>
  );
}

function IntakeCard({ intake, onOpenGear }: { intake: ConfiguredLineIntakeSource; onOpenGear: () => void }) {
  const { source } = intake;
  const displayName = source.name;

  return (
    <article className="w-full rounded-lg bg-card shadow-sm" data-testid={`line-intake-source-${intake.intakeId}`}>
      <div className="flex items-start gap-2 px-3 py-3">
        <img src={source.iconSrc} alt={source.iconAlt} className="mt-0.5 size-5 shrink-0" />
        <div className="min-w-0 flex-1">
          <h3 className="flex flex-wrap items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em] leading-[19.5px] text-foreground">
            {displayName}
            {intake.healthy ? null : (
              <span
                className="text-[11px] font-medium text-amber-700 dark:text-amber-400"
                data-testid={`line-intake-source-${intake.intakeId}-needs-repair`}
              >
                {LINE_INTAKE_COPY.needsRepair}
              </span>
            )}
          </h3>
          <p className="workspace-body-text mt-0.5 text-muted-foreground">
            {intake.healthy ? source.description : LINE_INTAKE_COPY.needsRepairHelper}
          </p>
        </div>
        <button
          type="button"
          onClick={onOpenGear}
          aria-label={`Open ${displayName} settings`}
          title={`Open ${displayName} settings`}
          data-testid={`line-intake-source-${intake.intakeId}-settings`}
          className="-mr-1 -mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Settings className="size-3.5" aria-hidden />
        </button>
      </div>
    </article>
  );
}
