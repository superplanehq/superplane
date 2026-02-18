export interface CostAllocationPayload {
  name?: string;
  start?: string;
  end?: string;
  cpuCost?: number;
  gpuCost?: number;
  ramCost?: number;
  pvCost?: number;
  networkCost?: number;
  totalCost?: number;
  threshold?: number;
  window?: string;
  aggregate?: string;
}

export interface OnCostExceedsThresholdConfiguration {
  window?: string;
  aggregate?: string;
  threshold?: number;
  filter?: string;
}

export interface GetCostAllocationConfiguration {
  window?: string;
  aggregate?: string;
  filter?: string;
}
