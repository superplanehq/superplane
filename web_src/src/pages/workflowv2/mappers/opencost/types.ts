export interface GetCostAllocationConfiguration {
  window?: string;
  aggregate?: string;
  step?: string;
  resolution?: string;
}

export interface CostAllocationPayload {
  window?: string;
  aggregate?: string;
  totalCost?: number;
  allocations?: AllocationEntry[];
}

export interface AllocationEntry {
  name?: string;
  cpuCost?: number;
  gpuCost?: number;
  ramCost?: number;
  pvCost?: number;
  networkCost?: number;
  totalCost?: number;
  start?: string;
  end?: string;
  minutes?: number;
  properties?: Record<string, string>;
}

export interface CostExceedsThresholdPayload {
  totalCost?: number;
  threshold?: number;
  window?: string;
  aggregate?: string;
  exceedingItems?: ExceedingItem[];
}

export interface ExceedingItem {
  name?: string;
  totalCost?: number;
  cpuCost?: number;
  gpuCost?: number;
  ramCost?: number;
  pvCost?: number;
}
