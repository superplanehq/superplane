import { X } from "lucide-react";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FirstRunSetup } from "./FirstRunSetup";
import { OnboardingCancelDialog } from "./OnboardingCancelDialog";
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
    <div className="relative min-h-full w-full bg-background text-foreground" data-testid="workspace-setup">
      <FirstRunSetup model={model} />

      {model.canDeleteWorkspace ? (
        <button
          type="button"
          aria-label="Cancel setup and delete this workspace"
          title="Cancel setup"
          disabled={model.deleting}
          onClick={() => model.setDeleteOpen(true)}
          // The setup screens keep their own controls in the top corners, so
          // the cancel control sits beside the step markers.
          className="fixed bottom-4 left-4 z-20 rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:pointer-events-none disabled:opacity-50"
          data-testid="workspace-setup-cancel"
        >
          <X className="size-6" aria-hidden />
        </button>
      ) : null}

      {model.integrationDialogs}
      <OnboardingCancelDialog
        open={model.deleteOpen}
        canDelete={model.canDeleteWorkspace}
        isDeleting={model.deleting}
        onClose={() => model.setDeleteOpen(false)}
        onConfirm={model.cancelSetup}
      />
    </div>
  );
}
