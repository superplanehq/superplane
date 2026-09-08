import { formatUsdCents } from "./workOrderUsage";

export const CREDIT_GRANT_KIND_WELCOME = "welcome";
export const CREDIT_GRANT_KIND_ADMIN = "admin";
export const CREDIT_GRANT_KIND_POLAR = "polar";
export const CREDIT_GRANT_KIND_POLAR_REFUND = "polar_refund";

export function creditGrantSourceLabel(kind: string | undefined): string {
  switch (kind) {
    case CREDIT_GRANT_KIND_WELCOME:
      return "Welcome credit";
    case CREDIT_GRANT_KIND_ADMIN:
      return "SuperPlane grant";
    case CREDIT_GRANT_KIND_POLAR:
      return "Purchased";
    case CREDIT_GRANT_KIND_POLAR_REFUND:
      return "Refund";
    default:
      return "Credit";
  }
}

export function creditGrantDetails(grant: {
  kind?: string;
  note?: string;
  actorName?: string;
  polarOrderId?: string;
}): string {
  const parts: string[] = [];
  const note = grant.note?.trim();
  if (note) {
    parts.push(note);
  }
  const actorName = grant.actorName?.trim();
  if (grant.kind === CREDIT_GRANT_KIND_ADMIN && actorName) {
    parts.push(`Granted by ${actorName}`);
  }
  const orderId = grant.polarOrderId?.trim();
  if ((grant.kind === CREDIT_GRANT_KIND_POLAR || grant.kind === CREDIT_GRANT_KIND_POLAR_REFUND) && orderId) {
    parts.push(`Order ${orderId}`);
  }
  return parts.join(" · ");
}

export function formatCreditGrantAmount(cents: number): string {
  const formatted = formatUsdCents(Math.abs(cents));
  if (cents > 0) {
    return `+${formatted}`;
  }
  if (cents < 0) {
    return `-${formatted}`;
  }
  return formatted;
}

export function hostedCreditBalanceWarning(remainingCents: number, remainingCreditWarning: boolean): string | null {
  if (remainingCents <= 0) {
    return "Hosted credit is empty. SuperPlane-hosted runs cannot start.";
  }
  if (!remainingCreditWarning) {
    return null;
  }
  return "Hosted credit is low.";
}
