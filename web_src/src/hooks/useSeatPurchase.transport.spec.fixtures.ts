import type { OrganizationsSeatPurchase } from "@/api-client";

type PurchaseBody = {
  additionalSeats?: number;
  expectedPurchasedSeats?: number;
  operationKey?: string;
  confirmedSubscriptionId?: string;
  confirmedPeriodStart?: string;
  confirmedPeriodEnd?: string;
  confirmedSeatChangeVersion?: string;
  confirmedPendingRemovalId?: string;
  confirmedPendingRemovalRevision?: number;
};
type StoredOperation = { counts: string; binding: string | undefined; purchase: OrganizationsSeatPurchase };
// A number is an HTTP status that the server returns before it processes the request.
type Fault = "lost request" | "lost response" | number;

export const DAY_MS = 24 * 60 * 60 * 1000;

function bindingOf({ confirmedSubscriptionId, confirmedPeriodStart, confirmedPeriodEnd }: PurchaseBody) {
  if (!confirmedSubscriptionId || !confirmedPeriodStart || !confirmedPeriodEnd) return undefined;
  return `${confirmedSubscriptionId}|${confirmedPeriodStart}|${confirmedPeriodEnd}`;
}

// FakeBillingServer follows the purchase contract of the server: the
// operation key with the same request and binding replays the stored
// purchase, also after a renewal. A legacy purchase without a binding
// replays by its counts. The same key with another request gets HTTP 412.
// A missing key, or a new operation without the confirmed subscription and
// period, gets HTTP 400. A stale binding or baseline gets HTTP 409.
export class FakeBillingServer {
  purchasedSeats = 4;
  subscriptionId = "sub_team";
  periodStart = new Date(Date.now() - 10 * DAY_MS).toISOString();
  periodEnd = new Date(Date.now() + 20 * DAY_MS).toISOString();
  lastSeatPurchase: OrganizationsSeatPurchase | undefined;
  charges = 0;
  billingUnavailable = false;
  previewOmitsBinding = false;
  purchaseRequests: PurchaseBody[] = [];
  private nextFault: Fault | undefined;
  private operations = new Map<string, StoredOperation>();

  failNextPurchase(fault: Fault) {
    this.nextFault = fault;
  }

  renew() {
    this.periodStart = this.periodEnd;
    this.periodEnd = new Date(Date.parse(this.periodEnd) + 30 * DAY_MS).toISOString();
    this.purchasedSeats = 4;
  }

  /** The binding that a preview shows now, in the request field names. */
  confirmedContext() {
    return {
      confirmedSubscriptionId: this.subscriptionId,
      confirmedPeriodStart: this.periodStart,
      confirmedPeriodEnd: this.periodEnd,
      confirmedSeatChangeVersion: "0",
      confirmedPendingRemovalId: "",
      confirmedPendingRemovalRevision: 0,
    };
  }

  /** Records a purchase that a client before the binding started and Polar charged. */
  recordLegacyPurchase(operationKey: string, additionalSeats: number, expectedPurchasedSeats: number) {
    this.charge(
      operationKey,
      { counts: `${additionalSeats}:${expectedPurchasedSeats}`, binding: undefined },
      additionalSeats,
    );
  }

  billing() {
    return {
      purchasedSeats: this.purchasedSeats,
      purchasableSeats: 10,
      currentPeriodStart: this.periodStart,
      currentPeriodEnd: this.periodEnd,
      lastSeatPurchase: this.lastSeatPurchase,
    };
  }

  fetch: typeof fetch = async (input) => {
    const request = input instanceof Request ? input : new Request(input);
    const path = new URL(request.url).pathname;
    if (path.endsWith("/seats/preview")) {
      const { additionalSeats = 0 } = (await request.json()) as PurchaseBody;
      return this.respond(200, {
        purchasedSeats: this.purchasedSeats,
        targetSeats: this.purchasedSeats + additionalSeats,
        proratedChargeCents: "500",
        // The coverage start. It is not the billing-period start.
        periodStart: new Date().toISOString(),
        periodEnd: this.periodEnd,
        ...(this.previewOmitsBinding
          ? {}
          : {
              subscriptionId: this.subscriptionId,
              billingPeriodStart: this.periodStart,
              seatChangeVersion: "0",
              pendingRemovalId: "",
              pendingRemovalRevision: 0,
              renewalSeats: this.purchasedSeats + additionalSeats,
              cancelsPendingRemoval: false,
            }),
      });
    }
    if (path.endsWith("/seats/purchase")) return this.purchase((await request.json()) as PurchaseBody);
    if (path.endsWith("/billing/sync") || path.endsWith("/billing")) {
      if (this.billingUnavailable) throw new TypeError("Failed to fetch");
      return this.respond(200, this.billing());
    }
    throw new Error(`Unexpected billing request: ${path}`);
  };

  private purchase(body: PurchaseBody) {
    this.purchaseRequests.push(body);
    const fault = this.nextFault;
    this.nextFault = undefined;
    if (fault === "lost request") throw new TypeError("Failed to fetch");
    if (typeof fault === "number") return this.respond(fault, { message: "The purchase was not processed." });
    const response = this.process(body);
    if (fault === "lost response") throw new TypeError("Failed to fetch");
    return response;
  }

  private process(body: PurchaseBody) {
    const { additionalSeats, expectedPurchasedSeats, operationKey } = body;
    if (!operationKey || !additionalSeats) return this.respond(400, { message: "The request is not valid." });
    const counts = `${additionalSeats}:${expectedPurchasedSeats}`;
    const binding = bindingOf(body);
    const stored = this.operations.get(operationKey);
    if (stored) {
      const matches = stored.counts === counts && (stored.binding === undefined || stored.binding === binding);
      if (!matches) return this.respond(412, { message: "The operation key belongs to another purchase." });
      return this.respond(200, { purchase: stored.purchase, billing: this.billing() });
    }
    if (!binding || body.confirmedSeatChangeVersion === undefined) {
      return this.respond(400, { message: "Preview the purchase again and confirm it." });
    }
    if (binding !== bindingOf(this.confirmedContext()))
      return this.respond(409, {
        message: "The subscription or billing period changed. Review the new price and confirm again.",
      });
    if (expectedPurchasedSeats !== this.purchasedSeats)
      return this.respond(409, { message: "The seat count changed. Preview this purchase again." });
    const purchase = this.charge(operationKey, { counts, binding }, additionalSeats);
    return this.respond(200, { purchase, billing: this.billing() });
  }

  private charge(operationKey: string, fingerprint: Omit<StoredOperation, "purchase">, additionalSeats: number) {
    this.charges += 1;
    const purchase = {
      id: `purchase-${this.charges}`,
      status: "succeeded",
      fromSeats: this.purchasedSeats,
      toSeats: this.purchasedSeats + additionalSeats,
      createdAt: new Date().toISOString(),
    };
    this.purchasedSeats = purchase.toSeats;
    this.lastSeatPurchase = purchase;
    this.operations.set(operationKey, { ...fingerprint, purchase });
    return purchase;
  }

  private respond(status: number, body: unknown) {
    return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  }
}
