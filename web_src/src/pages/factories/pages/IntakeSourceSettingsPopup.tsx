import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { History, Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";

import {
  INTAKE_SETTINGS_COPY,
  intakePlacementActivity,
  intakePlacementLabel,
  intakeRelativeTime,
  normalizeIntakeSourceSettings,
  type IntakeAutomationRun,
  type IntakeListenMode,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
  type IntakeTicketPlacement,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { SettingsAutomationCanvas } from "./SettingsAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { LineIntakeSourceId } from "./lineIntakeModel";

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  sourceId?: LineIntakeSourceId;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  runs?: IntakeAutomationRun[];
  runsLoading?: boolean;
  runsError?: boolean;
  onRetryRuns?: () => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  onOpenRun?: (run: IntakeAutomationRun) => void;
  editAutomationHref?: string;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: IntakeSettingsTab;
}

export function IntakeSourceSettingsPopup({
  settings,
  sourceId = "github-issues",
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  runs = [],
  runsLoading = false,
  runsError = false,
  onRetryRuns,
  onSave,
  savePending = false,
  saveError,
  onOpenRun,
  editAutomationHref,
  onClose,
  fixed = true,
  initialTab = "general",
}: IntakeSourceSettingsPopupProps) {
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<IntakeSettingsTab>(initialTab);

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <PopupShell testId="intake-source-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={`Intake ${settings.name}`} onClose={onClose}>
        <Tabs value={tab} onValueChange={(value) => setTab(value as IntakeSettingsTab)} className="mt-3">
          <TabsList aria-label={INTAKE_SETTINGS_COPY.tabsLabel}>
            <TabsTrigger value="general" data-testid="intake-settings-tab-general">
              <Settings />
              {INTAKE_SETTINGS_COPY.generalTab}
            </TabsTrigger>
            <TabsTrigger value="runs" data-testid="intake-settings-tab-runs">
              <History />
              {INTAKE_SETTINGS_COPY.runsTab}
            </TabsTrigger>
            <TabsTrigger value="automation" data-testid="intake-settings-tab-automation">
              <Workflow />
              {INTAKE_SETTINGS_COPY.automationTab}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </PopupHeader>
      {tab === "automation" ? (
        <IntakeAutomationTab
          graph={automationGraph}
          title={settings.name}
          editHref={editAutomationHref}
          loading={automationLoading}
          error={automationError}
          onRetry={onRetryAutomation}
        />
      ) : tab === "runs" ? (
        <IntakeRunsList
          runs={runs}
          loading={runsLoading}
          error={runsError}
          onRetry={onRetryRuns}
          onOpenRun={onOpenRun}
        />
      ) : (
        <>
          <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
            <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
              <section>
                <Label htmlFor="intake-source-name">{INTAKE_SETTINGS_COPY.nameLabel}</Label>
                <p className="workspace-body-text mt-1 text-muted-foreground">{INTAKE_SETTINGS_COPY.nameHelper}</p>
                <Input
                  id="intake-source-name"
                  className="mt-2"
                  value={draft.name}
                  onChange={(event) => update("name", event.target.value)}
                  data-testid="intake-source-name"
                />
              </section>

              <fieldset className="min-w-0">
                <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
                  {INTAKE_SETTINGS_COPY.listenLabel}
                </legend>
                <div className="mt-2 flex flex-col gap-2">
                  <IntakeSettingsRadioOption
                    name="intake-listen-mode"
                    value="listen"
                    checked={draft.listenMode === "listen"}
                    title={INTAKE_SETTINGS_COPY.listenOption}
                    helper={INTAKE_SETTINGS_COPY.listenHelper}
                    onChange={() => update("listenMode", "listen" satisfies IntakeListenMode)}
                  />
                  <IntakeSettingsRadioOption
                    name="intake-listen-mode"
                    value="schedule"
                    checked={draft.listenMode === "schedule"}
                    title={INTAKE_SETTINGS_COPY.scheduleOption}
                    helper={INTAKE_SETTINGS_COPY.scheduleHelper}
                    disabled
                    onChange={() => update("listenMode", "schedule" satisfies IntakeListenMode)}
                  />
                </div>
              </fieldset>

              <GitHubIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={setDraft} />
            </div>
          </div>
          <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
            {saveError ? (
              <p className="workspace-body-text text-destructive" role="alert">
                {saveError}
              </p>
            ) : (
              <span />
            )}
            <Button
              type="button"
              disabled={savePending}
              onClick={async () => {
                try {
                  await onSave(normalizeIntakeSourceSettings(draft));
                  onClose();
                } catch {
                  // The parent supplies the actionable error message.
                }
              }}
              data-testid="intake-source-settings-save"
            >
              {savePending ? INTAKE_SETTINGS_COPY.saving : INTAKE_SETTINGS_COPY.save}
            </Button>
          </footer>
        </>
      )}
    </PopupShell>
  );
}

function IntakeAutomationTab({
  graph,
  title,
  editHref,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  title: string;
  editHref?: string;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <IntakeAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        editHref={editHref}
        onRetry={automationRetry(error, onRetry)}
      />
    );
  }

  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <div className="flex shrink-0 items-center justify-between gap-2 px-5 pt-3 pb-2">
        <p className="min-w-0 truncate text-[15px] font-semibold tracking-[-0.02em] text-foreground">{title}</p>
        {editHref ? (
          <Link href={editHref} className={buttonVariants({ size: "sm" })} data-testid="split-run-canvas-edit">
            {INTAKE_SETTINGS_COPY.editAutomation}
          </Link>
        ) : null}
      </div>
      <div className="min-h-[18rem] flex-1">
        <SettingsAutomationCanvas graph={graph} />
      </div>
    </section>
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return INTAKE_SETTINGS_COPY.automationLoading;
  }
  return error ? INTAKE_SETTINGS_COPY.automationError : INTAKE_SETTINGS_COPY.automationEmpty;
}

