import { useUpdateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { ChevronRight, Plus, Settings, XIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

import { AddIntakePicker } from "./AddIntakePicker";
import { AnalyzingIntakeTicketList } from "./AnalyzingIntakeTicketList";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import {
  intakeSettingsToApi,
  type IntakeAutomationRun,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import {
  intakeAutomationFixture,
  intakeTicketAnalysisFixture,
  LINE_INTAKE_COPY,
  type AddIntakeTemplate,
  type ConfiguredLineIntakeSource,
  type LineIntakeAnalyzingTicket,
  type LineIntakeSource,
} from "./lineIntakeModel";
import type { LineIntakeDrawerProps } from "./lineIntakeDrawerTypes";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";
import { useIntakeAutomationRuns } from "./useIntakeAutomationRuns";
import { useLiveIntakeTickets } from "./useLiveIntakeTickets";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";

/**
 * Opens the sole intake of a workspace, one time, when the URL names no
 * intake. Setup opens the drawer this way, and a single collapsed row hides
 * the work the intake is doing. The intakes load after the drawer, so this
 * waits for them. It runs one time only, so a collapse holds.
 */
function useExpandLoneIntake(
  configuredSources: ConfiguredLineIntakeSource[],
  initialIntakeId: string | undefined,
  setExpandedIntakeIds: (expanded: ReadonlySet<string>) => void,
) {
  const expanded = useRef(false);

  useEffect(() => {
    if (expanded.current || initialIntakeId || configuredSources.length !== 1) {
      return;
    }
    expanded.current = true;
    setExpandedIntakeIds(new Set([configuredSources[0].intakeId]));
  }, [configuredSources, initialIntakeId, setExpandedIntakeIds]);
}

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
  const [expandedIntakeIds, setExpandedIntakeIds] = useState<ReadonlySet<string>>(
    () => new Set(initialIntakeId ? [initialIntakeId] : []),
  );
  useExpandLoneIntake(configuredSources, initialIntakeId, setExpandedIntakeIds);
  const [openTicket, setOpenTicket] = useState<LineIntakeAnalyzingTicket | null>(null);
  const [previewIntakeId, setPreviewIntakeId] = useState<string | null>(null);
  const [settingsIntakeId, setSettingsIntakeId] = useState<string | null>(
    initialSettingsOpen ? (initialIntakeId ?? configuredSources[0]?.intakeId ?? null) : null,
  );
  const [pickerOpen, setPickerOpen] = useState(false);

  function toggleIntake(intakeId: string) {
    setExpandedIntakeIds((current) => {
      const next = new Set(current);
      if (next.has(intakeId)) {
        next.delete(intakeId);
      } else {
        next.add(intakeId);
      }
      return next;
    });
  }

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

  function openAnalyzingTicket(ticket: LineIntakeAnalyzingTicket) {
    setPreviewIntakeId(null);
    setSettingsIntakeId(null);
    if (ticket.appId && ticket.runId && onOpenTicket) {
      onOpenTicket(ticket);
      return;
    }
    setOpenTicket(ticket);
    onOpenTicket?.(ticket);
  }

  function selectIntakeTemplate(template: AddIntakeTemplate) {
    setPickerOpen(false);
    onSelectIntakeTemplate?.(template);
  }

  return {
    expandedIntakeIds,
    openTicket,
    pickerOpen,
    settingsIntake: configuredSources.find((intake) => intake.intakeId === settingsIntakeId),
    previewIntake: configuredSources.find((intake) => intake.intakeId === previewIntakeId),
    closePreview: () => setPreviewIntakeId(null),
    closeOpenTicket: () => setOpenTicket(null),
    closePicker: () => setPickerOpen(false),
    closeSettings: () => setSettingsIntakeId(null),
    openAnalyzingTicket,
    openIntakeRun,
    openPicker: () => setPickerOpen(true),
    openIntakeGear,
    selectIntakeTemplate,
    toggleIntake,
  };
}

export function LineIntakeDrawer({
  onClose,
  initialIntakeId,
  initialSettingsOpen = false,
  initialSettingsTab = "general",
  configuredSources = [],
  analyzingTickets = [],
  onOpenTicket,
  onSelectIntakeTemplate,
  organizationId,
  factoryId,
  editAutomationHref,
  editAutomationHrefFor,
  previewSource,
  onSettingsSaved,
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
          organizationId={organizationId}
          factoryId={factoryId}
          intakes={configuredSources}
          fallbackTickets={analyzingTickets}
          expandedIntakeIds={drawer.expandedIntakeIds}
          openTicketId={drawer.openTicket?.id ?? null}
          onToggleIntake={drawer.toggleIntake}
          onOpenGear={drawer.openIntakeGear}
          onOpenAnalyzingTicket={drawer.openAnalyzingTicket}
          onOpenPicker={drawer.openPicker}
        />
      </aside>

      <LineIntakeDrawerPopups
        pickerOpen={drawer.pickerOpen}
        initialSettingsTab={initialSettingsTab}
        organizationId={organizationId}
        factoryId={factoryId}
        settingsIntake={drawer.settingsIntake}
        editAutomationHref={
          (drawer.settingsIntake ? editAutomationHrefFor?.(drawer.settingsIntake) : undefined) ?? editAutomationHref
        }
        previewSource={drawer.previewIntake?.source ?? previewSource}
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
  organizationId,
  factoryId,
  intakes,
  fallbackTickets,
  expandedIntakeIds,
  openTicketId,
  onToggleIntake,
  onOpenGear,
  onOpenAnalyzingTicket,
  onOpenPicker,
}: {
  organizationId?: string;
  factoryId?: string;
  intakes: ConfiguredLineIntakeSource[];
  fallbackTickets: LineIntakeAnalyzingTicket[];
  expandedIntakeIds: ReadonlySet<string>;
  openTicketId: string | null;
  onToggleIntake: (intakeId: string) => void;
  onOpenGear: (intake: ConfiguredLineIntakeSource) => void;
  onOpenAnalyzingTicket: (ticket: LineIntakeAnalyzingTicket) => void;
  onOpenPicker: () => void;
}) {
  return (
    <ul className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-2 [scrollbar-width:thin]">
      {intakes.map((intake) => (
        <IntakeListItem
          key={intake.intakeId}
          organizationId={organizationId}
          factoryId={factoryId}
          intake={intake}
          fallbackTickets={fallbackTickets}
          expanded={expandedIntakeIds.has(intake.intakeId)}
          openTicketId={openTicketId}
          onToggleIntake={onToggleIntake}
          onOpenGear={onOpenGear}
          onOpenAnalyzingTicket={onOpenAnalyzingTicket}
        />
      ))}
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
    </ul>
  );
}

function IntakeListItem({
  organizationId,
  factoryId,
  intake,
  fallbackTickets,
  expanded,
  openTicketId,
  onToggleIntake,
  onOpenGear,
  onOpenAnalyzingTicket,
}: {
  organizationId?: string;
  factoryId?: string;
  intake: ConfiguredLineIntakeSource;
  fallbackTickets: LineIntakeAnalyzingTicket[];
  expanded: boolean;
  openTicketId: string | null;
  onToggleIntake: (intakeId: string) => void;
  onOpenGear: (intake: ConfiguredLineIntakeSource) => void;
  onOpenAnalyzingTicket: (ticket: LineIntakeAnalyzingTicket) => void;
}) {
  const live = useLiveIntakeTickets(organizationId, factoryId, intake, expanded);
  const connected = Boolean(organizationId && factoryId);
  const tickets = connected ? live.tickets : fallbackTickets;

  return (
    <li>
      <IntakeCard
        intake={intake}
        expanded={expanded}
        childCount={tickets.length}
        onToggle={() => onToggleIntake(intake.intakeId)}
        onOpenGear={() => onOpenGear(intake)}
      >
        {expanded ? (
          <AnalyzingIntakeTicketList
            tickets={tickets}
            openTicketId={openTicketId}
            onOpenTicket={onOpenAnalyzingTicket}
            loading={connected && live.isLoading}
            error={connected && live.isError}
            onRetry={live.retry}
          />
        ) : null}
      </IntakeCard>
    </li>
  );
}

function LineIntakeDrawerPopups({
  pickerOpen,
  initialSettingsTab,
  organizationId,
  factoryId,
  settingsIntake,
  editAutomationHref,
  previewSource,
  openTicket,
  onClosePicker,
  onSelectTemplate,
  onOpenRun,
  onSettingsSaved,
  onCloseSettings,
  onClosePreview,
  onCloseOpenTicket,
}: {
  pickerOpen: boolean;
  initialSettingsTab: IntakeSettingsTab;
  organizationId?: string;
  factoryId?: string;
  settingsIntake?: ConfiguredLineIntakeSource;
  editAutomationHref?: string;
  previewSource?: LineIntakeSource;
  openTicket: LineIntakeAnalyzingTicket | null;
  onClosePicker: () => void;
  onSelectTemplate: (template: AddIntakeTemplate) => void;
  onOpenRun: (run: IntakeAutomationRun) => void;
  onSettingsSaved?: () => void;
  onCloseSettings: () => void;
  onClosePreview: () => void;
  onCloseOpenTicket: () => void;
}) {
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
          fixture={intakeAutomationFixture(previewSource)}
          onClose={onClosePreview}
          fixed
        />
      ) : null}
      {openTicket ? (
        <WorkOrderSplitRunPopup
          key={openTicket.id}
          fixture={intakeTicketAnalysisFixture(openTicket)}
          onClose={onCloseOpenTicket}
          fixed
        />
      ) : null}
    </>
  );
}

