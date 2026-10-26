import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { AgentRunnerSettings } from "./AgentRunnerSettings";
import { ImplementDurationDraftProvider } from "./ImplementDurationDraft";
import { MergeConfidenceCheckList } from "./MergeConfidenceCheckList";
import { MERGE_CONFIDENCE_SETTINGS_COPY } from "./mergeConfidenceCopy";
import { useMergeConfidenceSettingsDraft } from "./mergeConfidenceSettingsDraft";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";

type ConfigurationProps = {
  agent: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  canvasId?: string;
  onSave: (update: NodeConfigurationSave) => Promise<void> | void;
  onClose: () => void;
};

export function MergeConfidenceConfiguration(
  props: ConfigurationProps & { settings?: ReturnType<typeof useMergeConfidenceSettingsDraft> },
) {
  if (props.settings) return <MergeConfidenceConfigurationView {...props} settings={props.settings} />;
  return <OwnedMergeConfidenceConfiguration {...props} />;
}

function OwnedMergeConfidenceConfiguration(props: ConfigurationProps) {
  const settings = useMergeConfidenceSettingsDraft(props.agent, props);
  return <MergeConfidenceConfigurationView {...props} settings={settings} />;
}

function MergeConfidenceConfigurationView({
  settings,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
  onClose,
}: ConfigurationProps & { settings: ReturnType<typeof useMergeConfidenceSettingsDraft> }) {
  const { draft, saving, saveError } = settings;
  if (!draft) return null;
  return (
    <ImplementDurationDraftProvider
      minutes={draft.durationMinutes}
      invalid={draft.durationInvalid}
      onChange={settings.updateDuration}
    >
      <div className="flex min-h-0 flex-1 flex-col bg-background">
        <section
          aria-label="Merge confidence configuration"
          inert={saving || undefined}
          className="min-h-0 flex-1 overflow-y-auto"
          data-testid="merge-confidence-settings"
        >
          <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8 sm:px-10">
            <AgentRunnerSettings
              node={draft.agentBase}
              draft={draft.agentDraft}
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              onSave={() => undefined}
              onDraftChange={settings.updateAgentDraft}
              view="implementation"
              disabled={saving}
            />
            <section className="space-y-3">
              <header>
                <h2 className="text-[15px] font-semibold text-foreground">
                  {MERGE_CONFIDENCE_SETTINGS_COPY.checksLabel}
                </h2>
                <p className="workspace-body-text mt-1 text-muted-foreground">
                  {MERGE_CONFIDENCE_SETTINGS_COPY.checksHelper}
                </p>
              </header>
              <MergeConfidenceCheckList
                checks={draft.checks}
                onToggle={settings.updateChecks}
                idPrefix="merge-confidence-check"
                disabled={saving}
              />
            </section>
          </div>
        </section>
        <footer className="flex shrink-0 items-center justify-end gap-3 border-t border-border px-5 py-3">
          {saveError ? (
            <p className="workspace-body-text mr-auto text-destructive" role="alert">
              {saveError}
            </p>
          ) : null}
          <Button
            type="button"
            disabled={saving || draft.durationInvalid}
            onClick={() => void settings.save(onSave, onClose)}
            data-testid="merge-confidence-settings-save"
          >
            {saving ? MERGE_CONFIDENCE_SETTINGS_COPY.saving : MERGE_CONFIDENCE_SETTINGS_COPY.save}
          </Button>
        </footer>
      </div>
    </ImplementDurationDraftProvider>
  );
}
