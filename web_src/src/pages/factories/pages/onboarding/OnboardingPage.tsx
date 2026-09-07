import { useConsumeIntegrationSetupReturnOnArrival } from "@/hooks/useConsumeIntegrationSetupReturnOnArrival";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingEntryPath } from "./useOnboardingEntryPath";
import { useOnboardingPageModel } from "./useOnboardingPageModel";
import { useOnboardingWorkspaceResolution } from "./useOnboardingWorkspaceResolution";

export function OnboardingPage() {
  const layout = useFactoriesLayout();
  const onboardingEntryPath = useOnboardingEntryPath();
  const reresolveWorkspace = useOnboardingWorkspaceResolution();
  useConsumeIntegrationSetupReturnOnArrival(layout.organizationId);
  const model = useOnboardingPageModel({ ...layout, onboardingEntryPath, reresolveWorkspace });

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
    <div className="relative min-h-full w-full bg-background text-foreground" data-testid="workspace-setup">
      <FirstRunSetup model={model} />
      {model.integrationDialogs}
    </div>
  );
}
