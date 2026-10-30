import { Tabs, TabsContent } from "@/components/ui/tabs";
import { usePageTitle } from "@/hooks/usePageTitle";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { ModelSourceBody } from "./llmModelSourceBody";
import { ORGANIZATION_LLM_MODELS_COPY as COPY } from "./organizationLLMModelsCopy";
import { useWorkspaceModelSettings } from "./useWorkspaceModelSettings";
import { WorkspaceModelSourceSelector } from "./WorkspaceModelSourceSelector";
import { ModelSourceSwitchDialogs } from "./workspaceModelSourceSwitch";

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
      <ModelSourceSwitchDialogs
        {...settings}
        providers={settings.providers.filter(
          (provider) => settings.canCreateIntegration || settings.connected.includes(provider),
        )}
      />
    </FactorySettingsPageFrame>
  );
}

function WorkspaceModelSettings({ settings }: { settings: ReturnType<typeof useWorkspaceModelSettings> }) {
  return (
    <Tabs
      value={settings.source}
      onValueChange={settings.chooseSource}
      activationMode="manual"
      className="gap-4"
      data-testid="factory-settings-llm-models"
    >
      <WorkspaceModelSourceSelector settings={settings} />
      <TabsContent value={settings.source}>
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
            codex={settings.subscription.codex}
            onChangeDefaultOwner={() => settings.switcher.setSubscriptionDialog("warn")}
            onChooseProvider={(target) => {
              if (target !== settings.currentProvider) settings.switcher.setDialog({ target, step: "warn" });
            }}
          />
        </FactorySettingsCard>
      </TabsContent>
    </Tabs>
  );
}
