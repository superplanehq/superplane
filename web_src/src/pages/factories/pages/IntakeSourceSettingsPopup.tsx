import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { History, Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";

import {
  GITHUB_INTAKE_LABEL_OPTIONS,
  GITHUB_INTAKE_RUNS,
  INTAKE_SETTINGS_COPY,
  intakePlacementActivity,
  intakePlacementLabel,
  intakeRelativeTime,
  normalizeIntakeSourceSettings,
  toggleIntakeLabel,
  type IntakeAssignmentFilter,
  type IntakeAutomationRun,
  type IntakeLabelFilterMode,
  type IntakeListenMode,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
  type IntakeTicketPlacement,
} from "./intakeSourceSettingsModel";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import { CompactLineCanvas } from "./work-order-split-run/CompactLineCanvas";
import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  automationCanvas: SplitRunCanvasModel;
  runs?: IntakeAutomationRun[];
  onSave: (next: IntakeSourceSettings) => void;
  onOpenRun?: (run: IntakeAutomationRun) => void;
  editAutomationHref?: string;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: IntakeSettingsTab;
}

export function IntakeSourceSettingsPopup({
  settings,
  automationCanvas,
  runs = GITHUB_INTAKE_RUNS,
  onSave,
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
      <PopupHeader title={INTAKE_SETTINGS_COPY.title} onClose={onClose}>
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
        <IntakeAutomationCanvas canvas={automationCanvas} editHref={editAutomationHref} />
      ) : tab === "runs" ? (
        <IntakeRunsList runs={runs} onOpenRun={onOpenRun} />
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
                  <RadioOption
                    name="intake-listen-mode"
                    value="listen"
                    checked={draft.listenMode === "listen"}
                    title={INTAKE_SETTINGS_COPY.listenOption}
                    helper={INTAKE_SETTINGS_COPY.listenHelper}
                    onChange={() => update("listenMode", "listen" satisfies IntakeListenMode)}
                  />
                  <RadioOption
                    name="intake-listen-mode"
                    value="schedule"
                    checked={draft.listenMode === "schedule"}
                    title={INTAKE_SETTINGS_COPY.scheduleOption}
                    helper={INTAKE_SETTINGS_COPY.scheduleHelper}
                    onChange={() => update("listenMode", "schedule" satisfies IntakeListenMode)}
                  />
                </div>
              </fieldset>

              <section>
                <div className="flex items-baseline justify-between gap-3">
                  <Label htmlFor="intake-confidence-slider">{INTAKE_SETTINGS_COPY.confidenceLabel}</Label>
                  <span
                    className="text-[13px] font-medium tabular-nums text-foreground"
                    data-testid="intake-confidence-value"
                  >
                    {draft.confidencePct}%
                  </span>
                </div>
                <p className="workspace-body-text mt-1 text-muted-foreground">
                  {INTAKE_SETTINGS_COPY.confidenceHelper}
                </p>
                <Slider
                  id="intake-confidence-slider"
                  className="mt-4 max-w-xs"
                  min={0}
                  max={100}
                  step={1}
                  value={[draft.confidencePct]}
                  onValueChange={(values) => update("confidencePct", values[0] ?? draft.confidencePct)}
                  aria-label={INTAKE_SETTINGS_COPY.confidenceLabel}
                  data-testid="intake-confidence-slider"
                />
              </section>

              <section className="flex flex-col gap-6">
                <h3 className="workspace-section-title">{INTAKE_SETTINGS_COPY.filtersLabel}</h3>

                <fieldset className="min-w-0">
                  <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
                    {INTAKE_SETTINGS_COPY.labelsLabel}
                  </legend>
                  <p className="workspace-body-text mt-1 text-muted-foreground">{INTAKE_SETTINGS_COPY.labelsHelper}</p>
                  <div className="mt-2 flex flex-col gap-2">
                    <RadioOption
                      name="intake-label-filter"
                      value="include"
                      checked={draft.labelFilterMode === "include"}
                      title={INTAKE_SETTINGS_COPY.includeLabels}
                      onChange={() => update("labelFilterMode", "include" satisfies IntakeLabelFilterMode)}
                    />
                    <RadioOption
                      name="intake-label-filter"
                      value="exclude"
                      checked={draft.labelFilterMode === "exclude"}
                      title={INTAKE_SETTINGS_COPY.excludeLabels}
                      onChange={() => update("labelFilterMode", "exclude" satisfies IntakeLabelFilterMode)}
                    />
                  </div>
                  <ul className="mt-3 flex flex-wrap gap-2" data-testid="intake-label-options">
                    {GITHUB_INTAKE_LABEL_OPTIONS.map((label) => {
                      const checked = draft.labels.includes(label);
                      return (
                        <li key={label}>
                          <label
                            className={cn(
                              "inline-flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-1.5 text-[13px]",
                              checked
                                ? "border-foreground/20 bg-accent/50 text-foreground"
                                : "border-border bg-card text-muted-foreground hover:border-foreground/15",
                            )}
                          >
                            <Checkbox
                              checked={checked}
                              onChange={() => update("labels", toggleIntakeLabel(draft.labels, label))}
                              aria-label={label}
                            />
                            {label}
                          </label>
                        </li>
                      );
                    })}
                  </ul>
                </fieldset>

                <fieldset className="min-w-0">
                  <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">
                    {INTAKE_SETTINGS_COPY.assignmentLabel}
                  </legend>
                  <div className="mt-2 flex flex-col gap-2">
                    <RadioOption
                      name="intake-assignment"
                      value="any"
                      checked={draft.assignment === "any"}
                      title={INTAKE_SETTINGS_COPY.assignmentAny}
                      onChange={() => update("assignment", "any" satisfies IntakeAssignmentFilter)}
                    />
                    <RadioOption
                      name="intake-assignment"
                      value="assigned"
                      checked={draft.assignment === "assigned"}
                      title={INTAKE_SETTINGS_COPY.assignmentAssigned}
                      onChange={() => update("assignment", "assigned" satisfies IntakeAssignmentFilter)}
                    />
                    <RadioOption
                      name="intake-assignment"
                      value="unassigned"
                      checked={draft.assignment === "unassigned"}
                      title={INTAKE_SETTINGS_COPY.assignmentUnassigned}
                      onChange={() => update("assignment", "unassigned" satisfies IntakeAssignmentFilter)}
                    />
                  </div>
                </fieldset>
              </section>
            </div>
          </div>
          <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
            <Button
              type="button"
              onClick={() => {
                onSave(normalizeIntakeSourceSettings(draft));
                onClose();
              }}
              data-testid="intake-source-settings-save"
            >
              {INTAKE_SETTINGS_COPY.save}
            </Button>
          </footer>
        </>
      )}
    </PopupShell>
  );
}

function RadioOption({
  name,
  value,
  checked,
  title,
  helper,
  onChange,
}: {
  name: string;
  value: string;
  checked: boolean;
  title: string;
  helper?: string;
  onChange: () => void;
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        onChange={onChange}
        className="mt-0.5 size-4 accent-gray-900"
      />
      <span className="min-w-0">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
        {helper ? <span className="mt-0.5 block text-[12px] leading-5 text-muted-foreground">{helper}</span> : null}
      </span>
    </label>
  );
}

function IntakeAutomationCanvas({ canvas, editHref }: { canvas: SplitRunCanvasModel; editHref?: string }) {
  const [nodeId, setNodeId] = useState<string | null>(null);

  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <CompactLineCanvas
        canvas={canvas}
        selectedId={nodeId}
        onSelect={setNodeId}
        headerEdit="button"
        editHref={editHref}
        editLabel={INTAKE_SETTINGS_COPY.editAutomation}
      />
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
  onOpenRun,
}: {
  runs: IntakeAutomationRun[];
  onOpenRun?: (run: IntakeAutomationRun) => void;
}) {
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