function automationRetry(error: boolean, onRetry: (() => void) | undefined): (() => void) | undefined {
  return error ? onRetry : undefined;
}

function IntakeAutomationEmpty({
  message,
  editHref,
  onRetry,
}: {
  message: string;
  editHref?: string;
  onRetry?: () => void;
}) {
  return (
    <section
      className="flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {INTAKE_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
      {editHref ? (
        <Link href={editHref} className={buttonVariants({ size: "sm" })} data-testid="split-run-canvas-edit">
          {INTAKE_SETTINGS_COPY.editAutomation}
        </Link>
      ) : null}
    </section>
  );
}

const PLACEMENT_CHIP_CLASS: Record<IntakeTicketPlacement, string> = {
  progressed:
    "border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]",
  backlog:
    "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)] text-[color:var(--status-draft-fg)]",
  rejected:
    "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]",
  "below-threshold":
    "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)] text-[color:var(--status-cancelled-fg)]",
};

const PLACEMENT_DOT_CLASS: Record<IntakeTicketPlacement, string> = {
  progressed: "bg-[color:var(--status-running-dot)]",
  backlog: "bg-[color:var(--status-draft-dot)]",
  rejected: "bg-[color:var(--status-failed-dot)]",
  "below-threshold": "bg-[color:var(--status-cancelled-dot)]",
};

function IntakeRunsList({
  runs,
  loading,
  error,
  onRetry,
  onOpenRun,
}: {
  runs: IntakeAutomationRun[];
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  onOpenRun?: (run: IntakeAutomationRun) => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="intake-source-runs">
        {INTAKE_SETTINGS_COPY.runsLoading}
      </p>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-3 px-6 py-6" data-testid="intake-source-runs">
        <p className="workspace-body-text text-destructive">{INTAKE_SETTINGS_COPY.runsError}</p>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {INTAKE_SETTINGS_COPY.retryRuns}
          </Button>
        ) : null}
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="intake-source-runs">
        {INTAKE_SETTINGS_COPY.runsEmpty}
      </p>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
      <ul
        className="mx-auto flex w-full max-w-2xl flex-col gap-2"
        data-testid="intake-source-runs"
        aria-label={INTAKE_SETTINGS_COPY.runsTab}
      >
        {runs.map((run) => (
          <li key={run.id}>
            <IntakeRunCard run={run} onOpenRun={onOpenRun} />
          </li>
        ))}
      </ul>
    </div>
  );
}

function IntakeRunCard({
  run,
  onOpenRun,
}: {
  run: IntakeAutomationRun;
  onOpenRun?: (run: IntakeAutomationRun) => void;
}) {
  const placementLabel = intakePlacementLabel(run);
  const activity = intakePlacementActivity(run);

  return (
    <article
      className="group relative w-full rounded-lg border border-border bg-card p-3.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`intake-source-run-${run.id}`}
    >
      {onOpenRun ? (
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-lg"
          aria-label={INTAKE_SETTINGS_COPY.viewRunFor(run.title)}
          onClick={() => onOpenRun(run)}
        />
      ) : null}
      <div className="relative z-10 pointer-events-none">
        <div className="flex items-start gap-3">
          <span className="relative mt-1.5 size-2 shrink-0" title={placementLabel} aria-label={placementLabel}>
            {run.placement === "progressed" ? (
              <span
                className={cn(
                  "absolute inset-0 animate-ping rounded-full opacity-60",
                  PLACEMENT_DOT_CLASS[run.placement],
                )}
                aria-hidden
              />
            ) : null}
            <span className={cn("relative block size-2 rounded-full", PLACEMENT_DOT_CLASS[run.placement])} />
          </span>
          <h3 className="min-w-0 flex-1 text-[13px] font-medium leading-snug tracking-[-0.01em] text-foreground">
            {run.title}
          </h3>
        </div>

        <dl className="mt-3 grid grid-cols-3 gap-3">
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.runWhen}</dt>
            <dd className="mt-0.5 text-[13px] tabular-nums text-foreground">{intakeRelativeTime(run.ranMinutesAgo)}</dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.analysisWhen}</dt>
            <dd className="mt-0.5 text-[13px] tabular-nums text-foreground">
              {intakeRelativeTime(run.analyzedMinutesAgo)}
            </dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{INTAKE_SETTINGS_COPY.scoreWhen}</dt>
            <dd className="mt-0.5 text-[13px] font-medium tabular-nums text-foreground">{run.confidencePct}%</dd>
          </div>
        </dl>

        <div className="mt-3 flex items-start gap-2">
          <Badge variant="outline" className={cn("mt-0.5", PLACEMENT_CHIP_CLASS[run.placement])}>
            {placementLabel}
          </Badge>
          {activity ? <p className="min-w-0 text-[12px] leading-5 text-muted-foreground">{activity}</p> : null}
        </div>

        {onOpenRun ? (
          <p className="mt-3 text-[12px] font-medium text-foreground">{INTAKE_SETTINGS_COPY.viewRun}</p>
        ) : null}
      </div>
    </article>
  );
}
