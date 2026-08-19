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
        {model.repositoriesLoading ? (
          <p className="mb-6 text-[13px] text-muted-foreground">Loading repositories…</p>
        ) : null}
        {model.repositoriesError ? (
          <p className="mb-6 text-[13px] text-destructive">
            {getApiErrorMessage(model.repositoriesError, "Failed to load repositories")}
          </p>
        ) : null}

        <div>
          <SetupSections
            setup={model.setup}
            openSection={model.openSection}
            setOpenSection={model.setOpenSection}
            requestConnect={model.requestConnect}
            requestConfigure={model.requestConfigure}
            onContinueRepo={model.saveRepository}
            onContinueIssues={model.saveIssues}
            onFinish={model.finish}
            repos={model.repositories}
            saving={model.saving}
          />
        </div>
      </div>
      {model.integrationDialogs}
    </div>
  );
}
