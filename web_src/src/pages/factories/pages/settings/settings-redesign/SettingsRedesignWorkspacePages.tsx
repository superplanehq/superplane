import { type ReactNode } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { LoadingButton } from "@/components/ui/loading-button";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { useOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { FactoryDeleteDialog } from "../../../FactoryDeleteDialog";
import { parseWorkOrderMetric } from "../../../lib/workOrderUsage";
import { RepositoryPicker } from "../../onboarding/onboardingSteps";
import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { HostedSpendLimitCard } from "../FactorySettingsUsagePage";
import { WorkspaceUsageByMachineTypeTable, WorkspaceUsageByModelTable } from "../WorkspaceUsageBreakdown";
import { WorkspaceDangerSection, WorkspaceIdentityForm } from "./SettingsRedesignWorkspaceGeneralCards";
import { SettingsIdentityHero } from "./settingsRedesignParts";
import { useSettingsRedesignWorkspaceGeneral } from "./useSettingsRedesignWorkspaceGeneral";
import { useSettingsRedesignWorkspaceRepository } from "./useSettingsRedesignWorkspaceRepository";
import { SettingsSpendingStats } from "./SettingsRedesignSpendingPages";

export function SettingsRedesignWorkspaceGeneralPage() {
  const general = useSettingsRedesignWorkspaceGeneral();
  const { organizationId } = useFactorySettingsLayout();
  const { data: organization } = useOrganization(organizationId);
  const organizationSlug = organization?.metadata?.slug || organizationId;
  usePageTitle(["General", "Settings", general.factory.name ?? "Workspace"]);

  return (
    <>
      <FactorySettingsPageFrame title="General" subtitle="Identity and URL for this workspace.">
        <div className="space-y-8" data-testid="factory-settings-general-form">
          <WorkspaceIdentityForm general={general} organizationSlug={organizationSlug} />
          <WorkspaceDangerSection general={general} />
        </div>
      </FactorySettingsPageFrame>

      <FactoryDeleteDialog
        open={general.deleteOpen}
        factoryName={general.factory.name ?? ""}
        canDelete={general.canDelete}
        isDeleting={general.deleteFactory.isPending}
        onClose={() => general.setDeleteOpen(false)}
        onConfirm={general.handleDelete}
      />
    </>
  );
}

export function SettingsRedesignWorkspaceRepositoryPage() {
  const repo = useSettingsRedesignWorkspaceRepository();
  usePageTitle(["Repository", "Settings", repo.factoryName]);

  return (
    <FactorySettingsPageFrame title="Repository" subtitle="GitHub repository for workspace work and issue intake.">
      <div className="space-y-6" data-testid="factory-settings-repository">
        <SettingsIdentityHero
          leading={
            <div className="flex size-14 items-center justify-center rounded-xl border border-border bg-muted">
              <IntegrationIcon integrationName="github" className="size-7" size={28} />
            </div>
          }
          title={repo.repository || "No repository"}
          caption="Tasks and pull request automations use this repository."
          testId="settings-redesign-repository-hero"
        />
        <WorkspaceRepositoryPicker
          integrationId={repo.integrationId}
          loading={repo.resourcesLoading}
          error={repo.resourcesError}
          repositories={repo.repositories}
          repository={repo.repository}
          onSelect={repo.setRepository}
        />
        <div className="flex justify-end border-t border-border pt-4">
          <PermissionTooltip
            allowed={repo.canUpdate || repo.permissionsLoading}
            message="You do not have permission to update this workspace."
          >
            <LoadingButton
              type="button"
              loading={repo.isSaving}
              loadingText="Saving..."
              disabled={!repo.canSave}
              onClick={repo.save}
              data-testid="factory-settings-repository-save"
            >
              Save repository
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </div>
    </FactorySettingsPageFrame>
  );
}

function WorkspaceRepositoryPicker({
  integrationId,
  loading,
  error,
  repositories,
  repository,
  onSelect,
}: {
  integrationId: string;
  loading: boolean;
  error: boolean;
  repositories: string[];
  repository: string;
  onSelect: (repository: string) => void;
}) {
  let content: ReactNode;
  if (!integrationId) {
    content = (
      <p className="text-[13px] text-muted-foreground">
        Connect GitHub during workspace setup before you select a repository.
      </p>
    );
  } else if (loading) {
    content = <p className="text-[13px] text-muted-foreground">Loading repositories...</p>;
  } else {
    content = (
      <RepositoryPicker
        host="github"
        repos={repositories}
        selectedRepo={repository || null}
        onSelect={onSelect}
        searchClassName="h-10"
        itemClassName="py-3"
      />
    );
  }

  return (
    <>
      {content}
      {error ? <p className="text-[13px] text-destructive">We could not load repositories. Try again.</p> : null}
    </>
  );
}

export function SettingsRedesignWorkspaceSpendingPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { data, isLoading, error } = useFactoryUsage(organizationId, factoryId);

  usePageTitle(["Spending", "Settings", factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame title="Spending" subtitle="Usage for this workspace in the last 30 days.">
      <p className="text-[12px] text-muted-foreground" data-testid="settings-redesign-spending-scope">
        Organization billing is under Organization, Spending.
      </p>

      {isLoading ? <p className="text-[13px] text-muted-foreground">Loading usage...</p> : null}
      {error || (!isLoading && !data) ? <p className="text-[13px] text-destructive">Unable to load usage.</p> : null}
      {data ? (
        <>
          <SettingsSpendingStats
            periodDays={data.periodDays ?? 30}
            totalTokens={parseWorkOrderMetric(data.totalTokens)}
            totalCostCents={parseWorkOrderMetric(data.totalCostCents)}
            totalDurationSeconds={parseWorkOrderMetric(data.totalDurationSeconds)}
          />
          <HostedCreditSummary
            remainingCreditCents={data.remainingCreditCents}
            grantTotalCents={data.grantTotalCents}
            hostedBilledCents={data.hostedBilledCents}
            remainingCreditWarning={data.remainingCreditWarning}
            cardClassName="border-t border-border pt-6"
            labelClassName="text-[12px] text-muted-foreground"
            valueClassName="mt-1 text-[26px] font-medium tracking-tight"
          />
          <HostedSpendLimitCard />
          <WorkspaceUsageByModelTable byModel={data.byModel ?? []} />
          <WorkspaceUsageByMachineTypeTable byMachineType={data.byMachineType ?? []} />
        </>
      ) : null}
    </FactorySettingsPageFrame>
  );
}
