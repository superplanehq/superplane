import { usePageTitle } from "@/hooks/usePageTitle";
import { Button } from "@/components/ui/button";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { ModelSourceBody } from "./llmModelSourceBody";
import { ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";
import { useWorkspaceModelSettings } from "./useWorkspaceModelSettings";
import { workspaceModelSourceLabel } from "./workspaceModelSource";
import { WorkspaceModelSourceDialog } from "./workspaceModelSourceSwitch";
import { WorkspaceCodexMembers } from "./WorkspaceCodexMembers";

export function FactorySettingsOrganizationLLMModelsPage() {
  const settings = useWorkspaceModelSettings();
  usePageTitle([COPY.pageTitle, "Settings", settings.factory.name ?? "Workspace"]);
  return (
    <FactorySettingsPageFrame title={COPY.pageTitle} subtitle={COPY.pageSubtitle}>
      {settings.canRead ? (
        <WorkspaceModelSettings settings={settings} />
      ) : (
        <p role="status" className="text-[13px] text-muted-foreground">
          {settings.permissionsLoading
            ? "Loading permissions..."
            : "You do not have permission to view workspace models."}
        </p>
      )}
      <WorkspaceModelSourceDialog settings={settings} />
    </FactorySettingsPageFrame>
  );
}

function WorkspaceModelSettings({ settings }: { settings: ReturnType<typeof useWorkspaceModelSettings> }) {
  const sourceLabel = workspaceModelSourceLabel(settings.source);
  return (
    <div className="flex flex-col gap-4" data-testid="factory-settings-llm-models">
      <FactorySettingsCard>
        <div className="flex items-center justify-between gap-4">
          <div className="min-w-0 space-y-1">
            <p className="text-xs text-muted-foreground">Model source</p>
            <p role="status" aria-label={`${sourceLabel} is active`} className="text-sm font-medium">
              {sourceLabel}
            </p>
          </div>
          <Button
            ref={settings.changeSourceButtonRef}
            type="button"
            variant="outline"
            size="sm"
            className="shrink-0"
            aria-label="Change model source"
            disabled={!settings.canUpdate || settings.switcher.saving}
            onClick={() => settings.switcher.openDialog()}
          >
            Change
          </Button>
        </div>
      </FactorySettingsCard>
      <FactorySettingsCard data-testid="llm-models-list">
        {settings.switcher.switchedNotice ? (
          <p role="status" className="mb-4 text-xs text-muted-foreground">
            {settings.switcher.switchedNotice}
          </p>
        ) : null}
        <ModelSourceBody
          {...settings}
          canUpdate={
            settings.canUpdate &&
            !settings.switcher.saving &&
            (settings.source !== "subscription" || settings.subscription.offered)
          }
        />
      </FactorySettingsCard>
      {settings.source === "subscription" ? (
        <FactorySettingsCard>
          <WorkspaceCodexMembers
            organizationId={settings.organizationId}
            factoryId={settings.factoryId}
            canUpdate={settings.canUpdate && !settings.switcher.saving && settings.subscription.offered}
            onChangeDefaultOwner={() => settings.switcher.openDialog("subscription")}
          />
        </FactorySettingsCard>
      ) : null}
    </div>
  );
}
