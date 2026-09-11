import type { Automation, SoftwareFactory, WorkOrder, WorkOrderEvent } from "../factoryTypes";

export const paymentsFactory: SoftwareFactory = {
  id: "factory-payments",
  name: "Payments Factory",
  description:
    "Delegated implementation work for the payments platform. Approved Work Orders are implemented and delivered as pull requests.",
};

export const implementationAutomation: Automation = {
  id: "automation-implementation",
  name: "Implementation pipeline",
  description: "Implements approved Work Orders, runs checks, and opens a pull request.",
  state: "running",
  runningCount: 2,
  queuedCount: 3,
  lastRunAt: "2026-07-30T15:20:00Z",
};

export const verificationAutomation: Automation = {
  id: "automation-verification",
  name: "Verification pipeline",
  description: "Reviews implementation output, reruns required checks, and records the final outcome.",
  state: "idle",
  runningCount: 0,
  queuedCount: 0,
  lastRunAt: "2026-07-30T14:42:00Z",
};

export const workOrders: WorkOrder[] = [
  {
    id: "wo-refund-test",
    title: "Add refund reconciliation test",
    description: "Cover the refund reconciliation path before the next release. Keep the fixture provider-agnostic.",
    state: "draft",
    createdByUserId: "user-darko",
    createdByName: "Darko",
    createdAt: "2026-07-30T15:48:00Z",
    updatedAt: "2026-07-30T15:48:00Z",
    automations: [
      { id: implementationAutomation.id, name: implementationAutomation.name, state: "planned" },
      { id: verificationAutomation.id, name: verificationAutomation.name, state: "planned" },
    ],
  },
  {
    id: "wo-webhook",
    title: "Preserve idempotency keys on retries",
    description: "Keep the original idempotency key when a failed payment request is queued for another attempt.",
    state: "ready",
    createdByUserId: "user-maya",
    createdByName: "Maya Chen",
    createdAt: "2026-07-30T15:10:00Z",
    updatedAt: "2026-07-30T16:30:00Z",
    automations: [
      { id: implementationAutomation.id, name: implementationAutomation.name, state: "planned" },
      { id: verificationAutomation.id, name: verificationAutomation.name, state: "planned" },
    ],
  },
  {
    id: "wo-unsigned-webhook",
    title: "Reject unsigned payment webhooks",
    description: "Return 401 when the payment signature header is missing instead of processing the payload.",
    state: "running",
    createdByUserId: "user-maya",
    createdByName: "Maya Chen",
    createdAt: "2026-07-30T14:30:00Z",
    updatedAt: "2026-07-30T16:22:00Z",
    automations: [
      { id: implementationAutomation.id, name: implementationAutomation.name, state: "running" },
      { id: verificationAutomation.id, name: verificationAutomation.name, state: "planned" },
    ],
  },
  {
    id: "wo-rounding",
    title: "Fix settlement rounding",
    description: "Round once per settlement so the exported total matches the ledger.",
    state: "successful",
    createdByUserId: "user-darko",
    createdByName: "Darko",
    automations: [
      { id: implementationAutomation.id, name: implementationAutomation.name, state: "done" },
      { id: verificationAutomation.id, name: verificationAutomation.name, state: "done" },
    ],
    primaryPullRequest: {
      repository: "acme/payments-api",
      number: 1843,
      url: "https://github.com/acme/payments-api/pull/1843",
    },
    createdAt: "2026-07-29T10:00:00Z",
    updatedAt: "2026-07-30T13:16:00Z",
  },
  {
    id: "wo-ledger",
    title: "Backfill missing merchant categories",
    description: "Fill category codes for ledger records imported before the field existed.",
    state: "unsuccessful",
    createdByUserId: "user-maya",
    createdByName: "Maya Chen",
    createdAt: "2026-07-28T09:00:00Z",
    updatedAt: "2026-07-29T17:42:00Z",
    automations: [
      { id: implementationAutomation.id, name: implementationAutomation.name, state: "done" },
      { id: verificationAutomation.id, name: verificationAutomation.name, state: "failed" },
    ],
  },
];

export const draftEvents: WorkOrderEvent[] = [
  {
    id: "event-created",
    kind: "created",
    summary: "Work Order created",
    actor: "Darko",
    occurredAt: "2026-07-30T15:48:00Z",
  },
];

const settlementPullRequest = {
  repository: "acme/payments-api",
  number: 1843,
  url: "https://github.com/acme/payments-api/pull/1843",
};

export const successfulEvents: WorkOrderEvent[] = [
  {
    id: "event-created",
    kind: "created",
    summary: "Work Order created",
    actor: "Darko",
    occurredAt: "2026-07-29T10:00:00Z",
  },
  {
    id: "event-approved",
    kind: "approved",
    summary: "Approved and moved to ready",
    actor: "Darko",
    occurredAt: "2026-07-29T10:12:00Z",
  },
  {
    id: "event-started",
    kind: "started",
    summary: "Implementation pipeline picked up the work",
    actor: "Implementation pipeline",
    occurredAt: "2026-07-29T10:13:00Z",
    detail: "The Automation created a branch, changed the settlement calculation, and ran the repository checks.",
  },
  {
    id: "event-pr",
    kind: "pull-request",
    summary: "Pull request opened",
    actor: "Implementation pipeline",
    occurredAt: "2026-07-30T12:54:00Z",
    pullRequest: settlementPullRequest,
  },
  {
    id: "event-outcome",
    kind: "outcome",
    summary: "Marked successful",
    actor: "Implementation pipeline",
    occurredAt: "2026-07-30T13:16:00Z",
    outcome: "successful",
  },
];
