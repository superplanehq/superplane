import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/buttonVariants";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { CanvasPage } from "@/ui/CanvasPage";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Settings, Workflow, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";
import {
  PR_FEEDBACK_SETTINGS_COPY,
  appendUniqueTrimmedString,
  normalizePRFeedbackDraft,
  prFeedbackDraftIsValid,
  type PRFeedbackDraftSettings,
  type PRFeedbackSettingsTab,
} from "./prFeedbackSettingsModel";

interface PRFeedbackSettingsPopupProps {
  organizationId?: string;
  settings: PRFeedbackDraftSettings;
  healthy: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  editAutomationHref?: string;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: PRFeedbackSettingsTab;
}

export function PRFeedbackSettingsPopup({
  organizationId,
  settings,
  healthy,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  onDelete,
  savePending = false,
  deletePending = false,
  saveError,
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
      ) : (
        <PRFeedbackGeneralTab
          organizationId={organizationId}
          draft={draft}
          healthy={healthy}
          confirmDelete={confirmDelete}
          savePending={savePending}
          deletePending={deletePending}
          saveError={saveError}
          onUpdate={(key, value) => update(key, value)}
          onSave={onSave}
          onDelete={onDelete}
          onConfirmDelete={(next) => setConfirmDelete(next)}
          onClose={onClose}
        />
      )}
    </PopupShell>
  );
}

function PRFeedbackGeneralTab({
  organizationId,
  draft,
  healthy,
  confirmDelete,
  savePending,
  deletePending,
  saveError,
  onUpdate,
  onSave,
  onDelete,
  onConfirmDelete,
  onClose,
}: {
  organizationId?: string;
  draft: PRFeedbackDraftSettings;
  healthy: boolean;
  confirmDelete: boolean;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onClose: () => void;
}) {
  const checks = draft.source === "checks";
  const [checkNameInput, setCheckNameInput] = useState("");

  function addCheckName() {
    onUpdate("checkNames", appendUniqueTrimmedString(draft.checkNames, checkNameInput));
    setCheckNameInput("");
  }

  return (
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
                ? checks
                  ? PR_FEEDBACK_SETTINGS_COPY.healthChecksReadyHelper
                  : PR_FEEDBACK_SETTINGS_COPY.healthReadyHelper
                : PR_FEEDBACK_SETTINGS_COPY.healthNeedsRepairHelper}
            </p>
          </section>

          <PRFeedbackTextField
            id="pr-feedback-name"
            label={PR_FEEDBACK_SETTINGS_COPY.nameLabel}
            helper={PR_FEEDBACK_SETTINGS_COPY.nameHelper}
            value={draft.name}
            onChange={(value) => onUpdate("name", value)}
          />
          <PRFeedbackTextField
            id="pr-feedback-repository"
            label={PR_FEEDBACK_SETTINGS_COPY.repositoryLabel}
            helper={
              checks ? PR_FEEDBACK_SETTINGS_COPY.checksRepositoryHelper : PR_FEEDBACK_SETTINGS_COPY.repositoryHelper
            }
            value={draft.repository}
            onChange={(value) => onUpdate("repository", value)}
          />
          {checks ? (
            <>
              <PRFeedbackCheckNamesField
                id="pr-feedback-check-names"
                names={draft.checkNames}
                inputValue={checkNameInput}
                onInputChange={setCheckNameInput}
                onAdd={addCheckName}
                onRemove={(name) =>
                  onUpdate(
                    "checkNames",
                    draft.checkNames.filter((item) => item !== name),
                  )
                }
              />
              <PRFeedbackTextField
                id="pr-feedback-maximum-attempts"
                label={PR_FEEDBACK_SETTINGS_COPY.maximumAttemptsLabel}
                helper={PR_FEEDBACK_SETTINGS_COPY.maximumAttemptsHelper}
                value={String(draft.maximumAttempts)}
                type="number"
                min={1}
                max={10}
                onChange={(value) => onUpdate("maximumAttempts", Number(value))}
              />
              <PRFeedbackIntegrationsField
                organizationId={organizationId}
                value={draft.runnerIntegrationIds}
                onChange={(value) => onUpdate("runnerIntegrationIds", value)}
              />
            </>
          ) : (
            <>
              <PRFeedbackTextField
                id="pr-feedback-mention"
                label={PR_FEEDBACK_SETTINGS_COPY.mentionLabel}
                helper={PR_FEEDBACK_SETTINGS_COPY.mentionHelper}
                value={draft.mention}
                onChange={(value) => onUpdate("mention", value)}
              />

              <div className="flex items-start gap-3">
                <Checkbox
                  id="pr-feedback-ignore-bots"
                  className="mt-0.5 cursor-pointer"
                  checked={draft.ignoreBots}
                  onChange={(event) => onUpdate("ignoreBots", event.currentTarget.checked)}
                  data-testid="pr-feedback-ignore-bots"
                />
                <Label htmlFor="pr-feedback-ignore-bots" className="flex-col items-start cursor-pointer">
                  <span className="block text-sm font-medium text-gray-800 dark:text-gray-100">
                    {PR_FEEDBACK_SETTINGS_COPY.ignoreBotsLabel}
                  </span>
                  <span className="workspace-body-text mt-1 block text-muted-foreground">
                    {PR_FEEDBACK_SETTINGS_COPY.ignoreBotsHelper}
                  </span>
                </Label>
              </div>

              <PRFeedbackListField
                id="pr-feedback-allowed-bots"
                label={PR_FEEDBACK_SETTINGS_COPY.allowedBotsLabel}
                helper={PR_FEEDBACK_SETTINGS_COPY.allowedBotsHelper}
                placeholder="coderabbitai, bugbot"
                value={draft.allowedBots}
                onChange={(value) => onUpdate("allowedBots", value)}
              />
            </>
          )}
        </div>
      </div>
      <PRFeedbackSettingsFooter
        draft={draft}
        pendingCheckName={checkNameInput}
        confirmDelete={confirmDelete}
        savePending={savePending}
        deletePending={deletePending}
        saveError={saveError}
        onSave={onSave}
        onDelete={onDelete}
        onConfirmDelete={onConfirmDelete}
        onClose={onClose}
      />
    </>
  );
}

