import { cn } from "@/lib/utils";
import { ChevronRight, Loader2, Plus, Settings, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import { AddIntakePicker } from "./AddIntakePicker";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  type IntakeAutomationRun,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import {
  GITHUB_ISSUES_ANALYZING_TICKETS,
  intakeAutomationCanvas,
  intakeAutomationFixture,
  intakeTicketAnalysisFixture,
  isLineIntakeSourceId,
  LINE_INTAKE_COPY,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
  type LineIntakeAnalyzingTicket,
  type LineIntakeSource,
  type LineIntakeSourceId,
} from "./lineIntakeModel";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";

interface LineIntakeDrawerProps {
  onClose: () => void;
  initialSourceId?: LineIntakeSourceId;
  initialSettingsOpen?: boolean;
  initialSettingsTab?: IntakeSettingsTab;
  sources?: LineIntakeSource[];
  analyzingTickets?: LineIntakeAnalyzingTicket[];
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  editAutomationHref?: string;
}

/**
 * Pane beside the line board. Each intake source is a white card that
 * expands and collapses on its own.
 */
export function LineIntakeDrawer({
  onClose,
  initialSourceId,
  initialSettingsOpen = false,
  initialSettingsTab = "general",
  sources = LINE_INTAKE_SOURCES,
  analyzingTickets = GITHUB_ISSUES_ANALYZING_TICKETS,
  onOpenTicket,
  editAutomationHref,
}: LineIntakeDrawerProps) {
  const [expandedSourceIds, setExpandedSourceIds] = useState<ReadonlySet<LineIntakeSourceId>>(
    () => new Set(initialSourceId ? [initialSourceId] : []),
  );
  const [openTicket, setOpenTicket] = useState<LineIntakeAnalyzingTicket | null>(null);
  const [openSourceId, setOpenSourceId] = useState<string | null>(null);
  const [githubSettings, setGithubSettings] = useState(DEFAULT_GITHUB_INTAKE_SETTINGS);
  const [settingsOpen, setSettingsOpen] = useState(initialSettingsOpen);
  const [pickerOpen, setPickerOpen] = useState(false);
  const openSource = openSourceId ? lineIntakeSourceById(openSourceId) : undefined;

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
    const ticket = { id: run.id, title: run.title };
    setOpenTicket(ticket);
    onOpenTicket?.(ticket);
  }

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
        <ul className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-2 [scrollbar-width:thin]">
          {sources.map((source) => {
            const expanded = expandedSourceIds.has(source.id);
            const tickets = source.id === "github-issues" ? analyzingTickets : [];
            return (
              <li key={source.id}>
                <IntakeSourceCard
                  source={source}
                  displayName={source.id === "github-issues" ? githubSettings.name : source.name}
                  expanded={expanded}
                  childCount={tickets.length}
                  onToggle={() => toggleSource(source.id)}
                  onOpenGear={() => {
                    setOpenTicket(null);
                    if (source.id === "github-issues") {
                      setOpenSourceId(null);
                      setSettingsOpen(true);
                      return;
                    }
                    setSettingsOpen(false);
                    setOpenSourceId(source.id);
                  }}
                >
                  {expanded ? (
                    <AnalyzingTicketList
                      tickets={tickets}
                      openTicketId={openTicket?.id ?? null}
                      onOpenTicket={(ticket) => {
                        setOpenSourceId(null);
                        setSettingsOpen(false);
                        setOpenTicket(ticket);
                        onOpenTicket?.(ticket);
                      }}
                    />
                  ) : null}
                </IntakeSourceCard>
              </li>
            );
          })}
          <li>
            <button
              type="button"
              onClick={() => setPickerOpen(true)}
              data-testid="line-intake-add"
              className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border bg-muted/60 px-3 py-3 text-[13px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:border-foreground/20 hover:bg-muted hover:text-foreground"
            >
              <Plus className="size-3.5 shrink-0" aria-hidden />
              Add intake
            </button>
          </li>
        </ul>
      </aside>

      <AddIntakePicker
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onSelect={(template) => {
          setPickerOpen(false);
          if (isLineIntakeSourceId(template.id)) {
            expandSource(template.id);
          }
        }}
      />

      {settingsOpen ? (
        <IntakeSourceSettingsPopup
          settings={githubSettings}
          automationCanvas={intakeAutomationCanvas(lineIntakeSourceById("github-issues")!)}
          onSave={setGithubSettings}
          onOpenRun={openIntakeRun}
          editAutomationHref={editAutomationHref}
          onClose={() => setSettingsOpen(false)}
          initialTab={initialSettingsTab}
          fixed
        />
      ) : null}

      {openSource ? (
        <WorkOrderSplitRunPopup
          key={openSource.id}
          fixture={intakeAutomationFixture(openSource)}
          onClose={() => setOpenSourceId(null)}
          fixed
        />
      ) : null}

      {openTicket ? (
        <WorkOrderSplitRunPopup
          key={openTicket.id}
          fixture={intakeTicketAnalysisFixture(openTicket)}
          onClose={() => setOpenTicket(null)}
          fixed
        />
      ) : null}
    </>
  );
}

function IntakeSourceCard({
  source,
  displayName,
  expanded,
  childCount,
  onToggle,
  onOpenGear,
  children,
}: {
  source: LineIntakeSource;
  displayName: string;
  expanded: boolean;
  childCount: number;
  onToggle: () => void;
  onOpenGear: () => void;
  children?: ReactNode;
}) {
  const gearKind = source.id === "github-issues" ? "settings" : "automation";
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

function AnalyzingTicketList({
  tickets,
  openTicketId,
  onOpenTicket,
}: {
  tickets: LineIntakeAnalyzingTicket[];
  openTicketId: string | null;
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
}) {
  if (tickets.length === 0) {
    return (
      <p className="workspace-body-text px-3 pb-3 pl-9 text-muted-foreground" data-testid="line-intake-empty">
        {LINE_INTAKE_COPY.analyzingEmpty}
      </p>
    );
  }

  return (
    <ul
      className="divide-y divide-border/70 border-t border-border/70"
      data-testid="line-intake-analyzing"
      aria-label={LINE_INTAKE_COPY.analyzingTitle}
    >
      {tickets.map((ticket) => {
        const selected = openTicketId === ticket.id;
        return (
          <li key={ticket.id} data-testid={`line-intake-analyzing-ticket-${ticket.id}`}>
            <button
              type="button"
              onClick={() => onOpenTicket?.(ticket)}
              aria-label={`Open ${ticket.title}`}
              aria-pressed={selected}
              className={cn(
                "flex w-full items-start gap-2.5 px-3 py-3.5 text-left transition-colors hover:bg-accent/70",
                selected && "bg-accent",
              )}
            >
              <Loader2
                className="mt-0.5 size-3.5 shrink-0 animate-spin text-muted-foreground"
                aria-hidden
                data-testid="line-intake-analyzing-spinner"
              />
              <span className="min-w-0 flex-1 text-left text-[13px] font-medium leading-snug tracking-[-0.01em] text-foreground">
                {ticket.title}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
