import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { ChevronLeft } from "lucide-react";

import type { WizardStepId } from "./onboardingFixtures";

const CONTINUE_LABELS: Record<WizardStepId, string> = {
  vcs: "Next",
  repo: "Next",
  issues: "Next",
  agent: "Next",
  name: "Finish setup",
};

// The Name step provisions the apps and the line, so it runs long enough that
// the button must say what it does.
const CONTINUE_LOADING_LABELS: Partial<Record<WizardStepId, string>> = {
  name: "Finishing setup...",
};

export function WizardStepFooter({
  step,
  advanceEnabled,
  saving,
  showBack,
  onBack,
  onNext,
}: {
  step: WizardStepId;
  advanceEnabled: boolean;
  saving: boolean;
  showBack: boolean;
  onBack: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-4 py-3">
      {showBack ? (
        <Button type="button" size="sm" variant="ghost" disabled={saving} onClick={onBack}>
          <ChevronLeft className="size-3.5" aria-hidden />
          Back
        </Button>
      ) : (
        <span />
      )}
      <LoadingButton
        type="button"
        size="sm"
        disabled={!advanceEnabled}
        loading={saving}
        loadingText={CONTINUE_LOADING_LABELS[step]}
        onClick={onNext}
        data-testid="workspace-setup-continue"
      >
        {CONTINUE_LABELS[step]}
      </LoadingButton>
    </div>
  );
}
