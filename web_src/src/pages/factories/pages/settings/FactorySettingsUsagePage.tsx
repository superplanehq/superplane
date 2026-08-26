import { usePageTitle } from "@/hooks/usePageTitle";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { useUpdateFactory } from "@/hooks/useFactoryData";
import { usePermissions } from "@/contexts/usePermissions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { centsToDollarInput, dollarInputToCents } from "@/lib/hostedCredit";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { factoryCardClassName, factoryContentBodyClassName } from "../factoryPageLayoutStyles";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
import { useEffect, useState } from "react";

export function FactorySettingsUsagePage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { data, isLoading, error } = useFactoryUsage(organizationId, factoryId);

  usePageTitle(["Usage", "Settings", factory.name ?? "Workspace"]);

  const totalTokens = parseWorkOrderMetric(data?.totalTokens);
  const totalCostCents = parseWorkOrderMetric(data?.totalCostCents);
  const periodDays = data?.periodDays ?? 30;
  const byModel = data?.byModel ?? [];

  return (
    <>
      <WorkspacePageHeader title="Usage" subtitle="LLM tokens and estimated spend for this workspace." />
      <div className={factoryContentBodyClassName}>
        {isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading usage...</p>
        ) : error ? (
          <p className="text-[13px] text-destructive">Unable to load usage.</p>
        ) : (
          <div className="flex flex-col gap-4">
            <div className={`grid gap-3 sm:grid-cols-2 ${factoryCardClassName} p-4`}>
              <div>
                <p className="workspace-section-label">Tokens · last {periodDays} days</p>
                <p className="workspace-page-title mt-1">{formatCompactTokens(totalTokens)}</p>
              </div>
              <div>
                <p className="workspace-section-label">Estimated spend</p>
                <p className="workspace-page-title mt-1">{formatUsdCents(totalCostCents)}</p>
              </div>
            </div>
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
            <div className={factoryCardClassName}>
              <p className="workspace-section-title px-4 pt-4">By model</p>
              {byModel.length === 0 ? (
                <p className="px-4 pb-4 pt-2 text-[13px] text-muted-foreground">
                  No factory LLM usage is recorded for this period.
                </p>
              ) : (
                <table className="mt-2 w-full text-left text-[13px]">
                  <thead>
                    <tr className="border-b border-border text-muted-foreground">
                      <th className="px-4 py-2 font-medium">Provider</th>
                      <th className="px-4 py-2 font-medium">Model</th>
                      <th className="px-4 py-2 font-medium">Tokens</th>
                      <th className="px-4 py-2 font-medium">Spend</th>
                    </tr>
                  </thead>
                  <tbody>
                    {byModel.map((row) => (
                      <tr key={`${row.provider}-${row.model}`} className="border-b border-border last:border-0">
                        <td className="px-4 py-2">{row.provider}</td>
                        <td className="px-4 py-2">{row.model}</td>
                        <td className="px-4 py-2">{formatCompactTokens(parseWorkOrderMetric(row.totalTokens))}</td>
                        <td className="px-4 py-2">{formatUsdCents(parseWorkOrderMetric(row.costCents))}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
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
  const [dollars, setDollars] = useState(hasLimit ? centsToDollarInput(Number(currentBudget)) : "0.00");

  useEffect(() => {
    const nextHasLimit = currentBudget !== undefined && currentBudget !== null;
    setNoLimit(!nextHasLimit);
    setDollars(nextHasLimit ? centsToDollarInput(Number(currentBudget)) : "0.00");
  }, [currentBudget]);

  const save = async () => {
    try {
      await updateFactory.mutateAsync({
        hostedSpendBudgetCents: noLimit ? null : dollarInputToCents(dollars),
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
        <Label className="text-[13px]">No limit</Label>
        <Switch checked={noLimit} disabled={!canUpdate || updateFactory.isPending} onCheckedChange={setNoLimit} />
      </div>
      {!noLimit ? (
        <div className="mt-3">
          <Label className="text-[13px]">Limit in USD</Label>
          <Input
            className="mt-1"
            value={dollars}
            onChange={(event) => setDollars(event.target.value)}
            disabled={!canUpdate || updateFactory.isPending}
          />
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
        <Button className="mt-3" type="button" onClick={() => void save()} disabled={updateFactory.isPending}>
          {updateFactory.isPending ? "Saving..." : "Save hosted spend limit"}
        </Button>
      ) : null}
    </div>
  );
}
