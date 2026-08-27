import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { factoryCardClassName } from "../factoryPageLayoutStyles";

type UsageByModel = {
  provider?: string;
  model?: string;
  totalTokens?: string | number;
  costCents?: string | number;
};

export function LLMUsageTotals({
  periodDays,
  totalTokens,
  totalCostCents,
}: {
  periodDays: number;
  totalTokens: number;
  totalCostCents: number;
}) {
  return (
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
  );
}

export function LLMUsageByModelTable({ byModel }: { byModel: UsageByModel[] }) {
  return (
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
  );
}