function IntakeCard({
  intake,
  expanded,
  childCount,
  onToggle,
  onOpenGear,
  children,
}: {
  intake: ConfiguredLineIntakeSource;
  expanded: boolean;
  childCount: number;
  onToggle: () => void;
  onOpenGear: () => void;
  children?: ReactNode;
}) {
  const { source } = intake;
  const displayName = source.name;

  return (
    <article
      className="relative w-full rounded-lg bg-card shadow-sm"
      data-testid={`line-intake-source-${intake.intakeId}`}
    >
      <div className="relative flex items-start gap-2 px-3 py-3">
        <ChevronRight
          className={cn("mt-1 size-3.5 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-90")}
          aria-hidden
        />
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-lg"
          aria-label={expanded ? `Collapse ${displayName}` : `Expand ${displayName}`}
          aria-expanded={expanded}
          onClick={onToggle}
        />
        <img
          src={source.iconSrc}
          alt={source.iconAlt}
          className="relative z-10 mt-0.5 size-5 shrink-0 pointer-events-none"
        />
        <div className="relative z-10 min-w-0 flex-1 pointer-events-none">
          <h3 className="flex flex-wrap items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em] leading-[19.5px] text-foreground">
            {displayName}
            {expanded && childCount > 0 ? (
              <span className="text-[11px] font-medium tabular-nums text-muted-foreground">{childCount}</span>
            ) : null}
            {expanded && childCount > 0 ? (
              <span className="text-[11px] font-medium text-muted-foreground">{LINE_INTAKE_COPY.analyzingStatus}</span>
            ) : null}
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
          onClick={(event) => {
            event.stopPropagation();
            onOpenGear();
          }}
          aria-label={`Open ${displayName} settings`}
          title={`Open ${displayName} settings`}
          data-testid={`line-intake-source-${intake.intakeId}-settings`}
          className="relative z-20 -mr-1 -mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Settings className="size-3.5" aria-hidden />
        </button>
      </div>
      {children}
    </article>
  );
}
