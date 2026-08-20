import { AlertCircle, Loader2 } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factorySetupPath } from "../lib/factoryPagePaths";
import { VELOCITY_PERIOD_OPTIONS } from "../lib/factoryVelocityFlow";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";
import { VelocityLoadedView, type VelocityPeriodDays, type VelocitySourceSplitConfig } from "./VelocityLoadedView";
import { useVelocityPageModel, type VelocityPageModel } from "./useVelocityPageModel";

const CARD_CLASSES =
  "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card px-6 py-16 text-center";

export function VelocityPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const model = useVelocityPageModel(organizationId, factoryId, factory?.onboarding);
  usePageTitle(["Velocity", factory?.name ?? "Workspace"]);

  const header = <VelocityHeader model={model} />;

  if (model.velocity.error && !model.velocity.data) {
    return renderShell(header, <VelocityErrorState onRetry={model.velocity.refetch} />);
  }

  if (model.velocity.isLoading || !model.velocity.data) {
    return renderShell(header, <VelocityLoadingState />);
  }

  const sourceSplit: VelocitySourceSplitConfig = {
    hasPeopleCohort: model.velocity.hasPeopleCohort,
    repositoryLabel: model.velocity.repositoryLabel,
    emptyState: renderSourceSplitEmptyState({
      organizationId,
      factoryKey,
      hasRepoConfigured: model.hasRepoConfigured,
      peopleSearchFailed: model.velocity.peopleSearchFailed,
    }),
  };

  return renderShell(
    header,
    <VelocityLoadedView
      periodLabel={model.periodLabel}
      periodDays={model.periodDays}
      data={model.velocity.data}
      sourceSplit={sourceSplit}
      workOrderFlow={
        model.workOrderFlow.isLoading
          ? undefined
          : {
              flow: model.workOrderFlow.flow,
              emptyLabel: model.workOrderFlow.error ? "We could not load work order time." : undefined,
            }
      }
    />,
    "factory-velocity-page",
  );
}

function renderShell(header: ReactNode, body: ReactNode, bodyTestId?: string) {
  return (
    <>
      {header}
      <div className={cn(factorySectionBodyClassName, "space-y-6")} data-testid={bodyTestId}>
        {body}
      </div>
    </>
  );
}

function VelocityHeader({ model }: { model: VelocityPageModel }) {
  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      title="Velocity"
      subtitle="Merged pull requests and work order time."
      actions={
        <SegmentedNav
          ariaLabel="Velocity period in days"
          size="xs"
          value={String(model.periodDays)}
          onValueChange={(value) => {
            const next = Number(value);
            if (next === 7 || next === 30) model.setPeriodDays(next as VelocityPeriodDays);
          }}
          options={VELOCITY_PERIOD_OPTIONS}
        />
      }
    />
  );
}

interface SourceSplitEmptyStateArgs {
  organizationId: string;
  factoryKey: string;
  hasRepoConfigured: boolean;
  peopleSearchFailed: boolean;
}

function renderSourceSplitEmptyState({
  organizationId,
  factoryKey,
  hasRepoConfigured,
  peopleSearchFailed,
}: SourceSplitEmptyStateArgs) {
  if (!hasRepoConfigured) {
    return (
      <p className="text-[13px] text-muted-foreground">
        Connect a GitHub repository in{" "}
        <Link
          to={factorySetupPath(organizationId, factoryKey)}
          className="underline underline-offset-2 hover:text-foreground"
        >
          workspace setup
        </Link>{" "}
        to compare People and SuperPlane.
      </p>
    );
  }
  if (peopleSearchFailed) {
    return (
      <p className="text-[13px] text-muted-foreground">
        We could not load People merges. SuperPlane counts still show.
      </p>
    );
  }
  return (
    <p className="text-[13px] text-muted-foreground">No merged pull requests in this repository for the period.</p>
  );
}

function VelocityLoadingState() {
  return (
    <div className={CARD_CLASSES} data-testid="velocity-loading-state">
      <Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />
      <p className="text-[13px] text-muted-foreground">Loading velocity…</p>
    </div>
  );
}

function VelocityErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className={CARD_CLASSES} data-testid="velocity-error-state">
      <AlertCircle className="size-5 text-destructive" aria-hidden />
      <div className="space-y-1">
        <p className="text-[13px] font-medium text-foreground">We could not load velocity.</p>
        <p className="text-[12px] text-muted-foreground">Check your network and try again.</p>
      </div>
      <Button type="button" variant="outline" size="sm" onClick={onRetry} data-testid="velocity-error-retry">
        Retry
      </Button>
    </div>
  );
}
