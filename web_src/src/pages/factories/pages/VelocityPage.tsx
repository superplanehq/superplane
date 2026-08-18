import { AlertCircle, Loader2 } from "lucide-react";
import type { ReactNode } from "react";

import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { VELOCITY_PERIOD_OPTIONS } from "../lib/factoryVelocityFlow";
import { factorySectionBodyClassName, factorySectionHeaderClassName } from "./factoryPageLayoutStyles";
import { VelocityLoadedView, type VelocityPeriodDays, type VelocitySourceSplitConfig } from "./VelocityLoadedView";
import { useVelocityPageModel, type VelocityPageModel } from "./useVelocityPageModel";

const CARD_CLASSES =
  "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card px-6 py-16 text-center";

const NO_REPO_SENTINEL = "__none__";

export function VelocityPage() {
  const { organizationId, factoryId } = useFactoriesLayout();
  const model = useVelocityPageModel(organizationId, factoryId);

  const header = <VelocityHeader model={model} />;

  if (model.velocity.error) {
    return renderShell(header, <VelocityErrorState onRetry={model.velocity.refetch} />);
  }

  if (model.velocity.isLoading || !model.velocity.data) {
    return renderShell(header, <VelocityLoadingState />);
  }

  const sourceSplit: VelocitySourceSplitConfig = {
    hasPeopleCohort: model.velocity.hasPeopleCohort,
    repositoryLabel: model.velocity.repositoryLabel,
    emptyState: renderSourceSplitEmptyState({
      hasGithubIntegration: model.githubIntegrations.length > 0,
      hasIntegrationSelected: Boolean(model.integrationId),
      hasRepositorySelected: Boolean(model.repository),
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
      workOrderFlow={model.workOrderFlow.isLoading ? undefined : { flow: model.workOrderFlow.flow }}
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
  const hasIntegrations = model.githubIntegrations.length > 0;

  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      title="Velocity"
      subtitle="Merged pull requests and work order time."
      actions={
        <div className="flex flex-wrap items-center gap-2">
          {model.githubIntegrations.length > 1 ? (
            <IntegrationPicker
              integrations={model.githubIntegrations}
              selectedIntegrationId={model.integrationId}
              onChange={model.setIntegrationId}
            />
          ) : null}

          <RepositoryPicker
            hasIntegrations={hasIntegrations}
            options={model.repositoryOptions}
            loading={model.repositoriesLoading}
            value={model.repository}
            onChange={model.setRepository}
          />

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
        </div>
      }
    />
  );
}

function IntegrationPicker({
  integrations,
  selectedIntegrationId,
  onChange,
}: {
  integrations: OrganizationsIntegration[];
  selectedIntegrationId: string;
  onChange: (integrationId: string) => void;
}) {
  return (
    <Select value={selectedIntegrationId} onValueChange={onChange}>
      <SelectTrigger className="h-8 min-w-40 text-[12px]" aria-label="GitHub integration">
        <SelectValue placeholder="GitHub integration" />
      </SelectTrigger>
      <SelectContent>
        {integrations.map((integration) => {
          const id = integration.metadata?.id ?? "";
          const name = integration.metadata?.name ?? id;
          return (
            <SelectItem key={id} value={id}>
              {name}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}

function repositoryPlaceholder(hasIntegrations: boolean, loading: boolean): string {
  if (!hasIntegrations) return "Connect GitHub";
  if (loading) return "Loading repositories…";
  return "Select repository";
}

function RepositoryPicker({
  hasIntegrations,
  options,
  loading,
  value,
  onChange,
}: {
  hasIntegrations: boolean;
  options: string[];
  loading: boolean;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Select
      value={value || NO_REPO_SENTINEL}
      onValueChange={(next) => onChange(next === NO_REPO_SENTINEL ? "" : next)}
      disabled={!hasIntegrations || loading}
    >
      <SelectTrigger className="h-8 min-w-52 text-[12px]" aria-label="Repository">
        <SelectValue placeholder={repositoryPlaceholder(hasIntegrations, loading)} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={NO_REPO_SENTINEL}>All work orders (no repo)</SelectItem>
        {options.map((repo) => (
          <SelectItem key={repo} value={repo}>
            {repo}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

interface SourceSplitEmptyStateArgs {
  hasGithubIntegration: boolean;
  hasIntegrationSelected: boolean;
  hasRepositorySelected: boolean;
  peopleSearchFailed: boolean;
}

function renderSourceSplitEmptyState({
  hasGithubIntegration,
  hasIntegrationSelected,
  hasRepositorySelected,
  peopleSearchFailed,
}: SourceSplitEmptyStateArgs) {
  if (!hasGithubIntegration) {
    return <p className="text-[13px] text-muted-foreground">Connect GitHub to compare People and SuperPlane.</p>;
  }
  if (!hasIntegrationSelected) {
    return <p className="text-[13px] text-muted-foreground">Select a GitHub integration and repository.</p>;
  }
  if (peopleSearchFailed) {
    return (
      <p className="text-[13px] text-muted-foreground">
        We could not load People merges. SuperPlane counts still show.
      </p>
    );
  }
  if (!hasRepositorySelected) {
    return <p className="text-[13px] text-muted-foreground">Select a repository to compare People and SuperPlane.</p>;
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
