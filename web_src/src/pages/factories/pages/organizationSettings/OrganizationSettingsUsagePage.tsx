import { useMemo, useState } from "react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useFactoryWorkOrderRunUsage, WORK_ORDER_RUN_USAGE_PAGE_SIZE } from "@/hooks/useFactoryWorkOrderRunUsage";
import { getApiErrorMessage } from "@/lib/errors";
import type { FactoriesWorkOrderRunUsageRow } from "@/api-client";

import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import {
  formatUsdCents,
  formatUsageOccurredAtUtc,
  formatUsageRunResources,
  formatUsageTaskName,
  formatUsageTokensAndTime,
  parseWorkOrderMetric,
} from "../../lib/workOrderUsage";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";
import { useFactorySettingsLayout } from "../settings/factorySettingsLayoutContext";
import {
  quantizeSpendingNow,
  rangeForPreset,
  type SpendingDateRange,
  type SpendingPeriodPreset,
} from "./spending-redesign/spendingRedesignLib";
import { SpendingPeriodControls } from "./spending-redesign/SpendingPeriodControls";

export function OrganizationSettingsUsagePage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const factoryKey = factory.key ?? "";
  const [period, setPeriod] = useState<SpendingPeriodPreset>("month");
  const [customRange, setCustomRange] = useState<SpendingDateRange | undefined>();
  const [customOpen, setCustomOpen] = useState(false);

  const range = useMemo(() => {
    const now = quantizeSpendingNow(new Date());
    if (period === "custom") {
      return customRange ?? rangeForPreset("week", now);
    }
    return rangeForPreset(period, now);
  }, [customRange, period]);

  const [offset, setOffset] = useUsagePageOffset(
    `${factoryId}|${range.start.toISOString()}|${range.end.toISOString()}`,
  );

  const query = useFactoryWorkOrderRunUsage({
    organizationId,
    factoryId,
    startTime: range.start.toISOString(),
    endTime: range.end.toISOString(),
    offset,
  });

  const rows = query.data?.rows ?? [];
  const totalCount = query.data?.totalCount ?? 0;
  const hasData = query.data !== undefined;
  const showFullPageLoading = query.isLoading && !hasData;

  usePageTitle(["Usage", factory.name ?? "Workspace"]);
  useReportPageReady(!query.isLoading, { failed: Boolean(query.error) });

  return (
    <FactorySettingsPageFrame
      title="Usage"
      subtitle="Review task spend for this workspace."
      actions={
        <SpendingPeriodControls
          customOpen={customOpen}
          customRange={range}
          label="Usage period"
          period={period}
          pickerTestId="usage-period-picker"
          testId="usage-period"
          onCustomOpenChange={setCustomOpen}
          onCustomRangeChange={(next) => {
            setCustomRange(next);
            setPeriod("custom");
          }}
          onPeriodChange={(next) => {
            setPeriod(next as SpendingPeriodPreset);
            if (next !== "custom") {
              setCustomOpen(false);
            }
          }}
        />
      }
    >
      <p className="text-[13px] text-muted-foreground">
        This list does not show which credit grant paid. Your keys spend is estimated.
      </p>
      <FactorySettingsCard data-testid="organization-usage-history">
        <UsageHistoryBody
          error={query.error}
          factoryKey={factoryKey}
          isLoading={showFullPageLoading}
          offset={offset}
          organizationId={organizationId}
          rows={rows}
          totalCount={totalCount}
          onOffsetChange={setOffset}
        />
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}

/**
 * Owns the table's page offset.
 *
 * A workspace switch or a new period starts a new list, so the offset resets
 * to the first page. This mirrors React's "adjust state during render" pattern
 * rather than an effect, so the reset lands before the
 * `useFactoryWorkOrderRunUsage` call that reads `offset` in the same render.
 */
function useUsagePageOffset(resetKey: string) {
  const [offset, setOffset] = useState(0);
  const [appliedResetKey, setAppliedResetKey] = useState(resetKey);
  if (appliedResetKey !== resetKey) {
    setAppliedResetKey(resetKey);
    setOffset(0);
  }
  return [offset, setOffset] as const;
}

