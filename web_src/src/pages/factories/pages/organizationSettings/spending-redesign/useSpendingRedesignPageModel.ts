import { useMemo, useState } from "react";

import {
  buildSpendingReport,
  EMPTY_SPENDING_FILTERS,
  filterSpendingEvents,
  rangeForPreset,
  sumSpendingTotals,
  type SpendingBreakdown,
  type SpendingCatalogs,
  type SpendingDateRange,
  type SpendingFilters,
  type SpendingPeriodPreset,
  type SpendingReport,
  type SpendingTotals,
  type SpendingUsageEvent,
} from "./spendingRedesignLib";

export interface SpendingRedesignControlledState {
  period?: SpendingPeriodPreset;
  customRange?: SpendingDateRange;
  customOpen?: boolean;
  modelFilters?: SpendingFilters;
  machineFilters?: SpendingFilters;
  modelBreakdown?: SpendingBreakdown;
  machineBreakdown?: SpendingBreakdown;
  range?: SpendingDateRange;
  kpiTotals?: SpendingTotals;
  modelReport?: SpendingReport;
  machineReport?: SpendingReport;
  onPeriodChange?: (period: SpendingPeriodPreset) => void;
  onCustomOpenChange?: (open: boolean) => void;
  onCustomRangeChange?: (range: SpendingDateRange) => void;
  onModelFiltersChange?: (filters: SpendingFilters) => void;
  onMachineFiltersChange?: (filters: SpendingFilters) => void;
  onModelBreakdownChange?: (breakdown: SpendingBreakdown) => void;
  onMachineBreakdownChange?: (breakdown: SpendingBreakdown) => void;
}

interface SpendingRedesignModelArgs extends SpendingRedesignControlledState {
  events?: SpendingUsageEvent[];
  catalogs: SpendingCatalogs;
  now?: Date;
  initialPeriod?: SpendingPeriodPreset;
  initialModelFilters?: SpendingFilters;
  initialMachineFilters?: SpendingFilters;
  initialModelBreakdown?: SpendingBreakdown;
  initialMachineBreakdown?: SpendingBreakdown;
  initialCustomRange?: SpendingDateRange;
}

function bindSpendingField<T>(setState: (value: T) => void, isProduction: boolean, onChange?: (value: T) => void) {
  return (value: T) => {
    onChange?.(value);
    if (!isProduction) {
      setState(value);
    }
  };
}

function resolveSpendingRange(
  rangeProp: SpendingDateRange | undefined,
  period: SpendingPeriodPreset,
  customRange: SpendingDateRange | undefined,
  now: Date,
): SpendingDateRange {
  if (rangeProp) {
    return rangeProp;
  }
  if (period === "custom") {
    return customRange ?? rangeForPreset("week", now);
  }
  return rangeForPreset(period, now);
}

function resolveSpendingReports(args: {
  events: SpendingUsageEvent[];
  range: SpendingDateRange;
  catalogs: SpendingCatalogs;
  modelFilters: SpendingFilters;
  machineFilters: SpendingFilters;
  modelBreakdown: SpendingBreakdown;
  machineBreakdown: SpendingBreakdown;
  modelReportProp?: SpendingReport;
  machineReportProp?: SpendingReport;
  kpiTotalsProp?: SpendingTotals;
}) {
  const rangeTotals =
    args.kpiTotalsProp ?? sumSpendingTotals(filterSpendingEvents(args.events, args.range, EMPTY_SPENDING_FILTERS));

  const modelReport =
    args.modelReportProp ??
    buildSpendingReport({
      events: args.events,
      range: args.range,
      filters: args.modelFilters,
      breakdown: args.modelBreakdown,
      catalogs: args.catalogs,
      usageKind: "model",
    });

  const machineReport =
    args.machineReportProp ??
    buildSpendingReport({
      events: args.events,
      range: args.range,
      filters: args.machineFilters,
      breakdown: args.machineBreakdown,
      catalogs: args.catalogs,
      usageKind: "compute",
    });

  return { rangeTotals, modelReport, machineReport };
}

function createSpendingPeriodChangeHandler(args: {
  isProduction: boolean;
  customRange?: SpendingDateRange;
  now: Date;
  onPeriodChange?: (period: SpendingPeriodPreset) => void;
  onCustomOpenChange?: (open: boolean) => void;
  setPeriodState: (period: SpendingPeriodPreset) => void;
  setCustomOpenState: (open: boolean) => void;
  setCustomRangeState: (range: SpendingDateRange) => void;
}) {
  return (value: string) => {
    const next = value as SpendingPeriodPreset;
    if (next === "custom") {
      args.onPeriodChange?.("custom");
      if (!args.isProduction) {
        args.setPeriodState("custom");
        args.setCustomOpenState(true);
        if (!args.customRange) {
          args.setCustomRangeState(rangeForPreset("week", args.now));
        }
      }
      args.onCustomOpenChange?.(true);
      return;
    }
    args.onPeriodChange?.(next);
    if (!args.isProduction) {
      args.setPeriodState(next);
      args.setCustomOpenState(false);
    }
    args.onCustomOpenChange?.(false);
  };
}

