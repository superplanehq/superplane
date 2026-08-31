import { usePageTitle } from "@/hooks/usePageTitle";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { useUpdateFactory } from "@/hooks/useFactoryData";
import { usePermissions } from "@/contexts/usePermissions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { centsToDollarInput, parseDollarInputToCents } from "@/lib/hostedCredit";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { factoryCardClassName, factoryContentBodyClassName } from "../factoryPageLayoutStyles";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import {
  WorkspaceUsageByMachineTypeTable,
  WorkspaceUsageByModelTable,
  WorkspaceUsageTotals,
} from "./WorkspaceUsageBreakdown";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { useEffect, useState } from "react";

export function FactorySettingsUsagePage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { data, isLoading, error } = useFactoryUsage(organizationId, factoryId);

  usePageTitle(["Usage", "Settings", factory.name ?? "Workspace"]);

  const totalTokens = parseWorkOrderMetric(data?.totalTokens);
  const totalCostCents = parseWorkOrderMetric(data?.totalCostCents);
  const totalDurationSeconds = parseWorkOrderMetric(data?.totalDurationSeconds);
  const periodDays = data?.periodDays ?? 30;
  const byModel = data?.byModel ?? [];
  const byMachineType = data?.byMachineType ?? [];

  return (
    <>
      <WorkspacePageHeader title="Usage" subtitle="LLM tokens, VM seconds, and estimated spend for this workspace." />
      <div className={factoryContentBodyClassName}>
        {isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading usage...</p>
        ) : error ? (
          <p className="text-[13px] text-destructive">Unable to load usage.</p>
        ) : (
          <div className="flex flex-col gap-4">
            <WorkspaceUsageTotals
              periodDays={periodDays}
              totalTokens={totalTokens}
              totalCostCents={totalCostCents}
              totalDurationSeconds={totalDurationSeconds}
            />
            <HostedSpendLimitCard />
            <HostedCreditSummary
              remainingCreditCents={data?.remainingCreditCents}
              grantTotalCents={data?.grantTotalCents}
              hostedBilledCents={data?.hostedBilledCents}
              remainingCreditWarning={data?.remainingCreditWarning}
              cardClassName={`${factoryCardClassName} p-4`}
              labelClassName="workspace-section-label"
              valueClassName="workspace-page-title mt-1"
            />
            <WorkspaceUsageByModelTable byModel={byModel} />
            <WorkspaceUsageByMachineTypeTable byMachineType={byMachineType} />
          </div>
        )}
      </div>
    </>
  );
}

function HostedSpendLimitCard() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { data } = useFactoryUsage(organizationId, factoryId);
  const { canAct } = usePermissions();
  const canUpdate = canAct("factories", "update");
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const currentBudget = factory.hostedSpendBudgetCents;
  const hasLimit = currentBudget !== undefined && currentBudget !== null;
  const [noLimit, setNoLimit] = useState(!hasLimit);
  const [dollars, setDollars] = useState(hasLimit ? centsToDollarInput(Number(currentBudget)) : "");
  const limitCents = parseDollarInputToCents(dollars);
  const canSaveLimit = noLimit || limitCents !== null;

  useEffect(() => {
    const nextHasLimit = currentBudget !== undefined && currentBudget !== null;
    setNoLimit(!nextHasLimit);
    setDollars(nextHasLimit ? centsToDollarInput(Number(currentBudget)) : "");
  }, [currentBudget]);

  const save = async () => {
    if (!noLimit && limitCents === null) {
      showErrorToast("Enter a valid spend limit in USD.");
      return;
    }
    try {
      await updateFactory.mutateAsync({
        hostedSpendBudgetCents: noLimit ? null : limitCents,
      });
      showSuccessToast("Hosted spend limit saved.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Unable to save hosted spend limit."));
    }
  };

  return (
    <div className={`${factoryCardClassName} p-4`}>
      <p className="workspace-section-title">Hosted spend limit</p>
      <div className="mt-3 flex items-center justify-between gap-3">
        <Label htmlFor="hosted-spend-no-limit" className="text-[13px]">
          No limit
        </Label>
        <Switch
          id="hosted-spend-no-limit"
          checked={noLimit}
          disabled={!canUpdate || updateFactory.isPending}
          onCheckedChange={setNoLimit}
        />
      </div>
      {!noLimit ? (
        <div className="mt-3">
          <Label htmlFor="hosted-spend-limit-usd" className="text-[13px]">
            Limit in USD
          </Label>
          <Input
            id="hosted-spend-limit-usd"
            className="mt-1"
            value={dollars}
            placeholder="50.00"
            onChange={(event) => setDollars(event.target.value)}
            disabled={!canUpdate || updateFactory.isPending}
          />
          <p className="mt-2 text-[13px] text-muted-foreground">
            Enter an amount in USD. SuperPlane-hosted runs stop when this workspace reaches the limit.
          </p>
        </div>
      ) : (
        <p className="mt-2 text-[13px] text-muted-foreground">
          This workspace can use remaining organization hosted credit.
        </p>
      )}
      {data?.factoryRemainingCreditWarning ? (
        <p className="mt-3 text-[13px] text-amber-700 dark:text-amber-400">
          Hosted credit for this workspace is low. SuperPlane-hosted runs stop when remaining credit is empty.
        </p>
      ) : null}
      {data?.hostedSpendBudgetCents !== undefined && data.hostedSpendBudgetCents !== null ? (
        <p className="mt-3 text-[13px] text-muted-foreground">
          Remaining in this workspace: {formatUsdCents(parseWorkOrderMetric(data.factoryRemainingCreditCents))} of{" "}
          {formatUsdCents(parseWorkOrderMetric(data.hostedSpendBudgetCents))}.
        </p>
      ) : null}
      {canUpdate ? (
        <Button
          className="mt-3"
          type="button"
          onClick={() => void save()}
          disabled={updateFactory.isPending || !canSaveLimit}
        >
          {updateFactory.isPending ? "Saving..." : "Save hosted spend limit"}
        </Button>
      ) : null}
    </div>
  );
}
