import { useUpdateCanvas } from "@/hooks/useCanvasData";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { useMutation } from "@tanstack/react-query";
import { ChevronRight, Plus, Settings, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import { AddIntakePicker } from "./AddIntakePicker";
import { AnalyzingIntakeTicketList } from "./AnalyzingIntakeTicketList";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import {
  type IntakeAutomationRun,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { intakeSettingsFromCanvas } from "./intakeAutomationSettings";
import {
  GITHUB_ISSUES_ANALYZING_TICKETS,
  intakeAutomationFixture,
  intakeTicketAnalysisFixture,
  isLineIntakeSourceId,
  LINE_INTAKE_COPY,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
  type AddIntakeTemplate,
  type ConfiguredLineIntakeSource,
  type LineIntakeAnalyzingTicket,
  type LineIntakeSource,
  type LineIntakeSourceId,
} from "./lineIntakeModel";
import type { LineIntakeDrawerProps } from "./lineIntakeDrawerTypes";
import { useIntakeAutomationCanvas } from "./useIntakeAutomationCanvas";
import { useIntakeAutomationRuns } from "./useIntakeAutomationRuns";
import { useLiveIntakeTickets } from "./useLiveIntakeTickets";
import { saveIntakeAutomationSettings } from "./saveIntakeAutomationSettings";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";

function useLineIntakeDrawerState({
  initialSourceId,
  initialSettingsOpen,
  configuredSources,
  onOpenTicket,
  onSelectIntakeTemplate,
}: {
  initialSourceId?: LineIntakeSourceId;
  initialSettingsOpen: boolean;
  configuredSources: ConfiguredLineIntakeSource[];
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  onSelectIntakeTemplate?: (template: AddIntakeTemplate) => void;
}) {
  const [expandedSourceIds, setExpandedSourceIds] = useState<ReadonlySet<LineIntakeSourceId>>(
    () => new Set(initialSourceId ? [initialSourceId] : []),
  );
  const [openTicket, setOpenTicket] = useState<LineIntakeAnalyzingTicket | null>(null);
  const [openSourceId, setOpenSourceId] = useState<string | null>(null);
  const [settingsSourceId, setSettingsSourceId] = useState<LineIntakeSourceId | null>(
    initialSettingsOpen ? (initialSourceId ?? "github-issues") : null,
  );
  const [pickerOpen, setPickerOpen] = useState(false);

  function toggleSource(sourceId: LineIntakeSourceId) {
    setExpandedSourceIds((current) => {
      const next = new Set(current);
      if (next.has(sourceId)) {
        next.delete(sourceId);
      } else {
        next.add(sourceId);
      }
      return next;
    });
  }

  function expandSource(sourceId: LineIntakeSourceId) {
    setExpandedSourceIds((current) => new Set(current).add(sourceId));
  }

  function openIntakeRun(run: IntakeAutomationRun) {
    const ticket = { id: run.id, title: run.title, appId: run.appId, runId: run.runId };
    setOpenTicket(ticket);
    onOpenTicket?.(ticket);
  }

  function openSourceGear(source: LineIntakeSource) {
    setOpenTicket(null);
    if (configuredSources.some((configured) => configured.source.id === source.id)) {
      setOpenSourceId(null);
      setSettingsSourceId(source.id);
      return;
    }
    setSettingsSourceId(null);
    setOpenSourceId(source.id);
  }

  function openAnalyzingTicket(ticket: LineIntakeAnalyzingTicket) {
    setOpenSourceId(null);
    setSettingsSourceId(null);
    if (ticket.appId && ticket.runId && onOpenTicket) {
      onOpenTicket(ticket);
      return;
    }
    setOpenTicket(ticket);
    onOpenTicket?.(ticket);
  }

  function selectIntakeTemplate(template: AddIntakeTemplate) {
    setPickerOpen(false);
    if (onSelectIntakeTemplate) {
      onSelectIntakeTemplate(template);
      return;
    }
    if (isLineIntakeSourceId(template.id)) {
      expandSource(template.id);
    }
  }

  const settingsSource = settingsSourceId ? lineIntakeSourceById(settingsSourceId) : undefined;

  return {
    expandedSourceIds,
    openSource: openSourceId ? lineIntakeSourceById(openSourceId) : undefined,
    openTicket,
    pickerOpen,
    settingsSource,
    closeOpenSource: () => setOpenSourceId(null),
    closeOpenTicket: () => setOpenTicket(null),
    closePicker: () => setPickerOpen(false),
    closeSettings: () => setSettingsSourceId(null),
    openAnalyzingTicket,
    openIntakeRun,
    openPicker: () => setPickerOpen(true),
    openSourceGear,
    selectIntakeTemplate,
    toggleSource,
  };
}

export function LineIntakeDrawer({
  onClose,
  initialSourceId,
  initialSettingsOpen = false,
  initialSettingsTab = "general",
  sources = LINE_INTAKE_SOURCES,
  configuredSources = [],
  analyzingTickets = GITHUB_ISSUES_ANALYZING_TICKETS,
  onOpenTicket,
  onSelectIntakeTemplate,
  organizationId,
  editAutomationHref,
  editAutomationHrefFor,
  onSettingsSaved,
}: LineIntakeDrawerProps) {
  const drawer = useLineIntakeDrawerState({
    initialSourceId,
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
        <LineIntakeSourceList
          sources={sources}
          configuredSources={configuredSources}
          analyzingTickets={analyzingTickets}
          expandedSourceIds={drawer.expandedSourceIds}
          openTicketId={drawer.openTicket?.id ?? null}
          onToggleSource={drawer.toggleSource}
          onOpenGear={drawer.openSourceGear}
          onOpenAnalyzingTicket={drawer.openAnalyzingTicket}
          onOpenPicker={drawer.openPicker}
        />
      </aside>

      <LineIntakeDrawerPopups
        pickerOpen={drawer.pickerOpen}
        settingsSource={drawer.settingsSource}
        initialSettingsTab={initialSettingsTab}
        organizationId={organizationId}
        configuredSource={configuredSources.find((configured) => configured.source.id === drawer.settingsSource?.id)}
        editAutomationHref={
          (drawer.settingsSource ? editAutomationHrefFor?.(drawer.settingsSource) : undefined) ?? editAutomationHref
        }
        openSource={drawer.openSource}
        openTicket={drawer.openTicket}
        onClosePicker={drawer.closePicker}
        onSelectTemplate={drawer.selectIntakeTemplate}
        onOpenRun={drawer.openIntakeRun}
        onSettingsSaved={onSettingsSaved}
        onCloseSettings={drawer.closeSettings}
        onCloseOpenSource={drawer.closeOpenSource}
        onCloseOpenTicket={drawer.closeOpenTicket}
      />
    </>
  );
}

function LineIntakeSourceList({
  sources,
  configuredSources,
  analyzingTickets,
  expandedSourceIds,
  openTicketId,
  onToggleSource,
  onOpenGear,
  onOpenAnalyzingTicket,
  onOpenPicker,
}: {
  sources: LineIntakeSource[];
  configuredSources: ConfiguredLineIntakeSource[];
  analyzingTickets: LineIntakeAnalyzingTicket[];
  expandedSourceIds: ReadonlySet<LineIntakeSourceId>;
  openTicketId: string | null;
  onToggleSource: (sourceId: LineIntakeSourceId) => void;
  onOpenGear: (source: LineIntakeSource) => void;
  onOpenAnalyzingTicket: (ticket: LineIntakeAnalyzingTicket) => void;
  onOpenPicker: () => void;
}) {
  return (
    <ul className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-2 [scrollbar-width:thin]">
      {sources.map((source) => {
        const expanded = expandedSourceIds.has(source.id);
        const configuredSource = configuredSources.find((configured) => configured.source.id === source.id);
        return (
          <IntakeSourceListItem
            key={source.id}
            source={source}
            configuredSource={configuredSource}
            fallbackTickets={source.id === "github-issues" ? analyzingTickets : []}
            expanded={expanded}
            displayName={source.name}
            gearKind={configuredSource ? "settings" : "automation"}
            openTicketId={openTicketId}
            onToggleSource={onToggleSource}
            onOpenGear={onOpenGear}
            onOpenAnalyzingTicket={onOpenAnalyzingTicket}
          />
        );
      })}
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

function IntakeSourceListItem({
  source,
  configuredSource,
  fallbackTickets,
  expanded,
  displayName,
  gearKind,
  openTicketId,
  onToggleSource,
  onOpenGear,
  onOpenAnalyzingTicket,
}: {
  source: LineIntakeSource;
  configuredSource?: ConfiguredLineIntakeSource;
  fallbackTickets: LineIntakeAnalyzingTicket[];
  expanded: boolean;
  displayName: string;
  gearKind: "settings" | "automation";
  openTicketId: string | null;
  onToggleSource: (sourceId: LineIntakeSourceId) => void;
  onOpenGear: (source: LineIntakeSource) => void;
  onOpenAnalyzingTicket: (ticket: LineIntakeAnalyzingTicket) => void;
}) {
  const liveTickets = useLiveIntakeTickets(configuredSource, expanded);
  const tickets = configuredSource ? liveTickets.tickets : fallbackTickets;

  return (
    <li>
      <IntakeSourceCard
        source={source}
        displayName={displayName}
        gearKind={gearKind}
        expanded={expanded}
        childCount={tickets.length}
        onToggle={() => onToggleSource(source.id)}
        onOpenGear={() => onOpenGear(source)}
      >
        {expanded ? (
          <AnalyzingIntakeTicketList
            tickets={tickets}
            openTicketId={openTicketId}
            onOpenTicket={onOpenAnalyzingTicket}
            loading={Boolean(configuredSource && liveTickets.isLoading)}
            error={Boolean(configuredSource && liveTickets.isError)}
            onRetry={liveTickets.retry}
          />
        ) : null}
      </IntakeSourceCard>
    </li>
  );
}

function LineIntakeDrawerPopups({
  pickerOpen,
  settingsSource,
  initialSettingsTab,
  organizationId,
  configuredSource,
  editAutomationHref,
  openSource,
  openTicket,
  onClosePicker,
  onSelectTemplate,
  onOpenRun,
  onSettingsSaved,
  onCloseSettings,
  onCloseOpenSource,
  onCloseOpenTicket,
}: {
  pickerOpen: boolean;
  settingsSource?: LineIntakeSource;
  initialSettingsTab: IntakeSettingsTab;
  organizationId?: string;
  configuredSource?: ConfiguredLineIntakeSource;
  editAutomationHref?: string;
  openSource?: LineIntakeSource;
  openTicket: LineIntakeAnalyzingTicket | null;
  onClosePicker: () => void;
  onSelectTemplate: (template: AddIntakeTemplate) => void;
  onOpenRun: (run: IntakeAutomationRun) => void;
  onSettingsSaved?: () => void;
  onCloseSettings: () => void;
  onCloseOpenSource: () => void;
  onCloseOpenTicket: () => void;
}) {
  const appId = configuredSource?.appId;
  const automation = useIntakeAutomationCanvas(organizationId, appId, settingsSource?.name ?? "");
  const runs = useIntakeAutomationRuns(configuredSource);
  const updateCanvas = useUpdateCanvas(organizationId ?? "", appId ?? "");
  const context = configuredSource
    ? {
        sourceId: configuredSource.source.id,
        triggerNodeId: configuredSource.triggerNodeId,
        analysisNodeId: configuredSource.analysisNodeId,
        createWorkOrderNodeId: configuredSource.createWorkOrderNodeId,
      }
    : undefined;
  const settings = context ? intakeSettingsFromCanvas(context, automation.sourceCanvas) : undefined;
  const saveSettings = useMutation({
    mutationFn: async (next: IntakeSourceSettings) => {
      if (!appId || !context || !automation.sourceCanvas) {
        throw new Error("The intake automation is not available.");
      }
      await saveIntakeAutomationSettings({
        canvasId: appId,
        context,
        canvas: automation.sourceCanvas,
        settings: next,
        updateCanvas: updateCanvas.mutateAsync,
      });
      await automation.refetch();
      onSettingsSaved?.();
    },
  });

  return (
    <>
      <AddIntakePicker open={pickerOpen} onClose={onClosePicker} onSelect={onSelectTemplate} />
      {settingsSource && settings ? (
        <IntakeSourceSettingsPopup
          key={settingsSource.id}
          settings={settings}
          sourceId={settingsSource.id}
          automationCanvas={automation.canvas}
          automationLoading={automation.isLoading}
          automationError={automation.isError}
          onRetryAutomation={automation.refetch}
          runs={runs.runs}
          runsLoading={runs.isLoading}
          runsError={runs.isError}
          onRetryRuns={runs.retry}
          onSave={saveSettings.mutateAsync}
          savePending={saveSettings.isPending || automation.isLoading}
          saveError={
            saveSettings.error
              ? getApiErrorMessage(saveSettings.error, "SuperPlane could not save the intake settings. Try again.")
              : undefined
          }
          onOpenRun={onOpenRun}
          editAutomationHref={editAutomationHref}
          onClose={onCloseSettings}
          initialTab={initialSettingsTab}
          fixed
        />
      ) : null}
      {openSource ? (
        <WorkOrderSplitRunPopup
          key={openSource.id}
          fixture={intakeAutomationFixture(openSource)}
          onClose={onCloseOpenSource}
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

function IntakeSourceCard({
  source,
  displayName,
  gearKind,
  expanded,
  childCount,
  onToggle,
  onOpenGear,
  children,
}: {
  source: LineIntakeSource;
  displayName: string;
  gearKind: "settings" | "automation";
  expanded: boolean;
  childCount: number;
  onToggle: () => void;
  onOpenGear: () => void;
  children?: ReactNode;
}) {
  return (
    <article className="relative w-full rounded-lg bg-card shadow-sm" data-testid={`line-intake-source-${source.id}`}>
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
          </h3>
          <p className="workspace-body-text mt-0.5 text-muted-foreground">{source.description}</p>
        </div>
        <button
          type="button"
          onClick={(event) => {
            event.stopPropagation();
            onOpenGear();
          }}
          aria-label={`Open ${displayName} ${gearKind}`}
          title={`Open ${displayName} ${gearKind}`}
          data-testid={`line-intake-source-${source.id}-${gearKind}`}
          className="relative z-20 -mr-1 -mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Settings className="size-3.5" aria-hidden />
        </button>
      </div>
      {children}
    </article>
  );
}
