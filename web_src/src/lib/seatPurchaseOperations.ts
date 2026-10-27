// A seat purchase operation is one confirmed purchase request and its
// operation key. The browser keeps it until a server answer for that key
// gives a known outcome. A retry of the same request, also after a page
// refresh or in another tab, then sends the same key and the server replays
// the stored purchase instead of charging again.
//
// Version 2 records also keep the subscription and the billing period of
// the confirmed preview. The server charges a new operation only for that
// context. Version 1 records have no such context: their period start was
// the coverage start. They are kept as legacy operations that can only
// replay a purchase that the server already recorded.

import type { OrganizationsPreviewSeatPurchaseResponse } from "@/api-client";

const STORAGE_VERSION = 3;
const PERIOD_STORAGE_VERSION = 2;
const LEGACY_STORAGE_VERSION = 1;
const STORAGE_PREFIX = "superplane:seat-purchase-operation:";

export type SeatPurchaseRequest = {
  additionalSeats: number;
  expectedPurchasedSeats: number;
};

/** The seat-removal version that the confirmed preview showed. */
export type SeatRemovalBinding = {
  seatChangeVersion: string;
  pendingRemovalId: string;
  pendingRemovalRevision: number;
};

/** The subscription, the billing period, and the seat removal that the confirmed preview showed. */
export type SeatPurchaseBinding = {
  subscriptionId: string;
  periodStart: string;
  periodEnd: string;
  /** Absent on a purchase recorded before seat-removal confirmation. */
  removal?: SeatRemovalBinding;
};

export type SeatPurchaseOperation = SeatPurchaseRequest & {
  organizationId: string;
  operationKey: string;
  confirmedAt: string;
  /** Undefined for a legacy operation from version 1 storage. */
  binding?: SeatPurchaseBinding;
};

type StoredSeatPurchaseOperation = SeatPurchaseOperation & { version: number };

/**
 * Returns the subscription and the billing period that a preview priced, or
 * undefined when the preview does not give all of them. The preview period
 * start is the coverage start, so it is not part of the binding.
 */
export function previewSeatPurchaseBinding(
  preview: OrganizationsPreviewSeatPurchaseResponse,
): SeatPurchaseBinding | undefined {
  const { subscriptionId, billingPeriodStart, periodEnd, seatChangeVersion } = preview;
  if (!subscriptionId || !billingPeriodStart || !periodEnd || typeof seatChangeVersion !== "string") return undefined;
  return {
    subscriptionId,
    periodStart: billingPeriodStart,
    periodEnd,
    removal: {
      seatChangeVersion,
      pendingRemovalId: preview.pendingRemovalId ?? "",
      pendingRemovalRevision: preview.pendingRemovalRevision ?? 0,
    },
  };
}

// The operations that this tab confirmed but could not save. Keys that other
// tabs saved stay only in storage, so a key that another tab retires after a
// failed write in this tab does not come back from memory.
const unsavedOperations = new Map<string, SeatPurchaseOperation[]>();
// A stale storage collection must not restore a key retired in this tab.
const retiredOperationKeys = new Map<string, Set<string>>();

function storageKey(organizationId: string) {
  return `${STORAGE_PREFIX}${organizationId}`;
}

function nonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value !== "";
}

function isBinding(value: unknown): value is SeatPurchaseBinding {
  if (!value || typeof value !== "object") return false;
  const record = value as StoredRecord;
  return (
    nonEmptyString(record.subscriptionId) && nonEmptyString(record.periodStart) && nonEmptyString(record.periodEnd)
  );
}

function isRemoval(value: unknown): value is SeatRemovalBinding {
  if (!value || typeof value !== "object") return false;
  const record = value as StoredRecord;
  return (
    typeof record.seatChangeVersion === "string" &&
    typeof record.pendingRemovalId === "string" &&
    Number.isSafeInteger(record.pendingRemovalRevision)
  );
}

type StoredRecord = Record<string, unknown>;

function isStoredRequest(
  record: StoredRecord,
  organizationId: string,
): record is StoredRecord & Omit<SeatPurchaseOperation, "binding"> {
  return (
    record.organizationId === organizationId &&
    nonEmptyString(record.operationKey) &&
    Number.isSafeInteger(record.additionalSeats) &&
    Number.isSafeInteger(record.expectedPurchasedSeats) &&
    typeof record.confirmedAt === "string"
  );
}

function parseOperation(value: unknown, organizationId: string): SeatPurchaseOperation | undefined {
  if (!value || typeof value !== "object") return undefined;
  const record = value as StoredRecord;
  if (!isStoredRequest(record, organizationId)) return undefined;
  const { operationKey, additionalSeats, expectedPurchasedSeats, confirmedAt } = record;
  const operation: SeatPurchaseOperation = {
    organizationId,
    operationKey,
    additionalSeats,
    expectedPurchasedSeats,
    confirmedAt,
  };
  if (record.version === STORAGE_VERSION && isBinding(record.binding) && isRemoval(record.binding.removal)) {
    const { subscriptionId, periodStart, periodEnd, removal } = record.binding;
    return { ...operation, binding: { subscriptionId, periodStart, periodEnd, removal } };
  }
  if (record.version === PERIOD_STORAGE_VERSION && isBinding(record.binding)) {
    const { subscriptionId, periodStart, periodEnd } = record.binding;
    return { ...operation, binding: { subscriptionId, periodStart, periodEnd } };
  }
  // A legacy period start is a coverage start, so it is not a binding.
  if (record.version === LEGACY_STORAGE_VERSION) return operation;
  return undefined;
}

