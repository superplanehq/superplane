import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { Link } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId } from "../../lib/factoryPagePaths";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

/**
 * Storybook-only post-setup checklist (v3 parity).
 * Shown on overview when tips are active for the current workspace.
 */
export function GettingStartedWireframe({ onDismiss }: { onDismiss?: () => void }) {
  const { organizationId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();

  function handleCreateWorkOrder() {
    onboarding?.clearOverviewTips();
    onDismiss?.();
    openCreateWorkOrder();
  }

  return (
    <div className="mx-auto w-full max-w-[720px] px-8 py-8" data-testid="onboarding-getting-started">
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Get started</h1>
          <p className="mt-1.5 text-[13px] text-muted-foreground">
            The app repository is connected. Hand off the first small task to your coding agent.
          </p>
        </div>
        <div className="shrink-0 text-[12px] text-muted-foreground">1 of 3 complete</div>
      </div>

      <div className="mt-3 h-1 overflow-hidden rounded-full bg-accent">
        <div className="h-full w-1/3 rounded-full bg-foreground/80" />
      </div>

      <div className="mt-6 overflow-hidden rounded-lg border border-border bg-background">
        <div className="flex items-start gap-3 border-b border-border px-4 py-4">
          <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-600 text-white">
            <Check className="size-3" strokeWidth={2.5} aria-hidden />
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium tracking-[-0.01em] text-muted-foreground">
              Connect app repository
            </div>
            <p className="mt-0.5 text-[13px] text-muted-foreground/80">
              Version control is linked. Agents can analyze the app, change code, and open pull requests.
            </p>
          </div>
        </div>

        <div className="flex items-start gap-3 border-b border-border bg-accent/30 px-4 py-4">
          <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border border-foreground bg-background text-[11px] font-medium text-foreground">
            2
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Create your first task</div>
            <p className="mt-0.5 text-[13px] text-muted-foreground">
              A task is one unit of work for the coding agent. Create one manually and watch the agent open a pull
              request.
            </p>
            <Button type="button" size="sm" className="mt-3" onClick={() => handleCreateWorkOrder()}>
              Create task
            </Button>
          </div>
        </div>

        <div className="flex items-start gap-3 px-4 py-4">
          <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border border-border bg-background text-[11px] font-medium text-muted-foreground">
            3
          </span>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">
              See the agent pipeline on a line
            </div>
            <p className="mt-0.5 text-[13px] text-muted-foreground">
              Each task runs through a line: intake, build, verify, and related phases you can configure later.
            </p>
            <Link
              to={factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory))}
              onClick={onDismiss}
              className={cn(
                "mt-3 inline-flex h-8 items-center rounded-md border border-border px-3 text-[13px]",
                "text-foreground transition-colors hover:bg-accent",
              )}
            >
              Open board
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
