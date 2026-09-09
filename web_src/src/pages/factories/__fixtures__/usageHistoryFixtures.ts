import type { FactoriesWorkOrderRunUsageRow } from "@/api-client";

import {
  ARNOLD_USER,
  HOUR_AGO,
  STORYBOOK_ME_USER_EMAIL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
} from "./factoryPageIds";
import { OPEN_WORK_ORDER, RUNNING_WORK_ORDER } from "./factoryPageWorkOrders";

export const BYOK_USAGE_HISTORY_ROW: FactoriesWorkOrderRunUsageRow = {
  workOrderExecutionId: "exec-usage-byok",
  workOrderId: OPEN_WORK_ORDER.id,
  workOrderNumber: OPEN_WORK_ORDER.number,
  workOrderKey: OPEN_WORK_ORDER.key,
  title: OPEN_WORK_ORDER.title,
  lastOccurredAt: HOUR_AGO,
  userId: STORYBOOK_ME_USER_ID,
  userName: STORYBOOK_ME_USER_NAME,
  userEmail: STORYBOOK_ME_USER_EMAIL,
  totalTokens: "22000",
  durationSeconds: "90",
  costCents: "3",
  hostedCostCents: "0",
  byokCostCents: "3",
  usedByok: true,
  models: ["anthropic/claude-sonnet-4-6"],
  machineTypes: ["e1-large-amd64"],
};

export const HOSTED_USAGE_HISTORY_ROW: FactoriesWorkOrderRunUsageRow = {
  workOrderExecutionId: "exec-usage-hosted",
  workOrderId: RUNNING_WORK_ORDER.id,
  workOrderNumber: RUNNING_WORK_ORDER.number,
  workOrderKey: RUNNING_WORK_ORDER.key,
  title: RUNNING_WORK_ORDER.title,
  lastOccurredAt: TWO_HOURS_AGO,
  userId: ARNOLD_USER.id,
  userName: ARNOLD_USER.name,
  userEmail: ARNOLD_USER.email,
  totalTokens: "1800",
  durationSeconds: "12",
  costCents: "150",
  hostedCostCents: "150",
  byokCostCents: "0",
  usedByok: false,
  models: ["anthropic/claude-sonnet-4-6"],
  machineTypes: ["e1-standard-amd64"],
};

export const DEFAULT_USAGE_HISTORY_ROWS: FactoriesWorkOrderRunUsageRow[] = [
  BYOK_USAGE_HISTORY_ROW,
  HOSTED_USAGE_HISTORY_ROW,
];

export function usageHistoryRows(count: number, seed: FactoriesWorkOrderRunUsageRow = HOSTED_USAGE_HISTORY_ROW) {
  return Array.from({ length: count }, (_, index) => ({
    ...seed,
    workOrderExecutionId: `${seed.workOrderExecutionId}-${index + 1}`,
    workOrderNumber: String(Number(seed.workOrderNumber ?? 1) + index),
    workOrderKey: `RF-${Number(seed.workOrderNumber ?? 1) + index}`,
    title: `${seed.title ?? "Task"} ${index + 1}`,
  }));
}
