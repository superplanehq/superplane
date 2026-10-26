import type { OrganizationsHostedCreditProduct } from "@/api-client";

import { formatUsdCents } from "../../lib/workOrderUsage";

export function hostedCreditPackCents(pack: OrganizationsHostedCreditProduct): number | null {
  if (!pack.id) return null;
  const cents = Number(pack.amountCents ?? 0);
  if (!Number.isSafeInteger(cents) || cents < 0) return null;
  return cents;
}

export function fixedHostedCreditPackCents(packs: OrganizationsHostedCreditProduct[]): number[] {
  const amounts = new Set<number>();
  for (const pack of packs) {
    const cents = hostedCreditPackCents(pack);
    if (cents !== null && cents > 0) amounts.add(cents);
  }
  return [...amounts].sort((left, right) => left - right);
}

export function hasCustomHostedCreditPack(packs: OrganizationsHostedCreditProduct[]): boolean {
  return packs.some((pack) => hostedCreditPackCents(pack) === 0);
}

export function hasPurchasableHostedCreditPack(packs: OrganizationsHostedCreditProduct[]): boolean {
  return packs.some((pack) => hostedCreditPackCents(pack) !== null);
}

export function hostedCreditPackButtonLabel(cents: number): string {
  return cents % 100 === 0 ? `$${cents / 100}` : formatUsdCents(cents);
}
