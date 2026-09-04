import { useMemo, useState } from "react";
import { useParams } from "react-router";

import { useOrganizationSpendingReport } from "@/hooks/useOrganizationSpendingReport";
import { SpendingRedesignPage } from "./spending-redesign/SpendingRedesignPage";
import {
  EMPTY_SPENDING_FILTERS,
  rangeForPreset,
  type SpendingBreakdown,
  type SpendingDateRange,
  type SpendingFilters,
  type SpendingPeriodPreset,
} from "./spending-redesign/spendingRedesignLib";
import {
  mapSpendingCatalogs,
  mapSpendingCreditSnapshot,
  mapSpendingExplorerReport,
  mapSpendingKpiTotals,
} from "./spending-redesign/spendingReportMapper";

export function OrganizationSettingsWorkspaceUsagePage() {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const [period, setPeriod] = useState<SpendingPeriodPreset>("month");
  const [customRange, setCustomRange] = useState<SpendingDateRange | undefined>();
  const [customOpen, setCustomOpen] = useState(false);
  const [modelFilters, setModelFilters] = useState<SpendingFilters>(EMPTY_SPENDING_FILTERS);
  const [machineFilters, setMachineFilters] = useState<SpendingFilters>(EMPTY_SPENDING_FILTERS);
  const [modelBreakdown, setModelBreakdown] = useState<SpendingBreakdown>("workspace");
  const [machineBreakdown, setMachineBreakdown] = useState<SpendingBreakdown>("workspace");

  const range = useMemo(() => {
    if (period === "custom") {
      return customRange ?? rangeForPreset("week", new Date());
    }
    return rangeForPreset(period, new Date());
  }, [customRange, period]);

  const modelQuery = useOrganizationSpendingReport({
    organizationId,
    range,
    usageKind: "model",
    filters: modelFilters,
    groupBy: modelBreakdown,
  });
  const machineQuery = useOrganizationSpendingReport({
    organizationId,
    range,
    usageKind: "compute",
    filters: machineFilters,
    groupBy: machineBreakdown,
  });

  const isLoading = modelQuery.isLoading || machineQuery.isLoading;
  const error = modelQuery.error ?? machineQuery.error;
  const baseResponse = modelQuery.data ?? machineQuery.data;

  const catalogs = useMemo(() => mapSpendingCatalogs(baseResponse?.catalogs), [baseResponse?.catalogs]);
  const credit = useMemo(() => mapSpendingCreditSnapshot(baseResponse?.credit), [baseResponse?.credit]);
  const kpiTotals = useMemo(() => mapSpendingKpiTotals(baseResponse?.kpiTotals), [baseResponse?.kpiTotals]);
  const modelReport = useMemo(
    () => (modelQuery.data ? mapSpendingExplorerReport(modelQuery.data, range) : undefined),
    [modelQuery.data, range],
  );
  const machineReport = useMemo(
    () => (machineQuery.data ? mapSpendingExplorerReport(machineQuery.data, range) : undefined),
    [machineQuery.data, range],
  );

  return (
    <SpendingRedesignPage
      catalogs={catalogs}
      credit={credit}
      customOpen={customOpen}
      customRange={customRange}
      errorMessage={error ? "Unable to load spending." : undefined}
      isLoading={isLoading}
      kpiTotals={kpiTotals}
      machineBreakdown={machineBreakdown}
      machineFilters={machineFilters}
      machineReport={machineReport}
      modelBreakdown={modelBreakdown}
      modelFilters={modelFilters}
      modelReport={modelReport}
      period={period}
      range={range}
      onCustomOpenChange={setCustomOpen}
      onCustomRangeChange={(next) => {
        setCustomRange(next);
        setPeriod("custom");
      }}
      onMachineBreakdownChange={setMachineBreakdown}
      onMachineFiltersChange={setMachineFilters}
      onModelBreakdownChange={setModelBreakdown}
      onModelFiltersChange={setModelFilters}
      onPeriodChange={(next) => {
        setPeriod(next);
        if (next !== "custom") {
          setCustomOpen(false);
        }
      }}
    />
  );
}
