import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useId } from "react";

import { AgentAdditionalInstructions } from "./AgentAdditionalInstructions";
import { AgentRunnerSettings } from "./AgentRunnerSettings";
import { ImplementDurationDraftProvider } from "./ImplementDurationDraft";
import { ImplementSettingRow } from "./ImplementSettingRow";
import { findImplementPullRequest, useImplementSettingsDraft } from "./implementSettingsDraft";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";
import type { PlanningReviewDraft } from "./planningReviewMockup";

type SaveSettings = (update: NodeConfigurationSave) => Promise<void> | void;

type ImplementSettingsController = {
  agentDraft: PlanningReviewDraft;
  pullRequest?: SuperplaneComponentsNode;
  durationMinutes: string;
  durationInvalid: boolean;
  saving: boolean;
  saveError?: string;
  onAgentDraftChange: (draft: PlanningReviewDraft) => void;
  onPullRequestDraftChange: (draft: boolean) => void;
  onDurationStateChange: (state: { minutes: string; invalid: boolean }) => void;
  onSave: () => void;
};

export function ImplementConfiguration({
  agent,
  nodes,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
  onClose,
  controller,
}: {
  agent: SuperplaneComponentsNode;
  nodes?: SuperplaneComponentsNode[];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onSave?: SaveSettings;
  onClose?: () => void;
  controller?: ImplementSettingsController;
}) {
  if (controller) {
    return (
      <ImplementConfigurationView
        agent={agent}
        agentDraft={controller.agentDraft}
        pullRequest={controller.pullRequest}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        durationMinutes={controller.durationMinutes}
        durationInvalid={controller.durationInvalid}
        saving={controller.saving}
        saveError={controller.saveError}
        onAgentDraftChange={controller.onAgentDraftChange}
        onPullRequestDraftChange={controller.onPullRequestDraftChange}
        onDurationStateChange={controller.onDurationStateChange}
        onSave={controller.onSave}
      />
    );
  }
  return (
    <OwnedImplementConfiguration
      agent={agent}
      nodes={nodes ?? []}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      onSave={onSave ?? (() => undefined)}
      onClose={onClose}
    />
  );
}

function OwnedImplementConfiguration({
  agent,
  nodes,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
  onClose,
}: {
  agent: SuperplaneComponentsNode;
  nodes: SuperplaneComponentsNode[];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onSave: SaveSettings;
  onClose?: () => void;
}) {
  const settings = useImplementSettingsDraft([agent, ...nodes]);
  if (!settings.draft) return null;
  return (
    <ImplementConfigurationView
      agent={settings.draft.agentBase}
      agentDraft={settings.draft.agentDraft}
      pullRequest={settings.draft.pullRequestNode ?? findImplementPullRequest(nodes)}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      durationMinutes={settings.draft.durationMinutes}
      durationInvalid={settings.draft.durationInvalid}
      saving={settings.saving}
      saveError={settings.saveError}
      onAgentDraftChange={settings.updateAgentDraft}
      onPullRequestDraftChange={settings.updatePullRequestDraft}
      onDurationStateChange={settings.updateDuration}
      onSave={() => void settings.save(onSave, onClose ?? (() => undefined))}
    />
  );
}

function ImplementConfigurationView({
  agent,
  agentDraft,
  pullRequest,
  organizationId,
  factoryId,
  factoryKey,
  durationMinutes,
  durationInvalid,
  saving,
  saveError,
  onAgentDraftChange,
  onPullRequestDraftChange,
  onDurationStateChange,
  onSave,
}: {
  agent: SuperplaneComponentsNode;
  agentDraft: PlanningReviewDraft;
  pullRequest?: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  durationMinutes: string;
  durationInvalid: boolean;
  saving: boolean;
  saveError?: string;
  onAgentDraftChange: (draft: PlanningReviewDraft) => void;
  onPullRequestDraftChange: (draft: boolean) => void;
  onDurationStateChange: (state: { minutes: string; invalid: boolean }) => void;
  onSave: () => void;
}) {
  return (
    <ImplementDurationDraftProvider
      minutes={durationMinutes}
      invalid={durationInvalid}
      onChange={onDurationStateChange}
    >
      <div className="flex min-h-0 flex-1 flex-col bg-background">
        <section
          aria-label="Implement configuration"
          inert={saving || undefined}
          className="min-h-0 flex-1 overflow-y-auto"
        >
          <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8 sm:px-10">
            <AgentRunnerSettings
              key={agent.id}
              node={agent}
              draft={agentDraft}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              onSave={() => undefined}
              onDraftChange={onAgentDraftChange}
              view="implementation"
              disabled={saving}
            />
            {pullRequest ? (
              <PullRequestSettings node={pullRequest} disabled={saving} onDraftChange={onPullRequestDraftChange} />
            ) : null}
            <AgentAdditionalInstructions
              draft={agentDraft}
              disabled={saving}
              onChange={onAgentDraftChange}
              description="Add requirements for coding, testing, or pull request descriptions. Applies to future runs."
              placeholder="For example: Avoid new production dependencies unless the task requires them."
            />
          </div>
        </section>
        <ImplementSettingsFooter
          saving={saving}
          saveError={saveError}
          disabled={saving || durationInvalid}
          onSave={onSave}
        />
      </div>
    </ImplementDurationDraftProvider>
  );
}

function ImplementSettingsFooter({
  saving,
  saveError,
  disabled,
  onSave,
}: {
  saving: boolean;
  saveError?: string;
  disabled: boolean;
  onSave: () => void;
}) {
  return (
    <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
      {saveError ? (
        <p className="workspace-body-text mr-auto text-destructive" role="alert">
          {saveError}
        </p>
      ) : null}
      <Button type="button" disabled={disabled} onClick={onSave} data-testid="implement-settings-save">
        {saving ? "Saving" : "Save"}
      </Button>
    </footer>
  );
}

function PullRequestSettings({
  node,
  disabled = false,
  onDraftChange,
}: {
  node: SuperplaneComponentsNode;
  disabled?: boolean;
  onDraftChange: (draft: boolean) => void;
}) {
  const id = useId();
  return (
    <ImplementSettingRow
      label="Create draft pull requests"
      description="Pull requests stay in draft until you mark them ready."
      htmlFor={id}
    >
      <Switch
        id={id}
        aria-describedby={`${id}-help`}
        checked={node.configuration?.draft === true}
        disabled={disabled}
        onCheckedChange={onDraftChange}
      />
    </ImplementSettingRow>
  );
}
