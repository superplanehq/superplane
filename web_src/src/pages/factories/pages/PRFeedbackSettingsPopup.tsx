import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { CanvasPage } from "@/ui/CanvasPage";
import { History, Settings, Workflow } from "lucide-react";
import { useEffect, useState } from "react";
import type { FactoriesFactoryPrFeedbackHandlerRun } from "@/api-client";

import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  normalizePRFeedbackDraft,
  prFeedbackDraftIsValid,
  prFeedbackRunStatusLabel,
  prFeedbackRunTimeLabel,
  prFeedbackRunTitle,
  prFeedbackRunTriggerLabel,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";

interface PRFeedbackSettingsPopupProps {
  settings: PRFeedbackDraftSettings;
  healthy: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  runs?: FactoriesFactoryPrFeedbackHandlerRun[];
  runsLoading?: boolean;
  runsError?: boolean;
  onRetryRuns?: () => void;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  onOpenRun?: (run: FactoriesFactoryPrFeedbackHandlerRun) => void;
  workOrderHrefFor?: (workOrderId: string) => string | undefined;
  editAutomationHref?: string;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PRFeedbackSettingsTab;
}

const STATUS_CHIP_CLASS: Record<string, string> = {
  STATUS_QUEUED:
    "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)] text-[color:var(--status-draft-fg)]",
  STATUS_RUNNING:
    "border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]",
  STATUS_PASSED:
    "border-[color:var(--status-passed-border)] bg-[color:var(--status-passed-bg)] text-[color:var(--status-passed-fg)]",
  STATUS_FAILED:
    "border-[color:var(--status-failed-border)] bg-[color:var(--status-failed-bg)] text-[color:var(--status-failed-fg)]",
  STATUS_CANCELLED:
    "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)] text-[color:var(--status-cancelled-fg)]",
};

