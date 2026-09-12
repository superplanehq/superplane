import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ArrowRight } from "lucide-react";

import {
  WORKSPACE_NEXT_STEPS_COPY,
  workspaceNextStepBanner,
  workspaceNextStepsProgressCopy,
  type WorkspaceNextStep,
} from "./workspaceNextStepCatalog";

export function NextStepsPanel({
  steps,
  collapsed = false,
  onContinue,
  onDefer,
}: {
  steps: WorkspaceNextStep[];
  collapsed?: boolean;
  onContinue: (step: WorkspaceNextStep) => void;
  onDefer: () => void;
}) {
  const banner = workspaceNextStepBanner(steps);
  if (!banner || collapsed) {
    return null;
  }

  const progress = workspaceNextStepsProgressCopy(banner.doneCount, banner.totalCount);

  return (
    <section className="workspace-next-steps-reveal px-3 pb-3" data-testid="workspace-next-steps">
      <div
        role="status"
        className="workspace-next-steps-banner flex items-center gap-3 rounded-lg border border-border bg-background px-3.5 py-3 sm:gap-4"
      >
        <WorkspaceNextStepsProgressBadge progress={progress} />
        <div className="min-w-0">
          <h2 className="workspace-next-steps-title text-[18px] font-semibold tracking-[-0.02em] text-foreground">
            {banner.title}
          </h2>
          {banner.description.split("\n").map((line) => (
            <p key={line} className="mt-1.5 text-[13px] text-muted-foreground">
              {line}
            </p>
          ))}
          <div className="mt-3 flex flex-wrap items-center gap-2">
            {banner.canDefer ? (
              <Button type="button" size="sm" variant="ghost" onClick={onDefer} data-testid="workspace-next-step-later">
                {WORKSPACE_NEXT_STEPS_COPY.later}
              </Button>
            ) : null}
            <Button
              type="button"
              size="sm"
              onClick={() => onContinue(banner.activeStep)}
              data-testid={`workspace-next-step-cta-${banner.activeStep.id}`}
            >
              {banner.ctaLabel}
              <ArrowRight className="size-3.5" aria-hidden />
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}

export function WorkspaceNextStepsHeaderBadge({
  progress,
  title,
  onOpen,
}: {
  progress: string;
  title: string;
  onOpen: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={WORKSPACE_NEXT_STEPS_COPY.restoreLabel(title)}
      data-testid="workspace-next-steps-restore"
      className="inline-flex h-8 min-w-0 max-w-[min(32rem,46vw)] cursor-pointer items-center gap-2 rounded-full bg-secondary py-1 pl-1 pr-3 text-secondary-foreground hover:bg-secondary/80"
    >
      <WorkspaceNextStepsProgressBadge progress={progress} compact />
      <span className="workspace-next-steps-title min-w-0 truncate text-[13px] font-medium tracking-[-0.01em]">
        {title}
      </span>
    </button>
  );
}

function WorkspaceNextStepsProgressBadge({ progress, compact = false }: { progress: string; compact?: boolean }) {
  return (
    <Badge
      variant="secondary"
      className={cn(
        "workspace-next-steps-progress shrink-0 rounded-full font-semibold tabular-nums tracking-tight",
        compact ? "h-6 bg-background px-2 text-[12px]" : "px-3 py-1.5 text-[15px]",
      )}
      data-testid={compact ? undefined : "workspace-next-steps-progress"}
    >
      {progress}
    </Badge>
  );
}
