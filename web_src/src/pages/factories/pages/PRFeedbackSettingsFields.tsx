import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConnectedIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";
import { organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { sortConnectedIntegrationsByType } from "@/lib/sortConnectedIntegrations";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check } from "lucide-react";
import { useMemo, useState } from "react";

import { isChecksHandlerCIIntegration } from "./checksPRFeedbackSetup";
import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";
import { normalizeReviewBotLogin } from "./reviewBotPickerModel";
import { ReviewBotPicker } from "./ReviewBotPicker";
import {
  catalogReviewBots,
  DISCUSSION_MENTION,
  discussionBotModeFromDraft,
  discussionBotSettings,
  type DiscussionBotMode,
} from "./useDiscussionPRFeedbackSetup";
import { StatusCheckPicker } from "./StatusCheckPicker";

export function PRFeedbackHealthSection({ healthy, checks }: { healthy: boolean; checks: boolean }) {
  const helper = healthy
    ? checks
      ? PR_FEEDBACK_SETTINGS_COPY.healthChecksReadyHelper
      : PR_FEEDBACK_SETTINGS_COPY.healthReadyHelper
    : PR_FEEDBACK_SETTINGS_COPY.healthNeedsRepairHelper;

  return (
    <section>
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">Health</h3>
        <Badge variant="outline" data-testid="pr-feedback-health">
          {healthy ? PR_FEEDBACK_SETTINGS_COPY.healthReady : PR_FEEDBACK_SETTINGS_COPY.healthNeedsRepair}
        </Badge>
      </div>
      <p className="workspace-body-text mt-1 text-muted-foreground">{helper}</p>
    </section>
  );
}

export function PRFeedbackTextField({
  id,
  label,
  helper,
  value,
  onChange,
  type = "text",
  min,
  max,
  step,
}: {
  id: string;
  label: string;
  helper: string;
  value: string;
  onChange: (value: string) => void;
  type?: "text" | "number";
  min?: number;
  max?: number;
  step?: number;
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
        step={step}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        data-testid={id}
      />
    </section>
  );
}

export function PRFeedbackChecksFields({
  organizationId,
  githubIntegrationId,
  draft,
  onUpdate,
}: {
  organizationId?: string;
  githubIntegrationId?: string;
  draft: PRFeedbackDraftSettings;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
}) {
  const catalogEnabled = Boolean(organizationId && githubIntegrationId);
  const catalogParameters = draft.repository.trim() ? { repository: draft.repository.trim() } : undefined;
  const catalogQuery = useIntegrationResources(
    organizationId ?? "",
    githubIntegrationId ?? "",
    "status_check",
    catalogParameters,
    { enabled: catalogEnabled },
  );
  return (
    <>
      <StatusCheckPicker
        names={draft.checkNames}
        catalog={catalogQuery.data ?? []}
        loading={catalogEnabled && (catalogQuery.isPending || catalogQuery.isFetching)}
        loadError={catalogQuery.isError}
        onToggle={(name) => onUpdate("checkNames", toggleUniqueString(draft.checkNames, name))}
      />
      <PRFeedbackTextField
        id="pr-feedback-maximum-attempts"
        label={PR_FEEDBACK_SETTINGS_COPY.maximumAttemptsLabel}
        helper={PR_FEEDBACK_SETTINGS_COPY.maximumAttemptsHelper}
        value={String(draft.maximumAttempts)}
        type="number"
        min={1}
        max={10}
        step={1}
        onChange={(value) => onUpdate("maximumAttempts", Number(value))}
      />
      <PRFeedbackIntegrationsField
        organizationId={organizationId}
        value={draft.runnerIntegrationIds}
        onChange={(value) => onUpdate("runnerIntegrationIds", value)}
      />
    </>
  );
}