export function PRFeedbackSettingsPopup({
  settings,
  healthy,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  runs = [],
  runsLoading = false,
  runsError = false,
  onRetryRuns,
  onSave,
  onDelete,
  savePending = false,
  deletePending = false,
  saveError,
  onOpenRun,
  workOrderHrefFor,
  editAutomationHref,
  onClose,
  fixed = true,
  initialTab = "general",
}: PRFeedbackSettingsPopupProps) {
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<PRFeedbackSettingsTab>(initialTab);
  const [confirmDelete, setConfirmDelete] = useState(false);

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  function update<K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  return (
    <PopupShell testId="pr-feedback-settings" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={settings.name} onClose={onClose}>
        <Tabs value={tab} onValueChange={(value) => setTab(value as PRFeedbackSettingsTab)} className="mt-3">
          <TabsList aria-label={PR_FEEDBACK_SETTINGS_COPY.tabsLabel}>
            <TabsTrigger value="general" data-testid="pr-feedback-settings-tab-general">
              <Settings />
              {PR_FEEDBACK_SETTINGS_COPY.generalTab}
            </TabsTrigger>
            <TabsTrigger value="runs" data-testid="pr-feedback-settings-tab-runs">
              <History />
              {PR_FEEDBACK_SETTINGS_COPY.runsTab}
            </TabsTrigger>
            <TabsTrigger value="automation" data-testid="pr-feedback-settings-tab-automation">
              <Workflow />
              {PR_FEEDBACK_SETTINGS_COPY.automationTab}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </PopupHeader>
      {tab === "automation" ? (
        <PRFeedbackAutomationTab
          graph={automationGraph}
          title={settings.name}
          editHref={editAutomationHref}
          loading={automationLoading}
          error={automationError}
          onRetry={onRetryAutomation}
        />
      ) : tab === "runs" ? (
        <PRFeedbackRunsList
          runs={runs}
          loading={runsLoading}
          error={runsError}
          onRetry={onRetryRuns}
          onOpenRun={onOpenRun}
          workOrderHrefFor={workOrderHrefFor}
        />
      ) : (
        <>
          <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
            <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
              <section>
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">Health</h3>
                  <Badge variant="outline" data-testid="pr-feedback-health">
                    {healthy ? PR_FEEDBACK_SETTINGS_COPY.healthReady : PR_FEEDBACK_SETTINGS_COPY.healthNeedsRepair}
                  </Badge>
                </div>
                <p className="workspace-body-text mt-1 text-muted-foreground">
                  {healthy
                    ? PR_FEEDBACK_SETTINGS_COPY.healthReadyHelper
                    : PR_FEEDBACK_SETTINGS_COPY.healthNeedsRepairHelper}
                </p>
              </section>

              <section>
                <Label htmlFor="pr-feedback-name">{PR_FEEDBACK_SETTINGS_COPY.nameLabel}</Label>
                <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.nameHelper}</p>
                <Input
                  id="pr-feedback-name"
                  className="mt-2"
                  value={draft.name}
                  onChange={(event) => update("name", event.target.value)}
                  data-testid="pr-feedback-name"
                />
              </section>

              <section>
                <Label htmlFor="pr-feedback-repository">{PR_FEEDBACK_SETTINGS_COPY.repositoryLabel}</Label>
                <p className="workspace-body-text mt-1 text-muted-foreground">
                  {PR_FEEDBACK_SETTINGS_COPY.repositoryHelper}
                </p>
                <Input
                  id="pr-feedback-repository"
                  className="mt-2"
                  value={draft.repository}
                  onChange={(event) => update("repository", event.target.value)}
                  data-testid="pr-feedback-repository"
                />
              </section>

              <section>
                <Label htmlFor="pr-feedback-mention">{PR_FEEDBACK_SETTINGS_COPY.mentionLabel}</Label>
                <p className="workspace-body-text mt-1 text-muted-foreground">
                  {PR_FEEDBACK_SETTINGS_COPY.mentionHelper}
                </p>
                <Input
                  id="pr-feedback-mention"
                  className="mt-2"
                  value={draft.mention}
                  onChange={(event) => update("mention", event.target.value)}
                  data-testid="pr-feedback-mention"
                />
              </section>

              <label className="flex items-start gap-3">
                <Checkbox
                  className="mt-0.5"
                  checked={draft.ignoreBots}
                  onChange={(event) => update("ignoreBots", event.currentTarget.checked)}
                  data-testid="pr-feedback-ignore-bots"
                />
                <span>
                  <span className="block text-sm font-medium text-gray-800 dark:text-gray-100">
                    {PR_FEEDBACK_SETTINGS_COPY.ignoreBotsLabel}
                  </span>
                  <span className="workspace-body-text mt-1 block text-muted-foreground">
                    {PR_FEEDBACK_SETTINGS_COPY.ignoreBotsHelper}
                  </span>
                </span>
              </label>
            </div>
          </div>
          <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
            {confirmDelete ? (
              <>
                <p className="workspace-body-text text-destructive" role="alert">
                  {PR_FEEDBACK_SETTINGS_COPY.confirmDelete}
                </p>
                <div className="flex items-center gap-2">
                  <Button type="button" variant="outline" size="sm" onClick={() => setConfirmDelete(false)}>
                    {PR_FEEDBACK_SETTINGS_COPY.keep}
                  </Button>
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    disabled={deletePending}
                    onClick={() => void onDelete?.()}
                    data-testid="pr-feedback-settings-delete-confirm"
                  >
                    {deletePending ? PR_FEEDBACK_SETTINGS_COPY.deleting : PR_FEEDBACK_SETTINGS_COPY.delete}
                  </Button>
                </div>
              </>
            ) : (
              <>
                {saveError ? (
                  <p className="workspace-body-text text-destructive" role="alert">
                    {saveError}
                  </p>
                ) : onDelete ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setConfirmDelete(true)}
                    data-testid="pr-feedback-settings-delete"
                  >
                    {PR_FEEDBACK_SETTINGS_COPY.delete}
                  </Button>
                ) : (
                  <span />
                )}
                <Button
                  type="button"
                  disabled={savePending || !prFeedbackDraftIsValid(draft)}
                  onClick={async () => {
                    try {
                      await onSave(normalizePRFeedbackDraft(draft));
                      onClose();
                    } catch {
                      // The parent supplies the actionable error message.
                    }
                  }}
                  data-testid="pr-feedback-settings-save"
                >
                  {savePending ? PR_FEEDBACK_SETTINGS_COPY.saving : PR_FEEDBACK_SETTINGS_COPY.save}
                </Button>
              </>
            )}
          </footer>
        </>
      )}
    </PopupShell>
  );
}

function PRFeedbackAutomationTab({
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
      <PRFeedbackAutomationEmpty
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
      data-testid="pr-feedback-automation"
    >
      <div className="flex shrink-0 items-center justify-between gap-2 px-5 pt-3 pb-2">
        <p className="min-w-0 truncate text-[15px] font-semibold tracking-[-0.02em] text-foreground">{title}</p>
        {editHref ? (
          <Link href={editHref} className={buttonVariants({ size: "sm" })}>
            {PR_FEEDBACK_SETTINGS_COPY.editAutomation}
          </Link>
        ) : null}
      </div>
      <div className="min-h-[18rem] flex-1">
        <CanvasPage
          nodes={graph.nodes}
          edges={graph.edges}
          factoryId={graph.factoryId}
          factoryEmbed
          isEditing
          readOnly
          hidePageChrome
          hideAddControls
          hideCanvasToolSidebar
          hideRightSideControls
          buildingBlocks={[]}
          activeCanvasVersionId=""
        />
      </div>
    </section>
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return PR_FEEDBACK_SETTINGS_COPY.automationLoading;
  }
  return error ? PR_FEEDBACK_SETTINGS_COPY.automationError : PR_FEEDBACK_SETTINGS_COPY.automationEmpty;
}

