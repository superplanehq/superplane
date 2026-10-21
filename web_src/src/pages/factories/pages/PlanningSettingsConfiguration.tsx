import { Button } from "@/components/ui/button";
import { AgentAdditionalInstructions } from "./AgentAdditionalInstructions";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import { PlanningSettingsFields } from "./PlanningSettingsFields";
import {
  PLANNING_SETTINGS_COPY,
  type PlanningAutoStartLine,
  type PlanningDraftSettings,
} from "./planningSettingsModel";

export function PlanningSettingsConfiguration({
  draft,
  lines,
  agent,
  agentDraft,
  onAgentChange,
  onUpdate,
  onRetry,
  saving,
}: {
  draft: PlanningDraftSettings;
  lines: PlanningAutoStartLine[];
  agent?: PlanningReviewAgentSlot;
  agentDraft?: PlanningReviewDraft;
  onAgentChange: (next: PlanningReviewDraft) => void;
  onUpdate: <K extends keyof PlanningDraftSettings>(key: K, value: PlanningDraftSettings[K]) => void;
  onRetry?: () => void;
  saving: boolean;
}) {
  return (
    <section aria-label="Planning configuration" inert={saving || undefined} className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8 sm:px-10">
        {!draft.enabled ? (
          <div
            role="status"
            data-testid="planning-settings-paused"
            className="rounded-lg border border-border bg-muted/50 px-4 py-3"
          >
            <p className="text-sm font-medium">Planning paused</p>
            <p className="mt-1 text-xs text-muted-foreground">
              New tasks do not start planning conversations. Existing conversations continue.
            </p>
          </div>
        ) : null}
        {agent?.isLoading ? (
          <p role="status" className="text-sm text-muted-foreground">
            {PLANNING_SETTINGS_COPY.loading}
          </p>
        ) : agentDraft?.components.length ? (
          <PlanningReviewForm
            draft={agentDraft}
            onChange={onAgentChange}
            organizationId={agent?.organizationId}
            factoryId={agent?.factoryId}
            appearance="fields"
            view="planning"
            showResources={false}
            disabled={saving}
          />
        ) : agent ? (
          <div className="flex items-center gap-3">
            <p role="alert" className="text-sm text-destructive">
              {PLANNING_SETTINGS_COPY.loadError}
            </p>
            {onRetry ? (
              <Button variant="outline" size="sm" onClick={onRetry}>
                {PLANNING_SETTINGS_COPY.retry}
              </Button>
            ) : null}
          </div>
        ) : null}
        <PlanningSettingsFields draft={draft} lines={lines} onUpdate={onUpdate} />
        {!agent?.isLoading && agentDraft?.components.length ? (
          <AgentAdditionalInstructions
            draft={agentDraft}
            onChange={onAgentChange}
            disabled={saving}
            description="Add requirements for new planning conversations. SuperPlane keeps the planning and scoring rules."
            placeholder="For example: Changes to billing must support customers on legacy annual plans."
          />
        ) : null}
      </div>
    </section>
  );
}

export function PlanningSettingsFooter({
  error,
  saving,
  disabled,
  onSave,
}: {
  error?: string;
  saving: boolean;
  disabled: boolean;
  onSave: () => void;
}) {
  return (
    <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
      {error ? (
        <p className="workspace-body-text mr-auto text-destructive" role="alert">
          {error}
        </p>
      ) : null}
      <Button type="button" disabled={disabled} onClick={onSave} data-testid="planning-settings-save">
        {saving ? PLANNING_SETTINGS_COPY.saving : PLANNING_SETTINGS_COPY.save}
      </Button>
    </footer>
  );
}
