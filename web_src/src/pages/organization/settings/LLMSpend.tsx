import { usePageTitle } from "@/hooks/usePageTitle";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import { HostedCreditSummary } from "./HostedCreditSummary";
import { settingsCardClassName, settingsInnerMetricCardClassName } from "./settingsPageStyles";

interface LLMSpendProps {
  organizationId: string;
}

export function LLMSpend({ organizationId }: LLMSpendProps) {
  usePageTitle(["LLM spend"]);
  const { data, isLoading, error } = useOrganizationLLMSpend(organizationId);

  const totalTokens = parseWorkOrderMetric(data?.totalTokens);
  const totalCostCents = parseWorkOrderMetric(data?.totalCostCents);
  const periodDays = data?.periodDays ?? 30;
  const byModel = data?.byModel ?? [];

  if (isLoading) {
    return (
      <div className="pt-6">
        <div className={settingsCardClassName}>
          <p className="text-sm text-gray-500 dark:text-gray-400">Loading LLM spend...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-6">
        <div className={settingsCardClassName}>
          <p className="text-sm text-red-500">Unable to load LLM spend.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 pt-6">
      <div className={`grid gap-3 sm:grid-cols-2 ${settingsCardClassName}`}>
        <div className={settingsInnerMetricCardClassName}>
          <p className="text-xs font-medium tracking-wide text-gray-500 uppercase">Tokens · last {periodDays} days</p>
          <p className="mt-2 text-lg font-semibold">{formatCompactTokens(totalTokens)}</p>
        </div>
        <div className={settingsInnerMetricCardClassName}>
          <p className="text-xs font-medium tracking-wide text-gray-500 uppercase">Estimated spend</p>
          <p className="mt-2 text-lg font-semibold">{formatUsdCents(totalCostCents)}</p>
        </div>
      </div>
      <HostedCreditSummary
        remainingCreditCents={data?.remainingCreditCents}
        grantTotalCents={data?.grantTotalCents}
        hostedBilledCents={data?.hostedBilledCents}
        remainingCreditWarning={data?.remainingCreditWarning}
        cardClassName={settingsCardClassName}
        labelClassName="text-xs font-medium tracking-wide text-gray-500 uppercase"
        valueClassName="mt-2 text-lg font-semibold"
      />
      <div className={settingsCardClassName}>
        <p className="text-sm font-semibold">By model</p>
        {byModel.length === 0 ? (
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            No factory LLM usage is recorded for this period.
          </p>
        ) : (
          <table className="mt-3 w-full text-left text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-gray-500 dark:border-gray-700">
                <th className="py-2 font-medium">Provider</th>
                <th className="py-2 font-medium">Model</th>
                <th className="py-2 font-medium">Tokens</th>
                <th className="py-2 font-medium">Spend</th>
              </tr>
            </thead>
            <tbody>
              {byModel.map((row) => (
                <tr
                  key={`${row.provider}-${row.model}`}
                  className="border-b border-gray-100 last:border-0 dark:border-gray-800"
                >
                  <td className="py-2">{row.provider}</td>
                  <td className="py-2">{row.model}</td>
                  <td className="py-2">{formatCompactTokens(parseWorkOrderMetric(row.totalTokens))}</td>
                  <td className="py-2">{formatUsdCents(parseWorkOrderMetric(row.costCents))}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
