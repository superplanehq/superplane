import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import { useState, type Dispatch, type SetStateAction } from "react";

import { DependabotIntakeFilterFields } from "./DependabotIntakeFilterFields";
import { INTAKE_SETTINGS_COPY, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { NodeConfigRepositoryLine } from "./NodeConfigPanel";

export function DependabotIntakeConfigBody({
  node,
  fallbackRepository,
  settings,
  savePending = false,
  saveError,
  onSettingsChange,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  fallbackRepository?: string;
  settings: IntakeSourceSettings;
  savePending?: boolean;
  saveError?: string;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
}) {
  const [submitError, setSubmitError] = useState<string>();
  const visibleError = saveError || submitError;

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="dependabot-intake-config">
      <div className="min-h-0 flex-1 overflow-y-auto px-10 py-6">
        <div className="space-y-8">
          <NodeConfigRepositoryLine value={dependabotRepositoryValue(node, fallbackRepository)} />
          <DependabotIntakeFilterFields
            sourceId="dependabot-alerts"
            settings={settings}
            onSettingsChange={onSettingsChange}
          />
          {visibleError ? (
            <p
              className="workspace-body-text text-destructive"
              role="alert"
              data-testid="dependabot-intake-config-save-error"
            >
              {visibleError}
            </p>
          ) : null}
          <div className="flex justify-end">
            <Button
              type="button"
              disabled={savePending}
              data-testid="dependabot-intake-config-save"
              onClick={() => {
                void saveDraft(settings, onSave, setSubmitError);
              }}
            >
              {savePending ? INTAKE_SETTINGS_COPY.saving : INTAKE_SETTINGS_COPY.save}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

async function saveDraft(
  settings: IntakeSourceSettings,
  onSave: (next: IntakeSourceSettings) => Promise<void> | void,
  setSubmitError: (message: string | undefined) => void,
) {
  setSubmitError(undefined);
  try {
    await onSave(settings);
  } catch (error) {
    setSubmitError(getApiErrorMessage(error, INTAKE_SETTINGS_COPY.saveError));
  }
}

function dependabotRepositoryValue(node: SuperplaneComponentsNode, fallback: string | undefined): unknown {
  const configured = node.configuration?.repository;
  if (typeof configured === "string" && configured.trim() !== "") {
    return configured;
  }
  return fallback;
}
