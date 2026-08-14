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
  [
    "Stop duplicate charges and failed retries on checkout.",
    "Work orders in this mission run one by one through the refund lines. Start with the open ledger reconciliation. Then run the retry patch. Then close the failed cases.",
    "Do not dispatch a new work order until the previous one leaves Running. If a work order fails, keep it in this mission and assign a follow-up work order. Do not move the failed work order to a different mission.",
    "Reviewers check the ledger report and the retry logs before they mark a work order complete. Operators watch the running line and stop a dispatch that repeats a charge.",
    "The mission is complete when every work order is completed or failed, and checkout no longer creates duplicate refund entries.",
  ].join("\n\n"),
  ALEX,
);

export const REFUNDS_V2_MISSION = mission(
  "mission-refunds-v2",
  "Refunds v2",
  [
    "Ship the next refund schema and retry path.",
    "Track every work order that belongs to this outcome in one place. The schema work order adds the reason enum. The retry work order makes replays idempotent. The closed work order records the failed audit case.",
    "Do not merge the schema change until the verifier line passes. If the implementer line fails, keep the work order in this mission and open a follow-up for the mismatch.",
    "Operators assign the schema work first. Reviewers confirm the enum values match the audit report. Do not start the retry path until the schema work order is complete.",
    "The mission is complete when the new schema is in use and every work order in this mission is completed or failed.",
  ].join("\n\n"),
  JAMIE,
);

export const SEEDED_MISSIONS: FactoryMission[] = [
  CHECKOUT_RELIABILITY_MISSION,
  REFUNDS_V2_MISSION,
  mission(
    "mission-invoice-matching",
    "Invoice matching",
    [
      "Match inbound invoices to payout records.",
      "Each work order covers one unmatched batch. Start with the oldest unpaid invoices. Then match the current week. Then close the leftover exceptions.",
      "Do not mark a work order complete until every invoice in the batch has a payout or a documented exception. Keep failed matches in this mission.",
      "The mission is complete when the unpaid invoice queue is empty and every work order is completed or failed.",
    ].join("\n\n"),
    STORYBOOK,
  ),
  mission(
    "mission-payout-delays",
    "Payout delays",
    [
      "Find and clear delayed payout batches.",
      "Work orders in this mission inspect one delayed batch at a time. Confirm the bank file. Then replay the payout. Then record the delay reason.",
      "Do not dispatch a replay until the bank file matches the ledger. If a replay fails, keep the work order in this mission.",
      "The mission is complete when no payout batch is delayed and every work order is completed or failed.",
    ].join("\n\n"),
    ALEX,
  ),
  mission(
    "mission-chargeback-intake",
    "Chargeback intake",
    [
      "Triage new chargebacks before they reach a line.",
      "Each work order reviews one intake window. Sort new chargebacks. Assign a reason. Then send ready cases to a refund line.",
      "Do not send a chargeback to a line until the reason is set. Keep rejected intake items in this mission for a follow-up work order.",
      "The mission is complete when the intake queue is empty and every work order is completed or failed.",
    ].join("\n\n"),
    JAMIE,
  ),
  mission(
    "mission-tax-receipts",
    "Tax receipts",
    [
      "Issue tax receipts for completed refunds.",
      "Work orders in this mission cover one receipt period. Collect completed refunds. Generate the receipts. Then send them to the customer archive.",
      "Do not issue a receipt for a refund that is still running. If generation fails, keep the work order in this mission.",
      "The mission is complete when every completed refund in the period has a receipt.",
    ].join("\n\n"),
    STORYBOOK,
  ),
  mission(
    "mission-wallet-top-up",
    "Wallet top-up",
    [
      "Fix failed wallet top-up retries.",
      "Each work order covers one failed top-up set. Confirm the wallet balance. Then replay the top-up. Then record the retry result.",
      "Do not replay a top-up that already succeeded. Keep failed replays in this mission and assign a follow-up work order.",
      "The mission is complete when failed top-ups are cleared and every work order is completed or failed.",
    ].join("\n\n"),
    ALEX,
  ),
  mission(
    "mission-subscription-renewals",
    "Subscription renewals",
    [
      "Track renewal work orders for the next cycle.",
      "Work orders in this mission cover one renewal cohort. Confirm the cycle dates. Then run the renewal line. Then close failed renewals.",
      "Do not start the next cohort until the current work order leaves Running. Keep failed renewals in this mission.",
      "The mission is complete when the cycle is closed and every work order is completed or failed.",
    ].join("\n\n"),
    JAMIE,
  ),
  mission(
    "mission-fraud-review",
    "Fraud review",
    [
      "Review flagged checkout sessions.",
      "Each work order reviews one flagged set. Confirm the risk score. Then release or hold the session. Then record the decision.",
      "Do not release a session until a reviewer confirms the score. Keep held sessions in this mission for a follow-up work order.",
      "The mission is complete when the flagged queue is empty and every work order is completed or failed.",
    ].join("\n\n"),
    STORYBOOK,
  ),
  mission(
    "mission-ledger-sync",
    "Ledger sync",
    [
      "Keep the refund ledger in sync with the bank feed.",
      "Work orders in this mission cover one sync window. Pull the bank feed. Match ledger rows. Then record unmatched rows.",
      "Do not close a work order while unmatched rows remain. Keep sync failures in this mission.",
      "The mission is complete when the ledger matches the bank feed and every work order is completed or failed.",
    ].join("\n\n"),
    ALEX,
  ),
  mission(
    "mission-customer-credits",
    "Customer credits",
    [
      "Apply store credits after a failed refund.",
      "Each work order covers one credit batch. Confirm the failed refund. Then apply the credit. Then notify the customer record.",
      "Do not apply a credit if the refund later completes. Keep failed credit applications in this mission.",
      "The mission is complete when every failed refund in the batch has a credit or a documented exception.",
    ].join("\n\n"),
    JAMIE,
  ),
  mission(
    "mission-settlement-reports",
    "Settlement reports",
    [
      "Close the weekly settlement report.",
      "Work orders in this mission cover one report week. Collect completed refunds. Build the report. Then archive the file.",
      "Do not close the week while a refund in that week is still running. Keep a failed report build in this mission.",
      "The mission is complete when the weekly report is archived and every work order is completed or failed.",
    ].join("\n\n"),
    STORYBOOK,
  ),
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
