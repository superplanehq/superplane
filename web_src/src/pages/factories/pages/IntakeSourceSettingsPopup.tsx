import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, History, Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";

import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsTabs,
  normalizeIntakeSourceSettings,
  type IntakeAutomationRun,
  type IntakeListenMode,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { IntakeRunsList } from "./IntakeSettingsRuns";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
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
  agent?: PlanningReviewAgentSlot;
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
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
}: IntakeSourceSettingsPopupProps) {
  const tabs = intakeSettingsTabs(Boolean(agent));
  const hasAgent = Boolean(agent);
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<IntakeSettingsTab>(() => (tabs.includes(initialTab) ? initialTab : "general"));

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  useEffect(() => {
    const next = intakeSettingsTabs(hasAgent);
    if (!next.includes(tab)) {
      setTab("general");
    }
  }, [tab, hasAgent]);

  const update = <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  return (
    <PopupShell testId="intake-source-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={`Intake ${settings.name}`} onClose={onClose}>
        <Tabs value={tab} onValueChange={(value) => setTab(value as IntakeSettingsTab)} className="mt-3">
          <TabsList aria-label={INTAKE_SETTINGS_COPY.tabsLabel}>
            <TabsTrigger value="general" data-testid="intake-settings-tab-general">
              <Settings />
              {INTAKE_SETTINGS_COPY.generalTab}
            </TabsTrigger>
            {tabs.includes("agent") ? (
              <TabsTrigger value="agent" data-testid="intake-settings-tab-agent">
                <Bot />
                {INTAKE_SETTINGS_COPY.agentTab}
              </TabsTrigger>
            ) : null}
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
      <IntakeSettingsTabPanel
        tab={tab}
        settings={settings}
        sourceId={sourceId}
        draft={draft}
        agent={agent}
        automationGraph={automationGraph}
        automationLoading={automationLoading}
        automationError={automationError}
        onRetryAutomation={onRetryAutomation}
        runs={runs}
        runsLoading={runsLoading}
        runsError={runsError}
        onRetryRuns={onRetryRuns}
        onOpenRun={onOpenRun}
        editAutomationHref={editAutomationHref}
        savePending={savePending}
        saveError={saveError}
        onUpdate={update}
        onDraftChange={setDraft}
        onSave={onSave}
        onClose={onClose}
      />
    </PopupShell>
  );
}

function IntakeSettingsTabPanel({
  tab,
  settings,
  sourceId,
  draft,
  agent,
  automationGraph,
  automationLoading,
  automationError,
  onRetryAutomation,
  runs,
  runsLoading,
  runsError,
  onRetryRuns,
  onOpenRun,
  editAutomationHref,
  savePending,
  saveError,
  onUpdate,
  onDraftChange,
  onSave,
  onClose,
}: {
  tab: IntakeSettingsTab;
  settings: IntakeSourceSettings;
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  agent?: PlanningReviewAgentSlot;
  automationGraph?: IntakeAutomationGraph;
  automationLoading: boolean;
  automationError: boolean;
  onRetryAutomation?: () => void;
  runs: IntakeAutomationRun[];
  runsLoading: boolean;
  runsError: boolean;
  onRetryRuns?: () => void;
  onOpenRun?: (run: IntakeAutomationRun) => void;
  editAutomationHref?: string;
  savePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => void;
  onDraftChange: (next: IntakeSourceSettings) => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  if (tab === "automation") {
    return (
      <IntakeAutomationTab
        graph={automationGraph}
        title={settings.name}
        editHref={editAutomationHref}
        loading={automationLoading}
        error={automationError}
        onRetry={onRetryAutomation}
      />
    );
  }
  if (tab === "agent" && agent) {
    return (
      <PlanningReviewEditor
        key={agent.draft?.components[0]?.id ?? "agent"}
        initialDraft={agent.draft}
        onSave={agent.onSave}
        organizationId={agent.organizationId}
        isLoading={agent.isLoading}
        showAutomationNote={false}
        showCancel={false}
      />
    );
  }
  if (tab === "runs") {
    return (
      <IntakeRunsList runs={runs} loading={runsLoading} error={runsError} onRetry={onRetryRuns} onOpenRun={onOpenRun} />
    );
  }
  return (
    <IntakeGeneralTab
      sourceId={sourceId}
      draft={draft}
      savePending={savePending}
      saveError={saveError}
      onUpdate={onUpdate}
      onDraftChange={onDraftChange}
      onSave={onSave}
      onClose={onClose}
    />
  );
}

function IntakeGeneralTab({
  sourceId,
  draft,
  savePending,
  saveError,
  onUpdate,
  onDraftChange,
  onSave,
  onClose,
}: {
  sourceId: LineIntakeSourceId;
  draft: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) => void;
  onDraftChange: (next: IntakeSourceSettings) => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  onClose: () => void;
}) {
  return (
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
              onChange={(event) => onUpdate("name", event.target.value)}
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
                onChange={() => onUpdate("listenMode", "listen" satisfies IntakeListenMode)}
              />
              <IntakeSettingsRadioOption
                name="intake-listen-mode"
                value="schedule"
                checked={draft.listenMode === "schedule"}
                title={INTAKE_SETTINGS_COPY.scheduleOption}
                helper={INTAKE_SETTINGS_COPY.scheduleHelper}
                disabled
                onChange={() => onUpdate("listenMode", "schedule" satisfies IntakeListenMode)}
              />
            </div>
          </fieldset>

          <GitHubIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={onDraftChange} />
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
