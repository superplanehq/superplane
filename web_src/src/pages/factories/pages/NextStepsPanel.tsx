import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import {
  workspaceNextStepBanner,
  workspaceNextStepsProgressCopy,
  type WorkspaceNextStep,
} from "./workspaceNextStepCatalog";

export function NextStepsPanel({
  steps,
  onContinue,
}: {
  steps: WorkspaceNextStep[];
  onContinue: (step: WorkspaceNextStep) => void;
}) {
  const banner = workspaceNextStepBanner(steps);
  if (!banner) {
    return null;
  }

  const progress = workspaceNextStepsProgressCopy(banner.doneCount, banner.totalCount);

  return (
    <section className="px-3 pb-3" data-testid="workspace-next-steps">
      <div
        role="status"
        className="flex flex-col gap-3 rounded-lg border border-border bg-background px-3.5 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-6"
      >
        <div className="flex min-w-0 items-start gap-3 sm:items-center sm:gap-4">
          <Badge
            variant="secondary"
            className="shrink-0 rounded-full px-3 py-1.5 text-[15px] font-semibold tabular-nums tracking-tight"
            data-testid="workspace-next-steps-progress"
          >
            {progress}
          </Badge>
          <div className="min-w-0">
            <h2 className="text-[18px] font-semibold tracking-[-0.02em] text-foreground">{banner.title}</h2>
            <p className="mt-1.5 text-[13px] text-muted-foreground">{banner.worksCopy}</p>
            <p className="mt-1 text-[13px] text-muted-foreground">{banner.missingCopy}</p>
          </div>
        </div>
        <Button
          type="button"
          className="shrink-0 self-start sm:self-center"
          onClick={() => onContinue(banner.activeStep)}
          data-testid={`workspace-next-step-cta-${banner.activeStep.id}`}
        >
          {banner.ctaLabel}
        </Button>
      </div>
    </section>
  );
}