function PRFeedbackTextField({
  id,
  label,
  helper,
  value,
  onChange,
  type = "text",
  min,
  max,
}: {
  id: string;
  label: string;
  helper: string;
  value: string;
  onChange: (value: string) => void;
  type?: "text" | "number";
  min?: number;
  max?: number;
}) {
  return (
    <section>
      <Label htmlFor={id}>{label}</Label>
      <p className="workspace-body-text mt-1 text-muted-foreground">{helper}</p>
      <Input
        id={id}
        className="mt-2"
        type={type}
        min={min}
        max={max}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        data-testid={id}
      />
    </section>
  );
}

function PRFeedbackCheckNamesField({
  id,
  names,
  inputValue,
  onInputChange,
  onAdd,
  onRemove,
}: {
  id: string;
  names: string[];
  inputValue: string;
  onInputChange: (value: string) => void;
  onAdd: () => void;
  onRemove: (name: string) => void;
}) {
  return (
    <section>
      <Label htmlFor={id}>{PR_FEEDBACK_SETTINGS_COPY.checkNamesLabel}</Label>
      <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.checkNamesHelper}</p>
      {names.length > 0 ? (
        <ul className="mt-2 flex flex-wrap items-center gap-1.5" data-testid={`${id}-list`}>
          {names.map((name) => (
            <li
              key={name}
              className="inline-flex max-w-full items-center gap-1 rounded-md border border-border bg-background px-2 py-1 text-sm text-foreground"
            >
              <span className="min-w-0 truncate">{name}</span>
              <button
                type="button"
                onClick={() => onRemove(name)}
                aria-label={`Remove check ${name}`}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                data-testid={`${id}-remove`}
              >
                <X className="size-3" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="mt-2 flex items-center gap-2">
        <Input
          id={id}
          placeholder={PR_FEEDBACK_SETTINGS_COPY.checkNamesPlaceholder}
          value={inputValue}
          onChange={(event) => onInputChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key !== "Enter") {
              return;
            }
            event.preventDefault();
            onAdd();
          }}
          data-testid={id}
        />
        <Button type="button" variant="outline" size="sm" onClick={onAdd} data-testid={`${id}-add`}>
          {PR_FEEDBACK_SETTINGS_COPY.checkNamesAdd}
        </Button>
      </div>
    </section>
  );
}

function PRFeedbackListField({
  id,
  label,
  helper,
  extraHelper,
  placeholder,
  value,
  onChange,
}: {
  id: string;
  label: string;
  helper: string;
  extraHelper?: string;
  placeholder?: string;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const [text, setText] = useState(value.join(", "));

  useEffect(() => {
    setText(value.join(", "));
  }, [value]);

  function commit(nextText: string) {
    const items = nextText
      .split(",")
      .map((item) => item.trim())
      .filter((item) => item.length > 0);
    onChange(items);
  }

  return (
    <section>
      <Label htmlFor={id}>{label}</Label>
      <p className="workspace-body-text mt-1 text-muted-foreground">{helper}</p>
      {extraHelper ? <p className="workspace-body-text mt-1 text-muted-foreground">{extraHelper}</p> : null}
      <Input
        id={id}
        className="mt-2"
        placeholder={placeholder}
        value={text}
        onChange={(event) => setText(event.target.value)}
        onBlur={(event) => commit(event.target.value)}
        data-testid={id}
      />
    </section>
  );
}

function PRFeedbackIntegrationsField({
  organizationId,
  value,
  onChange,
}: {
  organizationId?: string;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const integrationsQuery = useConnectedIntegrations(organizationId ?? "", { enabled: Boolean(organizationId) });
  const options = useMemo(() => {
    return (integrationsQuery.data ?? []).filter((integration) => {
      const type = integration.metadata?.integrationName?.toLowerCase();
      return type !== "github" && integration.status?.state === "ready" && integration.metadata?.id;
    });
  }, [integrationsQuery.data]);

  return (
    <section>
      <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">
        {PR_FEEDBACK_SETTINGS_COPY.integrationsLabel}
      </h3>
      <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.integrationsHelper}</p>
      {options.length === 0 ? (
        <p className="workspace-body-text mt-2 text-muted-foreground" data-testid="pr-feedback-integrations-empty">
          {PR_FEEDBACK_SETTINGS_COPY.integrationsEmpty}
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-2" data-testid="pr-feedback-integrations">
          {options.map((integration) => {
            const id = integration.metadata?.id ?? "";
            const name = integration.metadata?.name || integration.metadata?.integrationName || id;
            const checked = value.includes(id);
            return (
              <li key={id} className="flex items-center gap-2">
                <Checkbox
                  id={`pr-feedback-integration-${id}`}
                  className="cursor-pointer"
                  checked={checked}
                  onChange={(event) => {
                    if (event.currentTarget.checked) {
                      onChange([...value, id]);
                      return;
                    }
                    onChange(value.filter((item) => item !== id));
                  }}
                  data-testid={`pr-feedback-integration-${id}`}
                />
                <Label htmlFor={`pr-feedback-integration-${id}`} className="min-w-0 cursor-pointer gap-2">
                  <IntegrationIcon
                    integrationName={integration.metadata?.integrationName}
                    className="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400"
                  />
                  <span className="truncate">{name}</span>
                </Label>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function PRFeedbackSettingsFooter({
  draft,
  pendingCheckName,
  confirmDelete,
  savePending,
  deletePending,
  saveError,
  onSave,
  onDelete,
  onConfirmDelete,
  onClose,
}: {
  draft: PRFeedbackDraftSettings;
  pendingCheckName: string;
  confirmDelete: boolean;
  savePending?: boolean;
  deletePending?: boolean;
  saveError?: string;
  onSave: (next: PRFeedbackDraftSettings) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onConfirmDelete: (next: boolean) => void;
  onClose: () => void;
}) {
  if (confirmDelete) {
    return (
      <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
        <p className="workspace-body-text text-destructive" role="alert">
          {PR_FEEDBACK_SETTINGS_COPY.confirmDelete}
        </p>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => onConfirmDelete(false)}>
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
      </footer>
    );
  }

  return (
    <footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-5 py-3">
      {saveError ? (
        <p className="workspace-body-text text-destructive" role="alert">
          {saveError}
        </p>
      ) : onDelete ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onConfirmDelete(true)}
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
            await onSave(
              normalizePRFeedbackDraft({
                ...draft,
                checkNames: appendUniqueTrimmedString(draft.checkNames, pendingCheckName),
              }),
            );
            onClose();
          } catch {
            // The parent supplies the actionable error message.
          }
        }}
        data-testid="pr-feedback-settings-save"
      >
        {savePending ? PR_FEEDBACK_SETTINGS_COPY.saving : PR_FEEDBACK_SETTINGS_COPY.save}
      </Button>
    </footer>
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
