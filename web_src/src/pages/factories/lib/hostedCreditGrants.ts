import { formatUsdCents } from "./workOrderUsage";

export const CREDIT_GRANT_KIND_WELCOME = "welcome";
export const CREDIT_GRANT_KIND_ADMIN = "admin";
export const CREDIT_GRANT_KIND_INCLUDED = "included";
export const CREDIT_GRANT_KIND_TOPUP = "topup";
export const CREDIT_GRANT_KIND_TOPUP_REFUND = "topup_refund";

export function creditGrantSourceLabel(kind: string | undefined): string {
  switch (kind) {
    case CREDIT_GRANT_KIND_WELCOME:
      return "Trial";
    case CREDIT_GRANT_KIND_ADMIN:
      return "SuperPlane grant";
    case CREDIT_GRANT_KIND_INCLUDED:
      return "Included";
    case CREDIT_GRANT_KIND_TOPUP:
      return "Top-up";
    case CREDIT_GRANT_KIND_TOPUP_REFUND:
      return "Refund";
    default:
      return "Credit";
  }
}

export function creditGrantDetails(
  grant: {
    kind?: string;
    note?: string;
    actorName?: string;
    polarOrderId?: string;
    expiresAt?: string;
  },
  now: Date = new Date(),
): string {
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
  if ((grant.kind === CREDIT_GRANT_KIND_TOPUP || grant.kind === CREDIT_GRANT_KIND_TOPUP_REFUND) && orderId) {
    parts.push(`Order ${orderId}`);
  }
  const expiry = welcomeCreditExpiryDetail(grant.kind, grant.expiresAt, now);
  if (expiry) {
    parts.push(expiry);
  }
  return parts.join(" · ");
}

export function welcomeCreditExpiryDetail(
  kind: string | undefined,
  expiresAt: string | undefined,
  now: Date = new Date(),
): string | null {
  if (kind !== CREDIT_GRANT_KIND_WELCOME && kind !== CREDIT_GRANT_KIND_INCLUDED && kind !== CREDIT_GRANT_KIND_TOPUP) {
    return null;
  }
  if (!expiresAt) {
    return null;
  }
  const parsed = new Date(expiresAt);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  const dateLabel = parsed.toLocaleDateString();
  if (parsed.getTime() <= now.getTime()) {
    return `Expired on ${dateLabel}`;
  }
  return `Expires on ${dateLabel}`;
}

export function welcomeCreditUnusedExpiryNote(args: {
  remainingCents: number;
  purchasedCents: number;
  welcomeCreditExpiresAt?: string;
  now?: Date;
}): string | null {
  if (args.purchasedCents > 0 || !args.welcomeCreditExpiresAt) {
    return null;
  }
  const parsed = new Date(args.welcomeCreditExpiresAt);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  if (parsed.getTime() > (args.now ?? new Date()).getTime()) {
    return null;
  }
  return "Welcome credit expired. Unused free credit no longer pays for SuperPlane-hosted runs.";
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

export function hostedCreditBalanceWarning(
  remainingCents: number,
  remainingCreditWarning: boolean,
  welcomeExpired = false,
): string | null {
  if (remainingCents <= 0 && welcomeExpired) {
    return "Welcome credit expired. SuperPlane-hosted runs cannot start.";
  }
  if (remainingCents <= 0) {
    return "Hosted credit is empty. SuperPlane-hosted runs cannot start.";
  }
  if (!remainingCreditWarning) {
    return null;
  }
  return "Hosted credit is low.";
}
