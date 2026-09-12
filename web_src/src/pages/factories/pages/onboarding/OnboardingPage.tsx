import { useConsumeIntegrationSetupReturnOnArrival } from "@/hooks/useConsumeIntegrationSetupReturnOnArrival";
import { Loader2 } from "lucide-react";

import { useFactoriesLayout, type FactoriesLayoutContextValue } from "../../layout/factoriesLayoutContext";
import { FirstRunSetup } from "./FirstRunSetup";
import { FirstRunShell } from "./first-run/FirstRunShell";
import { GithubAppRequiredNotice } from "./GithubAppRequiredNotice";
import { useGithubAppAvailability } from "./useGithubAppAvailability";
import { useOnboardingEntryPath } from "./useOnboardingEntryPath";
import { useOnboardingPageModel } from "./useOnboardingPageModel";
import { useOnboardingWorkspaceResolution } from "./useOnboardingWorkspaceResolution";

export function OnboardingPage() {
  const layout = useFactoriesLayout();
  return <WorkspaceOnboardingPage key={layout.factoryId} layout={layout} />;
}

function WorkspaceOnboardingPage({ layout }: { layout: FactoriesLayoutContextValue }) {
  const onboardingEntryPath = useOnboardingEntryPath();
  const reresolveWorkspace = useOnboardingWorkspaceResolution();
  useConsumeIntegrationSetupReturnOnArrival(layout.organizationId);
  const githubApp = useGithubAppAvailability(layout.organizationId);
  const model = useOnboardingPageModel({ ...layout, onboardingEntryPath, reresolveWorkspace });

  if (!githubApp.resolved) {
    return (
      <FirstRunShell testId="workspace-setup-loading" busy>
        <p className="inline-flex items-center gap-2 text-[13px] text-muted-foreground" role="status">
          <Loader2 className="size-4 animate-spin" aria-hidden />
          Checking GitHub setup…
        </p>
      </FirstRunShell>
    );
  }

  if (githubApp.resolved && !githubApp.failed && !githubApp.available) {
    return <GithubAppRequiredNotice />;
  }

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