function automationRetry(error: boolean, onRetry: (() => void) | undefined): (() => void) | undefined {
  return error ? onRetry : undefined;
}

function PRFeedbackAutomationEmpty({
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
      data-testid="pr-feedback-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {PR_FEEDBACK_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
      {editHref ? (
        <Link href={editHref} className={buttonVariants({ size: "sm" })}>
          {PR_FEEDBACK_SETTINGS_COPY.editAutomation}
        </Link>
      ) : null}
    </section>
  );
}

function PRFeedbackRunsList({
  runs,
  loading,
  error,
  onRetry,
  onOpenRun,
  workOrderHrefFor,
}: {
  runs: FactoriesFactoryPrFeedbackHandlerRun[];
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
  onOpenRun?: (run: FactoriesFactoryPrFeedbackHandlerRun) => void;
  workOrderHrefFor?: (workOrderId: string) => string | undefined;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="pr-feedback-runs">
        {PR_FEEDBACK_SETTINGS_COPY.runsLoading}
      </p>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-3 px-6 py-6" data-testid="pr-feedback-runs">
        <p className="workspace-body-text text-destructive">{PR_FEEDBACK_SETTINGS_COPY.runsError}</p>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {PR_FEEDBACK_SETTINGS_COPY.retryRuns}
          </Button>
        ) : null}
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <p className="workspace-body-text px-6 py-6 text-muted-foreground" data-testid="pr-feedback-runs">
        {PR_FEEDBACK_SETTINGS_COPY.runsEmpty}
      </p>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
      <ul
        className="mx-auto flex w-full max-w-2xl flex-col gap-2"
        data-testid="pr-feedback-runs"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.runsTab}
      >
        {runs.map((run) => (
          <li key={run.id}>
            <PRFeedbackRunCard run={run} onOpenRun={onOpenRun} workOrderHrefFor={workOrderHrefFor} />
          </li>
        ))}
      </ul>
    </div>
  );
}

function PRFeedbackRunCard({
  run,
  onOpenRun,
  workOrderHrefFor,
}: {
  run: FactoriesFactoryPrFeedbackHandlerRun;
  onOpenRun?: (run: FactoriesFactoryPrFeedbackHandlerRun) => void;
  workOrderHrefFor?: (workOrderId: string) => string | undefined;
}) {
  const title = prFeedbackRunTitle(run);
  const workOrderHref = run.workOrderId ? workOrderHrefFor?.(run.workOrderId) : undefined;
  const commentHref = run.triggerUrl;
  const pullRequestHref = run.pullRequestUrl;

  return (
    <article
      className="group relative w-full rounded-lg border border-border bg-card p-3.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`pr-feedback-run-${run.id}`}
    >
      {onOpenRun ? (
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-lg"
          aria-label={PR_FEEDBACK_SETTINGS_COPY.viewRun}
          onClick={() => onOpenRun(run)}
        />
      ) : null}
      <div className="relative z-10 pointer-events-none">
        <div className="flex items-start justify-between gap-3">
          <h3 className="min-w-0 flex-1 text-[13px] font-medium leading-snug tracking-[-0.01em] text-foreground">
            {title}
          </h3>
          <Badge variant="outline" className={cn("mt-0.5", STATUS_CHIP_CLASS[run.status ?? ""])}>
            {prFeedbackRunStatusLabel(run.status)}
          </Badge>
        </div>
        <dl className="mt-3 grid grid-cols-3 gap-3">
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.runWhen}</dt>
            <dd className="mt-0.5 text-[13px] tabular-nums text-foreground">{prFeedbackRunTimeLabel(run)}</dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.runTrigger}</dt>
            <dd className="mt-0.5 text-[13px] text-foreground">{prFeedbackRunTriggerLabel(run.trigger)}</dd>
          </div>
          <div>
            <dt className="text-[11px] font-medium text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.runStatus}</dt>
            <dd className="mt-0.5 text-[13px] text-foreground">{prFeedbackRunStatusLabel(run.status)}</dd>
          </div>
        </dl>
      </div>
      <div className="relative z-10 mt-3 flex flex-wrap gap-3 pointer-events-auto">
        {workOrderHref ? (
          <Link href={workOrderHref} className="text-[12px] font-medium text-foreground">
            {PR_FEEDBACK_SETTINGS_COPY.viewWorkOrder}
          </Link>
        ) : null}
        {pullRequestHref ? (
          <a
            href={pullRequestHref}
            target="_blank"
            rel="noreferrer"
            className="text-[12px] font-medium text-foreground"
          >
            {PR_FEEDBACK_SETTINGS_COPY.viewPullRequest}
          </a>
        ) : null}
        {commentHref && commentHref !== pullRequestHref ? (
          <a href={commentHref} target="_blank" rel="noreferrer" className="text-[12px] font-medium text-foreground">
            {PR_FEEDBACK_SETTINGS_COPY.viewComment}
          </a>
        ) : null}
      </div>
    </article>
  );
}
