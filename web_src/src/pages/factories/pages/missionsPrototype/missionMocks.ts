/**
 * Storybook-only missions. A work order belongs to one mission, or to none.
 * This map is not sent to the API and does not change live work-order payloads.
 */
export interface MissionAssignee {
  id: string;
  name: string;
}

export interface FactoryMission {
  id: string;
  name: string;
  description: string;
  assignee: MissionAssignee;
}

const ALEX: MissionAssignee = { id: "user-reviewer-alex", name: "Alex Reviewer" };
const JAMIE: MissionAssignee = { id: "user-operator-jamie", name: "Jamie Operator" };
const STORYBOOK: MissionAssignee = { id: "storybook-user", name: "Storybook User" };

function mission(id: string, name: string, description: string, assignee: MissionAssignee): FactoryMission {
  return { id, name, description, assignee };
}

export const CHECKOUT_RELIABILITY_MISSION = mission(
  "mission-checkout-reliability",
  "Checkout reliability",
  "Stop duplicate charges and failed retries on checkout. Work orders in this mission run one by one through the refund lines.",
  ALEX,
);

export const REFUNDS_V2_MISSION = mission(
  "mission-refunds-v2",
  "Refunds v2",
  "Ship the next refund schema and retry path. Track every work order that belongs to this outcome in one place.",
  JAMIE,
);

export const SEEDED_MISSIONS: FactoryMission[] = [
  CHECKOUT_RELIABILITY_MISSION,
  REFUNDS_V2_MISSION,
  mission("mission-invoice-matching", "Invoice matching", "Match inbound invoices to payout records.", STORYBOOK),
  mission("mission-payout-delays", "Payout delays", "Find and clear delayed payout batches.", ALEX),
  mission("mission-chargeback-intake", "Chargeback intake", "Triage new chargebacks before they reach a line.", JAMIE),
  mission("mission-tax-receipts", "Tax receipts", "Issue tax receipts for completed refunds.", STORYBOOK),
  mission("mission-wallet-top-up", "Wallet top-up", "Fix failed wallet top-up retries.", ALEX),
  mission(
    "mission-subscription-renewals",
    "Subscription renewals",
    "Track renewal work orders for the next cycle.",
    JAMIE,
  ),
  mission("mission-fraud-review", "Fraud review", "Review flagged checkout sessions.", STORYBOOK),
  mission("mission-ledger-sync", "Ledger sync", "Keep the refund ledger in sync with the bank feed.", ALEX),
  mission("mission-customer-credits", "Customer credits", "Apply store credits after a failed refund.", JAMIE),
  mission("mission-settlement-reports", "Settlement reports", "Close the weekly settlement report.", STORYBOOK),
];

/** Work-order id → mission id. Orders that are absent have no mission. */
export const missionByWorkOrderId: Record<string, string> = {
  "wo-open-refunds": CHECKOUT_RELIABILITY_MISSION.id,
  "wo-running-refunds": CHECKOUT_RELIABILITY_MISSION.id,
  "wo-closed-refunds": CHECKOUT_RELIABILITY_MISSION.id,
  "wo-open-refunds-schema": REFUNDS_V2_MISSION.id,
  "wo-failed-refunds": REFUNDS_V2_MISSION.id,
  "wo-closed-failed-refunds": REFUNDS_V2_MISSION.id,
};