export function useSpendingRedesignPageModel(args: SpendingRedesignModelArgs) {
  const {
    events = [],
    catalogs,
    now = new Date(),
    initialPeriod = "month",
    initialModelFilters = EMPTY_SPENDING_FILTERS,
    initialMachineFilters = EMPTY_SPENDING_FILTERS,
    initialModelBreakdown = "workspace",
    initialMachineBreakdown = "workspace",
    initialCustomRange,
    kpiTotals: kpiTotalsProp,
    modelReport: modelReportProp,
    machineReport: machineReportProp,
    range: rangeProp,
    period: periodProp,
    customRange: customRangeProp,
    customOpen: customOpenProp,
    modelFilters: modelFiltersProp,
    machineFilters: machineFiltersProp,
    modelBreakdown: modelBreakdownProp,
    machineBreakdown: machineBreakdownProp,
    onPeriodChange,
    onCustomOpenChange,
    onCustomRangeChange,
    onModelFiltersChange,
    onMachineFiltersChange,
    onModelBreakdownChange,
    onMachineBreakdownChange,
  } = args;

  const isProduction = modelReportProp !== undefined && machineReportProp !== undefined;
  const [periodState, setPeriodState] = useState(initialPeriod);
  const [customRangeState, setCustomRangeState] = useState(initialCustomRange);
  const [customOpenState, setCustomOpenState] = useState(false);
  const [modelFiltersState, setModelFiltersState] = useState(initialModelFilters);
  const [machineFiltersState, setMachineFiltersState] = useState(initialMachineFilters);
  const [modelBreakdownState, setModelBreakdownState] = useState(initialModelBreakdown);
  const [machineBreakdownState, setMachineBreakdownState] = useState(initialMachineBreakdown);

  const period = periodProp ?? periodState;
  const customRange = customRangeProp ?? customRangeState;
  const customOpen = customOpenProp ?? customOpenState;
  const modelFilters = modelFiltersProp ?? modelFiltersState;
  const machineFilters = machineFiltersProp ?? machineFiltersState;
  const modelBreakdown = modelBreakdownProp ?? modelBreakdownState;
  const machineBreakdown = machineBreakdownProp ?? machineBreakdownState;
  const range = useMemo(
    () => resolveSpendingRange(rangeProp, period, customRange, now),
    [customRange, now, period, rangeProp],
  );
  const { rangeTotals, modelReport, machineReport } = useMemo(
    () =>
      resolveSpendingReports({
        events,
        range,
        catalogs,
        modelFilters,
        machineFilters,
        modelBreakdown,
        machineBreakdown,
        modelReportProp,
        machineReportProp,
        kpiTotalsProp,
      }),
    [
      catalogs,
      events,
      kpiTotalsProp,
      machineBreakdown,
      machineFilters,
      machineReportProp,
      modelBreakdown,
      modelFilters,
      modelReportProp,
      range,
    ],
  );

  const handlePeriodChange = createSpendingPeriodChangeHandler({
    isProduction,
    customRange,
    now,
    onPeriodChange,
    onCustomOpenChange,
    setPeriodState,
    setCustomOpenState,
    setCustomRangeState,
  });

  return {
    period,
    customRange: customRange ?? range,
    customOpen,
    modelFilters,
    machineFilters,
    modelBreakdown,
    machineBreakdown,
    range,
    rangeTotals,
    modelReport,
    machineReport,
    handlePeriodChange,
    setCustomOpen: bindSpendingField(setCustomOpenState, isProduction, onCustomOpenChange),
    setCustomRange: (next: SpendingDateRange) => {
      onCustomRangeChange?.(next);
      if (!isProduction) {
        setCustomRangeState(next);
        setPeriodState("custom");
      }
    },
    setModelFilters: bindSpendingField(setModelFiltersState, isProduction, onModelFiltersChange),
    setMachineFilters: bindSpendingField(setMachineFiltersState, isProduction, onMachineFiltersChange),
    setModelBreakdown: bindSpendingField(setModelBreakdownState, isProduction, onModelBreakdownChange),
    setMachineBreakdown: bindSpendingField(setMachineBreakdownState, isProduction, onMachineBreakdownChange),
  };
}
