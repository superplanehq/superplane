import { getApiErrorMessage } from "@/lib/errors";
import { IntegrationsSection } from "@/pages/home/InstallIntegrationsSection";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { SetupSections } from "./OnboardingWireframe";
import { useOnboardingPageModel } from "./useOnboardingPageModel";

export function OnboardingPage() {
  const layout = useFactoriesLayout();
  const model = useOnboardingPageModel(layout);

  if (!model.canConfigureWorkspace) {
    return (
      <div className="flex min-h-full items-center justify-center bg-background px-6 text-foreground">
        <div className="max-w-md rounded-lg border border-border bg-card p-8 text-center">
          <h1 className="text-[16px] font-semibold">An organization admin must finish setup</h1>
          <p className="mt-2 text-[13px] text-muted-foreground">
            Ask an organization admin to connect the integrations and configure this workspace.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full w-full bg-background text-foreground" data-testid="workspace-onboarding">
      <div className="mx-auto w-full max-w-3xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>
        <p className="mt-1.5 max-w-2xl text-[13px] text-muted-foreground">
          Connect your repositories and coding agent. SuperPlane will create an app and a line for work orders.
        </p>

        {model.repositoriesLoading ? (
          <p className="mt-6 text-[13px] text-muted-foreground">Loading repositories…</p>
        ) : null}
        {model.repositoriesError ? (
          <p className="mt-6 text-[13px] text-destructive">
            {getApiErrorMessage(model.repositoriesError, "Failed to load repositories")}
          </p>
        ) : null}

        <div className="mt-8">
          <SetupSections
            setup={model.setup}
            openSection={model.openSection}
            setOpenSection={model.setOpenSection}
            requestConnect={() => undefined}
            onContinueName={model.saveName}
            onContinueRepo={model.saveRepository}
            onContinueIssues={model.saveIssues}
            onFinish={model.finish}
            inviteUrl={model.inviteUrl}
            inviteLoading={model.inviteLoading}
            canInvite={model.canInvite}
            repos={model.repositories}
            vcsIntegrationControls={
              <IntegrationsSection
                integrations={["github"]}
                organizationId={layout.organizationId}
                selections={model.selections}
                onSelectionsChange={(next) => model.updateSelection("github", next)}
              />
            }
            agentIntegrationControls={
              <IntegrationsSection
                integrations={["claude"]}
                organizationId={layout.organizationId}
                selections={model.selections}
                onSelectionsChange={(next) => model.updateSelection("claude", next)}
              />
            }
            showExternalTrackers={false}
            allowIssueSkip={false}
            saving={model.saving}
          />
        </div>
      </div>
    </div>
  );
}
