import type { Dispatch, SetStateAction } from "react";

import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { CustomAutomationWebhookFields } from "./CustomAutomationWebhookFields";
import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";
import { WebhookIntakeFieldPathInputs } from "./WebhookIntakeFieldPathInputs";
import { WEBHOOK_INTAKE_SETUP_COPY } from "./webhookIntakeSetupCopy";

export function WebhookIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  disabled,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  disabled?: boolean;
}) {
  const layout = useOptionalFactoriesLayout();
  if (sourceId !== "webhook") {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      {organizationId && layout ? (
        <CustomAutomationWebhookFields
          idPrefix="webhook-intake-settings"
          organizationId={organizationId}
          factoryId={layout.factoryId}
          factoryKey={layout.routeSegment}
          webhookId={settings.webhookId}
          usedWebhookIds={[]}
          onChange={(webhookId) => onSettingsChange((current) => ({ ...current, webhookId }))}
          helper={WEBHOOK_INTAKE_SETUP_COPY.existingHelper}
        />
      ) : null}
      <WebhookIntakeFieldPathInputs
        idPrefix="webhook-intake-settings"
        paths={{
          title: settings.webhookTitlePath,
          description: settings.webhookDescriptionPath,
          link: settings.webhookLinkPath,
        }}
        disabled={disabled}
        onChange={(paths) =>
          onSettingsChange((current) => ({
            ...current,
            webhookTitlePath: paths.title,
            webhookDescriptionPath: paths.description,
            webhookLinkPath: paths.link,
          }))
        }
      />
    </div>
  );
}