function parseOperations(raw: string | null, organizationId: string): SeatPurchaseOperation[] {
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    // Keep the single-operation format from earlier clients readable.
    const records: unknown[] = Array.isArray(parsed) ? parsed : [parsed];
    return records
      .map((record) => parseOperation(record, organizationId))
      .filter((operation): operation is SeatPurchaseOperation => operation !== undefined);
  } catch {
    return [];
  }
}

/** Returns all unresolved purchase operations of the organization. */
export function loadSeatPurchaseOperations(organizationId: string): SeatPurchaseOperation[] {
  if (!organizationId) return [];
  const operations = new Map(
    (unsavedOperations.get(organizationId) ?? []).map((operation) => [operation.operationKey, operation]),
  );
  try {
    const stored = parseOperations(window.localStorage.getItem(storageKey(organizationId)), organizationId);
    for (const operation of stored) operations.set(operation.operationKey, operation);
  } catch {
    // Storage can be unavailable while this tab still holds unresolved keys.
  }
  const retired = retiredOperationKeys.get(organizationId);
  return [...operations.values()].filter((operation) => !retired?.has(operation.operationKey));
}

/**
 * Writes the complete collection of the organization. The collection
 * includes the unsaved operations of this tab, so a successful write saves
 * them too. A failed write keeps them unsaved.
 */
function saveSeatPurchaseOperations(organizationId: string, operations: SeatPurchaseOperation[]) {
  const stored: StoredSeatPurchaseOperation[] = operations.map((operation) => ({
    version: operation.binding?.removal
      ? STORAGE_VERSION
      : operation.binding
        ? PERIOD_STORAGE_VERSION
        : LEGACY_STORAGE_VERSION,
    ...operation,
  }));
  try {
    if (stored.length === 0) window.localStorage.removeItem(storageKey(organizationId));
    else window.localStorage.setItem(storageKey(organizationId), JSON.stringify(stored));
    unsavedOperations.delete(organizationId);
  } catch {
    // Storage can be full or blocked. The unsaved operations stay in memory.
  }
}

function keepUnsaved(organizationId: string, operations: SeatPurchaseOperation[]) {
  if (operations.length === 0) unsavedOperations.delete(organizationId);
  else unsavedOperations.set(organizationId, operations);
}

function sameRequest(operation: SeatPurchaseOperation, request: SeatPurchaseRequest) {
  return (
    operation.additionalSeats === request.additionalSeats &&
    operation.expectedPurchasedSeats === request.expectedPurchasedSeats
  );
}

/**
 * Returns the operation for a confirmed request. The same request reuses the
 * unresolved operation with its original key and binding, so the server can
 * replay it. Another request gets a new key with the binding of the confirmed
 * preview, which is stored before the request is sent. Earlier unresolved
 * requests keep their keys until their own outcomes are known.
 */
export function confirmSeatPurchaseOperation(
  organizationId: string,
  request: SeatPurchaseRequest,
  binding: SeatPurchaseBinding,
): SeatPurchaseOperation {
  const operations = loadSeatPurchaseOperations(organizationId);
  const stored = operations.find((operation) => sameRequest(operation, request));
  if (stored) return stored;
  const operation: SeatPurchaseOperation = {
    organizationId,
    operationKey: crypto.randomUUID(),
    ...request,
    binding: { ...binding },
    confirmedAt: new Date().toISOString(),
  };
  keepUnsaved(organizationId, [...(unsavedOperations.get(organizationId) ?? []), operation]);
  saveSeatPurchaseOperations(organizationId, [...operations, operation]);
  return operation;
}

/**
 * Forgets the operation after the server gave a known outcome for its key.
 * An operation with another key, for example from another tab, stays.
 */
export function retireSeatPurchaseOperation({
  organizationId,
  operationKey,
}: Pick<SeatPurchaseOperation, "organizationId" | "operationKey">) {
  const operations = loadSeatPurchaseOperations(organizationId);
  const retired = retiredOperationKeys.get(organizationId) ?? new Set<string>();
  retired.add(operationKey);
  retiredOperationKeys.set(organizationId, retired);
  const unsaved = unsavedOperations.get(organizationId) ?? [];
  keepUnsaved(
    organizationId,
    unsaved.filter((operation) => operation.operationKey !== operationKey),
  );
  const remaining = operations.filter((operation) => operation.operationKey !== operationKey);
  if (remaining.length === operations.length) return;
  saveSeatPurchaseOperations(organizationId, remaining);
}