function UsageHistoryBody({
  error,
  factoryKey,
  isLoading,
  offset,
  organizationId,
  rows,
  totalCount,
  onOffsetChange,
}: {
  error: unknown;
  factoryKey: string;
  isLoading: boolean;
  offset: number;
  organizationId: string;
  rows: FactoriesWorkOrderRunUsageRow[];
  totalCount: number;
  onOffsetChange: (offset: number) => void;
}) {
  if (isLoading) {
    return (
      <p className="text-[13px] text-muted-foreground" data-testid="organization-usage-loading">
        Loading usage...
      </p>
    );
  }
  if (error) {
    return <p className="text-[13px] text-destructive">{getApiErrorMessage(error, "Unable to load usage.")}</p>;
  }
  if (rows.length === 0) {
    return <p className="text-[13px] text-muted-foreground">No task spend in this period.</p>;
  }

  return (
    <>
      <table className="mt-1 w-full text-left text-[13px]">
        <thead>
          <tr className="border-b border-border text-muted-foreground">
            <th className="py-2 pr-3 font-medium">Date (UTC)</th>
            <th className="py-2 pr-3 font-medium">User</th>
            <th className="py-2 pr-3 font-medium">Task</th>
            <th className="py-2 pr-3 font-medium">Tokens | time</th>
            <th className="py-2 pr-3 font-medium">Cost</th>
            <th className="py-2 font-medium">Model | machine</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <UsageHistoryRow
              key={row.workOrderExecutionId ?? `${row.workOrderId}-${row.lastOccurredAt}`}
              factoryKey={factoryKey}
              organizationId={organizationId}
              row={row}
            />
          ))}
        </tbody>
      </table>
      <UsageHistoryPagination
        offset={offset}
        pageSize={WORK_ORDER_RUN_USAGE_PAGE_SIZE}
        totalCount={totalCount}
        onOffsetChange={onOffsetChange}
      />
    </>
  );
}

function UsageHistoryRow({
  factoryKey,
  organizationId,
  row,
}: {
  factoryKey: string;
  organizationId: string;
  row: FactoriesWorkOrderRunUsageRow;
}) {
  const taskName = formatUsageTaskName(row.workOrderKey, row.title);
  const href = workOrderDetailPath(organizationId, factoryKey, row.workOrderNumber ?? "");

  return (
    <tr className="border-b border-border last:border-0" data-testid="organization-usage-row">
      <td className="py-2 pr-3 whitespace-nowrap text-muted-foreground">
        {formatUsageOccurredAtUtc(row.lastOccurredAt)}
      </td>
      <td className="py-2 pr-3 whitespace-nowrap text-muted-foreground">{row.userName || row.userEmail || "—"}</td>
      <td className="py-2 pr-3">
        <Link className="font-medium text-foreground underline-offset-2 hover:underline" to={href}>
          {taskName}
        </Link>
      </td>
      <td className="py-2 pr-3 whitespace-nowrap text-muted-foreground">
        {formatUsageTokensAndTime(parseWorkOrderMetric(row.totalTokens), parseWorkOrderMetric(row.durationSeconds))}
      </td>
      <td className="py-2 pr-3 whitespace-nowrap">{formatUsdCents(parseWorkOrderMetric(row.costCents))}</td>
      <td className="py-2 text-muted-foreground">
        {formatUsageRunResources({
          models: row.models,
          byokModels: row.byokModels,
          machineTypes: row.machineTypes,
        })}
      </td>
    </tr>
  );
}

function UsageHistoryPagination({
  offset,
  pageSize,
  totalCount,
  onOffsetChange,
}: {
  offset: number;
  pageSize: number;
  totalCount: number;
  onOffsetChange: (offset: number) => void;
}) {
  if (totalCount <= pageSize) {
    return null;
  }

  const from = offset + 1;
  const to = Math.min(offset + pageSize, totalCount);
  const previousOffset = Math.max(0, offset - pageSize);
  const nextOffset = offset + pageSize;

  return (
    <div className="mt-3 flex items-center justify-between gap-3" data-testid="organization-usage-pagination">
      <p className="text-[12px] text-muted-foreground">
        Showing {from}–{to} of {totalCount}
      </p>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={offset === 0}
          onClick={() => onOffsetChange(previousOffset)}
        >
          Previous
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={nextOffset >= totalCount}
          onClick={() => onOffsetChange(nextOffset)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
