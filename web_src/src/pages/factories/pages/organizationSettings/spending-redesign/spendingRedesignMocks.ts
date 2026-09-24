import {
  ACME_ONBOARDING_FACTORY_ID,
  ARNOLD_USER,
  EMPTY_FACTORY_ID,
  OPERATOR_USER,
  PRIMARY_FACTORY_ID,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../../__fixtures__/factoryPageIds";
import type { SpendingCatalogs, SpendingUsageEvent } from "./spendingRedesignLib";
import { modelKey } from "./spendingRedesignLib";

/** Frozen "now" so Storybook charts and tests stay on the same days. */
export const SPENDING_REDESIGN_NOW = new Date("2026-09-03T12:00:00.000Z");

const DAY_MS = 24 * 60 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;

export const SPENDING_WORKSPACES = [
  { id: PRIMARY_FACTORY_ID, label: "Semaphore" },
  { id: ACME_ONBOARDING_FACTORY_ID, label: "Acme onboarding" },
  { id: EMPTY_FACTORY_ID, label: "SuperPlane" },
] as const;

export const SPENDING_USERS = [
  { id: STORYBOOK_ME_USER_ID, label: STORYBOOK_ME_USER_NAME },
  { id: ARNOLD_USER.id, label: ARNOLD_USER.name },
  { id: REVIEWER_USER.id, label: REVIEWER_USER.name },
  { id: OPERATOR_USER.id, label: OPERATOR_USER.name },
] as const;

const MODELS = [
  { provider: "anthropic", model: "claude-sonnet-4-6" },
  { provider: "openai", model: "gpt-4o" },
  { provider: "anthropic", model: "claude-opus-4-6" },
] as const;

const MACHINES = ["e1-large-amd64", "e1-tiny-amd64"] as const;

export const SPENDING_CATALOGS: SpendingCatalogs = {
  users: [...SPENDING_USERS],
  workspaces: [...SPENDING_WORKSPACES],
  models: MODELS.map((item) => ({
    id: modelKey(item.provider, item.model),
    label: item.model,
  })),
  machines: MACHINES.map((machineType) => ({ id: machineType, label: machineType })),
};

export interface SpendingCreditSnapshot {
  remainingCreditCents: number;
  grantTotalCents: number;
  superplaneGrantCents: number;
  purchasedCreditCents: number;
  hostedBilledCents: number;
  remainingCreditWarning: boolean;
  billingEnabled: boolean;
  hasBillingCustomer: boolean;
}

export const SPENDING_CREDIT: SpendingCreditSnapshot = {
  remainingCreditCents: 4124,
  grantTotalCents: 15000,
  superplaneGrantCents: 5000,
  purchasedCreditCents: 10000,
  hostedBilledCents: 10876,
  remainingCreditWarning: false,
  billingEnabled: true,
  hasBillingCustomer: true,
};

export const SPENDING_CREDIT_WARNING: SpendingCreditSnapshot = {
  ...SPENDING_CREDIT,
  remainingCreditCents: 0,
  hostedBilledCents: 15000,
  remainingCreditWarning: true,
};

function at(daysAgo: number, hoursAgo = 0): string {
  return new Date(SPENDING_REDESIGN_NOW.getTime() - daysAgo * DAY_MS - hoursAgo * HOUR_MS).toISOString();
}

function modelEvent(row: {
  id: string;
  daysAgo: number;
  hoursAgo: number;
  factoryId: string;
  userId: string;
  provider: string;
  model: string;
  tokens: number;
  costCents: number;
  fundingSource?: "hosted" | "byok";
}): SpendingUsageEvent {
  return {
    id: row.id,
    occurredAt: at(row.daysAgo, row.hoursAgo),
    factoryId: row.factoryId,
    userId: row.userId,
    provider: row.provider,
    model: row.model,
    usageKind: "model",
    fundingSource: row.fundingSource ?? "hosted",
    machineType: "",
    totalTokens: row.tokens,
    durationSeconds: 0,
    costCents: row.costCents,
  };
}

function computeEvent(row: {
  id: string;
  daysAgo: number;
  hoursAgo: number;
  factoryId: string;
  userId: string;
  machineType: string;
  durationSeconds: number;
  costCents: number;
}): SpendingUsageEvent {
  return {
    id: row.id,
    occurredAt: at(row.daysAgo, row.hoursAgo),
    factoryId: row.factoryId,
    userId: row.userId,
    provider: "runner",
    model: row.machineType,
    usageKind: "compute",
    fundingSource: "hosted",
    machineType: row.machineType,
    totalTokens: 0,
    durationSeconds: row.durationSeconds,
    costCents: row.costCents,
  };
}

/**
 * Ledger shaped like `workspace_usage_events`: model rows plus runner VM
 * rows, scoped to factory, work-order owner, provider/model, and machine type.
 */
export const SPENDING_LEDGER: SpendingUsageEvent[] = [
  ...[
    {
      id: "e1",
      daysAgo: 0,
      hoursAgo: 2,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 18400,
      costCents: 620,
    },
    {
      id: "e2",
      daysAgo: 0,
      hoursAgo: 4,
      factoryId: PRIMARY_FACTORY_ID,
      userId: ARNOLD_USER.id,
      provider: "openai",
      model: "gpt-4o",
      tokens: 7200,
      costCents: 256,
    },
    {
      id: "e4",
      daysAgo: 1,
      hoursAgo: 6,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: REVIEWER_USER.id,
      provider: "anthropic",
      model: "claude-opus-4-6",
      tokens: 9100,
      costCents: 840,
    },
    {
      id: "e6",
      daysAgo: 2,
      hoursAgo: 8,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 22100,
      costCents: 745,
    },
    {
      id: "e7",
      daysAgo: 2,
      hoursAgo: 9,
      factoryId: PRIMARY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      provider: "openai",
      model: "gpt-4o",
      tokens: 5400,
      costCents: 190,
      fundingSource: "byok" as const,
    },
    {
      id: "e9",
      daysAgo: 4,
      hoursAgo: 11,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 13200,
      costCents: 410,
    },
    {
      id: "e10",
      daysAgo: 5,
      hoursAgo: 3,
      factoryId: PRIMARY_FACTORY_ID,
      userId: REVIEWER_USER.id,
      provider: "anthropic",
      model: "claude-opus-4-6",
      tokens: 6400,
      costCents: 590,
    },
    {
      id: "e12",
      daysAgo: 8,
      hoursAgo: 4,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 19800,
      costCents: 680,
    },
    {
      id: "e13",
      daysAgo: 9,
      hoursAgo: 1,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: ARNOLD_USER.id,
      provider: "openai",
      model: "gpt-4o",
      tokens: 8100,
      costCents: 275,
      fundingSource: "byok" as const,
    },
    {
      id: "e15",
      daysAgo: 13,
      hoursAgo: 2,
      factoryId: PRIMARY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 15600,
      costCents: 512,
    },
    {
      id: "e16",
      daysAgo: 16,
      hoursAgo: 8,
      factoryId: EMPTY_FACTORY_ID,
      userId: REVIEWER_USER.id,
      provider: "openai",
      model: "gpt-4o",
      tokens: 2100,
      costCents: 74,
    },
    {
      id: "e18",
      daysAgo: 21,
      hoursAgo: 3,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-opus-4-6",
      tokens: 11200,
      costCents: 980,
    },
    {
      id: "e19",
      daysAgo: 24,
      hoursAgo: 10,
      factoryId: PRIMARY_FACTORY_ID,
      userId: ARNOLD_USER.id,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 17600,
      costCents: 598,
    },
    {
      id: "e21",
      daysAgo: 33,
      hoursAgo: 6,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: REVIEWER_USER.id,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 14800,
      costCents: 488,
    },
    {
      id: "e22",
      daysAgo: 40,
      hoursAgo: 2,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "openai",
      model: "gpt-4o",
      tokens: 9300,
      costCents: 310,
    },
    {
      id: "e24",
      daysAgo: 55,
      hoursAgo: 1,
      factoryId: PRIMARY_FACTORY_ID,
      userId: ARNOLD_USER.id,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 20500,
      costCents: 702,
    },
    {
      id: "e25",
      daysAgo: 70,
      hoursAgo: 4,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-opus-4-6",
      tokens: 8700,
      costCents: 760,
    },
    {
      id: "e27",
      daysAgo: 110,
      hoursAgo: 3,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 16400,
      costCents: 555,
    },
    {
      id: "e28",
      daysAgo: 140,
      hoursAgo: 8,
      factoryId: EMPTY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      provider: "openai",
      model: "gpt-4o",
      tokens: 1800,
      costCents: 62,
    },
    {
      id: "e30",
      daysAgo: 220,
      hoursAgo: 5,
      factoryId: PRIMARY_FACTORY_ID,
      userId: REVIEWER_USER.id,
      provider: "anthropic",
      model: "claude-sonnet-4-6",
      tokens: 19100,
      costCents: 640,
    },
    {
      id: "e31",
      daysAgo: 280,
      hoursAgo: 6,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      provider: "openai",
      model: "gpt-4o",
      tokens: 7600,
      costCents: 248,
    },
  ].map(modelEvent),
  ...[
    {
      id: "e3",
      daysAgo: 0,
      hoursAgo: 3,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      machineType: "e1-large-amd64",
      durationSeconds: 2400,
      costCents: 133,
    },
    {
      id: "e5",
      daysAgo: 1,
      hoursAgo: 5,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: REVIEWER_USER.id,
      machineType: "e1-tiny-amd64",
      durationSeconds: 900,
      costCents: 17,
    },
    {
      id: "e8",
      daysAgo: 3,
      hoursAgo: 2,
      factoryId: PRIMARY_FACTORY_ID,
      userId: ARNOLD_USER.id,
      machineType: "e1-large-amd64",
      durationSeconds: 1800,
      costCents: 99,
    },
    {
      id: "e11",
      daysAgo: 6,
      hoursAgo: 7,
      factoryId: PRIMARY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      machineType: "e1-tiny-amd64",
      durationSeconds: 1500,
      costCents: 28,
    },
    {
      id: "e14",
      daysAgo: 11,
      hoursAgo: 6,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      machineType: "e1-large-amd64",
      durationSeconds: 3200,
      costCents: 178,
    },
    {
      id: "e17",
      daysAgo: 18,
      hoursAgo: 5,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: ARNOLD_USER.id,
      machineType: "e1-tiny-amd64",
      durationSeconds: 700,
      costCents: 12,
    },
    {
      id: "e20",
      daysAgo: 27,
      hoursAgo: 4,
      factoryId: PRIMARY_FACTORY_ID,
      userId: STORYBOOK_ME_USER_ID,
      machineType: "e1-large-amd64",
      durationSeconds: 2100,
      costCents: 116,
    },
    {
      id: "e23",
      daysAgo: 47,
      hoursAgo: 9,
      factoryId: PRIMARY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      machineType: "e1-tiny-amd64",
      durationSeconds: 1100,
      costCents: 21,
    },
    {
      id: "e26",
      daysAgo: 88,
      hoursAgo: 7,
      factoryId: PRIMARY_FACTORY_ID,
      userId: REVIEWER_USER.id,
      machineType: "e1-large-amd64",
      durationSeconds: 2600,
      costCents: 144,
    },
    {
      id: "e29",
      daysAgo: 180,
      hoursAgo: 2,
      factoryId: ACME_ONBOARDING_FACTORY_ID,
      userId: ARNOLD_USER.id,
      machineType: "e1-tiny-amd64",
      durationSeconds: 800,
      costCents: 14,
    },
    {
      id: "e32",
      daysAgo: 320,
      hoursAgo: 4,
      factoryId: PRIMARY_FACTORY_ID,
      userId: OPERATOR_USER.id,
      machineType: "e1-large-amd64",
      durationSeconds: 1900,
      costCents: 105,
    },
  ].map(computeEvent),
];
