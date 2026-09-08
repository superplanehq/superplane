import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { organizationIntegrationsPath } from "@/lib/integrationSettingsPaths";
import { sortConnectedIntegrationsByType } from "@/lib/sortConnectedIntegrations";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { useEffect, useMemo, useState } from "react";

import { useFactoryRepositoryStatusChecks } from "@/hooks/useFactoryPRFeedbackData";

import { PR_FEEDBACK_SETTINGS_COPY, toggleUniqueString, type PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";
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

export function PRFeedbackChecksFields({
  organizationId,
  factoryId,
  draft,
  onUpdate,
}: {
  organizationId?: string;
  factoryId?: string;
  draft: PRFeedbackDraftSettings;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
}) {
  const catalogEnabled = Boolean(organizationId && factoryId);
  const catalogQuery = useFactoryRepositoryStatusChecks(organizationId ?? "", factoryId ?? "", draft.repository, {
    enabled: catalogEnabled,
  });
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
  draft,
  onUpdate,
}: {
  draft: PRFeedbackDraftSettings;
  onUpdate: <K extends keyof PRFeedbackDraftSettings>(key: K, value: PRFeedbackDraftSettings[K]) => void;
}) {
  return (
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
    const filtered = (integrationsQuery.data ?? []).filter((integration) => {
      const type = integration.metadata?.integrationName?.toLowerCase();
      return type !== "github" && integration.status?.state === "ready" && integration.metadata?.id;
    });
    return sortConnectedIntegrationsByType(filtered);
  }, [integrationsQuery.data]);

  return (
    <section>
      <h3 className="text-sm font-medium text-gray-800 dark:text-gray-100">
        {PR_FEEDBACK_SETTINGS_COPY.integrationsLabel}
      </h3>
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
