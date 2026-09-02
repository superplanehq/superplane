import { formatCompactTokens, formatDurationSeconds, formatUsdCents } from "../../../lib/workOrderUsage";

import { SettingsStat } from "./settingsRedesignParts";

export function SettingsSpendingStats({
  periodDays,
  totalTokens,
  totalCostCents,
  totalDurationSeconds,
}: {
  periodDays: number;
  totalTokens: number;
  totalCostCents: number;
  totalDurationSeconds?: number;
}) {
  return (
    <div
      className="grid gap-6 border-y border-border py-6 sm:grid-cols-3"
      data-testid="settings-redesign-spending-stats"
    >
      <SettingsStat label={`Tokens · last ${periodDays} days`} value={formatCompactTokens(totalTokens)} />
      <SettingsStat label="VM time" value={formatDurationSeconds(totalDurationSeconds ?? 0)} />
      <SettingsStat label="Estimated spend" value={formatUsdCents(totalCostCents)} />
    </div>
  );
}
