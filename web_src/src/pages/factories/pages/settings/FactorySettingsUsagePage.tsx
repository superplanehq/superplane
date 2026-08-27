import { usePageTitle } from "@/hooks/usePageTitle";
import { useFactoryUsage } from "@/hooks/useFactoryUsage";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { factoryCardClassName, factoryContentBodyClassName } from "../factoryPageLayoutStyles";
import { HostedCreditSummary } from "@/pages/organization/settings/HostedCreditSummary";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

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
