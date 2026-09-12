import type { OrganizationsOrganizationCreditGrant } from "@/api-client";

/** Storybook payload for GET /organizations/{id}/credit-grants. */
export const DEFAULT_CREDIT_GRANTS: OrganizationsOrganizationCreditGrant[] = [
  {
    id: "grant-welcome",
    kind: "welcome",
    amountCents: "5000",
    createdAt: "2026-08-01T12:00:00.000Z",
    expiresAt: "2026-09-22T12:00:00.000Z",
  },
];

export const MIXED_CREDIT_GRANTS: OrganizationsOrganizationCreditGrant[] = [
  {
    id: "grant-refund",
    kind: "topup_refund",
    amountCents: "-500",
    polarOrderId: "ord_100",
    createdAt: "2026-08-20T15:00:00.000Z",
  },
  {
    id: "grant-topup",
    kind: "topup",
    amountCents: "2500",
    polarOrderId: "ord_100",
    createdAt: "2026-08-18T10:00:00.000Z",
  },
  {
    id: "grant-admin",
    kind: "admin",
    amountCents: "1500",
    note: "Support grant",
    actorName: "Ada",
    createdAt: "2026-08-10T09:00:00.000Z",
  },
  {
    id: "grant-welcome",
    kind: "welcome",
    amountCents: "5000",
    createdAt: "2026-08-01T12:00:00.000Z",
    expiresAt: "2026-09-22T12:00:00.000Z",
  },
];
