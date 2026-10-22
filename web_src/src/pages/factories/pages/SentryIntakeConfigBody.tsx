import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import { useState, type Dispatch, type SetStateAction } from "react";

import { INTAKE_SETTINGS_COPY, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";
import { SentryIntakeFilterFields } from "./SentryIntakeFilterFields";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

export function SentryIntakeConfigBody({
  settings,
  organizationId,
  integrationId,
  savePending = false,
  saveError,
  onSettingsChange,
  onSave,
}: {
  settings: IntakeSourceSettings;
  organizationId: string;
  integrationId?: string;
  savePending?: boolean;
  saveError?: string;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
}) {
  const [submitError, setSubmitError] = useState<string>();
  const projectsMissing = settings.sentryProjectIds.length === 0;
  const visibleError =
    saveError || submitError || (projectsMissing ? SENTRY_INTAKE_SETUP_COPY.projectsRequired : undefined);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="sentry-intake-config">
      <div className="min-h-0 flex-1 overflow-y-auto px-10 py-6">
        <div className="space-y-8">
          <IntakeDefaultOwnerField
            organizationId={organizationId}
            value={settings.defaultOwnerId}
            disabled={savePending}
            onChange={(defaultOwnerId) => onSettingsChange((current) => ({ ...current, defaultOwnerId }))}
          />
          <SentryIntakeFilterFields
            sourceId="sentry-exceptions"
            settings={settings}
            onSettingsChange={onSettingsChange}
            organizationId={organizationId}
            integrationId={integrationId}
          />
          {visibleError ? (
            <p
              className="workspace-body-text text-destructive"
              role="alert"
              data-testid="sentry-intake-config-save-error"
            >
              {visibleError}
            </p>
          ) : null}
          <div className="flex justify-end">
            <Button
              type="button"
              disabled={savePending || projectsMissing}
              data-testid="sentry-intake-config-save"
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
