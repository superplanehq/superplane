import type { OrganizationsSeatPurchase } from "@/api-client";

export function isSeatPurchasePending(purchase: OrganizationsSeatPurchase | undefined) {
  return purchase?.status === "pending" || purchase?.status === "awaiting_confirmation";
}

export function seatPurchaseMessage(purchase: OrganizationsSeatPurchase | undefined) {
  if (purchase?.status === "succeeded") return "Seats added.";
  return purchase?.message || null;
}

export function validAdditionalSeats(value: string, maximum: number | undefined) {
  if (!/^\d+$/.test(value)) return undefined;
  const count = Number(value);
  if (!Number.isSafeInteger(count) || count < 1 || (maximum !== undefined && count > maximum)) return undefined;
  return count;
}

export function recentSeatPurchase(purchase: OrganizationsSeatPurchase | undefined) {
  if (!purchase) return undefined;
  if (isSeatPurchasePending(purchase)) return purchase;
  const age = Date.now() - Date.parse(purchase.createdAt ?? "");
  return age >= 0 && age <= 24 * 60 * 60 * 1000 ? purchase : undefined;
}