export function PRFeedbackDiscussionFields({
  organizationId,
  githubIntegrationId,
  draft,
  onUpdate,
}: {
  organizationId?: string;
  githubIntegrationId?: string;
  draft: PRFeedbackDraftSettings;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
}) {
  const mentionRequired = draft.mention.trim().length > 0;
  const [botMode, setBotMode] = useState<DiscussionBotMode>(() => discussionBotModeFromDraft(draft));
  const catalogEnabled = Boolean(organizationId && githubIntegrationId);
  const catalogParameters = draft.repository.trim() ? { repository: draft.repository.trim() } : undefined;
  const catalogQuery = useIntegrationResources(
    organizationId ?? "",
    githubIntegrationId ?? "",
    "review_bot",
    catalogParameters,
    { enabled: catalogEnabled && botMode === "address" },
  );
  const catalog = catalogReviewBots(catalogQuery.data ?? []);

  const setBotModeAndDraft = (mode: DiscussionBotMode) => {
    setBotMode(mode);
    const next = discussionBotSettings(mode, mode === "address" ? draft.allowedBots : []);
    onUpdate("ignoreBots", next.ignoreBots);
    onUpdate("allowedBots", next.allowedBots);
  };

  return (
    <>
      <section className="space-y-3">
        <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">
          {PR_FEEDBACK_SETTINGS_COPY.wizardStepHumanComments}
        </h3>
        <div
          className="flex flex-col gap-2"
          role="radiogroup"
          aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardStepHumanComments}
        >
          <IntakeSettingsRadioOption
            name="pr-feedback-mention-mode"
            value="require"
            checked={mentionRequired}
            title={PR_FEEDBACK_SETTINGS_COPY.wizardMentionRequireOption}
            helper={PR_FEEDBACK_SETTINGS_COPY.wizardMentionRequireHelper}
            onChange={() => onUpdate("mention", DISCUSSION_MENTION)}
          />
          <IntakeSettingsRadioOption
            name="pr-feedback-mention-mode"
            value="any"
            checked={!mentionRequired}
            title={PR_FEEDBACK_SETTINGS_COPY.wizardMentionAnyOption}
            helper={PR_FEEDBACK_SETTINGS_COPY.wizardMentionAnyHelper}
            onChange={() => onUpdate("mention", "")}
          />
        </div>
      </section>

      <section className="space-y-3">
        <div>
          <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">
            {PR_FEEDBACK_SETTINGS_COPY.wizardStepAIComments}
          </h3>
          <p className="workspace-body-text mt-1 text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsIntro}</p>
        </div>
        <div
          className="flex flex-col gap-2"
          role="radiogroup"
          aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardStepAIComments}
        >
          <IntakeSettingsRadioOption
            name="pr-feedback-bot-mode"
            value="ignore"
            checked={botMode === "ignore"}
            title={PR_FEEDBACK_SETTINGS_COPY.wizardBotIgnoreOption}
            helper={PR_FEEDBACK_SETTINGS_COPY.wizardBotIgnoreHelper}
            onChange={() => setBotModeAndDraft("ignore")}
          />
          <IntakeSettingsRadioOption
            name="pr-feedback-bot-mode"
            value="address"
            checked={botMode === "address"}
            title={PR_FEEDBACK_SETTINGS_COPY.wizardBotAddressOption}
            helper={PR_FEEDBACK_SETTINGS_COPY.wizardBotAddressHelper}
            onChange={() => setBotModeAndDraft("address")}
          />
        </div>
        {botMode === "address" ? (
          <ReviewBotPicker
            selected={draft.allowedBots}
            catalog={catalog}
            loading={catalogEnabled && (catalogQuery.isPending || catalogQuery.isFetching)}
            loadError={catalogQuery.isError}
            onToggle={(login) => onUpdate("allowedBots", toggleUniqueString(draft.allowedBots, login))}
            onAdd={(login) => {
              const normalized = normalizeReviewBotLogin(login);
              if (!normalized) {
                return false;
              }
              if (draft.allowedBots.some((item) => item.toLowerCase() === normalized.toLowerCase())) {
                return true;
              }
              onUpdate("allowedBots", [...draft.allowedBots, normalized]);
              return true;
            }}
          />
        ) : null}
      </section>
    </>
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
    const filtered = (integrationsQuery.data ?? []).filter((integration) => {
      const type = integration.metadata?.integrationName?.toLowerCase();
      return (
        isChecksHandlerCIIntegration(type) && integration.status?.state === "ready" && Boolean(integration.metadata?.id)
      );
    });
    return sortConnectedIntegrationsByType(filtered);
  }, [integrationsQuery.data]);

  return (
    <section>
      <Label>{PR_FEEDBACK_SETTINGS_COPY.integrationsLabel}</Label>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        {PR_FEEDBACK_SETTINGS_COPY.integrationsHelper} {PR_FEEDBACK_SETTINGS_COPY.integrationsMissingBefore}
        {organizationId ? (
          <Link
            href={organizationIntegrationsPath(organizationId)}
            target="_blank"
            rel="noreferrer"
            className="text-gray-800 underline underline-offset-2 dark:text-gray-100"
            data-testid="pr-feedback-integrations-page"
          >
            {PR_FEEDBACK_SETTINGS_COPY.integrationsMissingLink}
          </Link>
        ) : (
          PR_FEEDBACK_SETTINGS_COPY.integrationsMissingLink
        )}
        {PR_FEEDBACK_SETTINGS_COPY.integrationsMissingAfter}
      </p>
      <div
        className="mt-2 max-h-56 overflow-y-auto rounded-lg border border-border"
        role="listbox"
        aria-label={PR_FEEDBACK_SETTINGS_COPY.integrationsLabel}
        aria-multiselectable="true"
        data-testid="pr-feedback-integrations"
      >
        {options.length === 0 ? (
          <p
            className="px-3 py-6 text-center text-[13px] text-muted-foreground"
            data-testid="pr-feedback-integrations-empty"
          >
            {PR_FEEDBACK_SETTINGS_COPY.integrationsEmpty}
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {options.map((integration) => {
              const id = integration.metadata?.id ?? "";
              const name = integration.metadata?.name || integration.metadata?.integrationName || id;
              const selected = value.includes(id);
              return (
                <li key={id}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onChange(toggleUniqueString(value, id))}
                    className={cn(
                      "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                    data-testid={`pr-feedback-integration-${id}`}
                  >
                    <IntegrationIcon
                      integrationName={integration.metadata?.integrationName}
                      className="size-4 shrink-0"
                      size={16}
                    />
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{name}</span>
                    {selected ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}
