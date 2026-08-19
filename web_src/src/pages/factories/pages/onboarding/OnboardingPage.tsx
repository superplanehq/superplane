import { getApiErrorMessage } from "@/lib/errors";

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
    <div className="min-h-full w-full bg-background text-foreground" data-testid="workspace-setup">
      <div className="mx-auto w-full max-w-3xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>
        <p className="mt-1.5 max-w-2xl text-[13px] text-muted-foreground">
          Hand off small engineering work to AI. SuperPlane finds candidate work in your app and backlog, then a coding
          agent opens pull requests. Finish one section to unlock the next.
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
            requestConnect={model.requestConnect}
            onContinueName={model.saveName}
            onContinueRepo={model.saveRepository}
            onContinueIssues={model.saveIssues}
            onFinish={model.finish}
            inviteUrl={model.inviteUrl}
            inviteLoading={model.inviteLoading}
            canInvite={model.canInvite}
            repos={model.repositories}
            saving={model.saving}
          />
        </div>
      </div>
      {model.integrationDialogs}
    </div>
  );
}
