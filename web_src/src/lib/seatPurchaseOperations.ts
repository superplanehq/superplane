// A seat purchase operation is one confirmed purchase request and its
// operation key. The browser keeps it until a server answer for that key
// gives a known outcome. A retry of the same request, also after a page
// refresh or in another tab, then sends the same key and the server replays
// the stored purchase instead of charging again.

const STORAGE_VERSION = 1;
const STORAGE_PREFIX = "superplane:seat-purchase-operation:";

export type SeatPurchaseRequest = {
  additionalSeats: number;
  expectedPurchasedSeats: number;
};

export type SeatPurchasePeriod = {
  periodStart?: string;
  periodEnd?: string;
};

export type SeatPurchaseOperation = SeatPurchaseRequest &
  SeatPurchasePeriod & {
    organizationId: string;
    operationKey: string;
    confirmedAt: string;
  };

type StoredSeatPurchaseOperation = SeatPurchaseOperation & { version: number };

// Holds the current collection when a storage write fails. It takes precedence
// over storage reads, which can still return an older collection.
const memoryOperations = new Map<string, SeatPurchaseOperation[]>();

function storageKey(organizationId: string) {
  return `${STORAGE_PREFIX}${organizationId}`;
}

function optionalString(value: unknown) {
  return value === undefined || typeof value === "string";
}

function isStoredOperation(value: unknown, organizationId: string): value is StoredSeatPurchaseOperation {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    record.version === STORAGE_VERSION &&
    record.organizationId === organizationId &&
    typeof record.operationKey === "string" &&
    record.operationKey !== "" &&
    Number.isSafeInteger(record.additionalSeats) &&
    Number.isSafeInteger(record.expectedPurchasedSeats) &&
    typeof record.confirmedAt === "string" &&
    optionalString(record.periodStart) &&
    optionalString(record.periodEnd)
  );
}

function parseOperations(raw: string | null, organizationId: string): SeatPurchaseOperation[] {
  if (!raw) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    // Keep the single-operation format from earlier clients readable.
    const records: unknown[] = Array.isArray(parsed) ? parsed : [parsed];
    return records
      .filter((record): record is StoredSeatPurchaseOperation => isStoredOperation(record, organizationId))
      .map(({ version: _version, ...operation }) => operation);
  } catch {
    return [];
  }
}

/** Returns all unresolved purchase operations of the organization. */
export function loadSeatPurchaseOperations(organizationId: string): SeatPurchaseOperation[] {
  if (!organizationId) return [];
  const fallback = memoryOperations.get(organizationId);
  if (fallback) return fallback;
  try {
    return parseOperations(window.localStorage.getItem(storageKey(organizationId)), organizationId);
  } catch {
    return [];
  }
}

function saveSeatPurchaseOperations(organizationId: string, operations: SeatPurchaseOperation[]) {
  const stored: StoredSeatPurchaseOperation[] = operations.map((operation) => ({
    version: STORAGE_VERSION,
    ...operation,
  }));
  try {
    if (stored.length === 0) window.localStorage.removeItem(storageKey(organizationId));
    else window.localStorage.setItem(storageKey(organizationId), JSON.stringify(stored));
    memoryOperations.delete(organizationId);
  } catch {
    memoryOperations.set(organizationId, operations);
  }
}

function sameRequest(operation: SeatPurchaseOperation, request: SeatPurchaseRequest) {
  return (
    operation.additionalSeats === request.additionalSeats &&
    operation.expectedPurchasedSeats === request.expectedPurchasedSeats
  );
}

/**
 * Returns the operation for a confirmed request. The same request reuses the
 * unresolved operation and keeps its original period. Another request gets a
 * new key, which is stored before the request is sent. Earlier unresolved
 * requests keep their keys until their own outcomes are known.
 */
export function confirmSeatPurchaseOperation(
  organizationId: string,
  request: SeatPurchaseRequest,
  period: SeatPurchasePeriod,
): SeatPurchaseOperation {
  const operations = loadSeatPurchaseOperations(organizationId);
  const stored = operations.find((operation) => sameRequest(operation, request));
  if (stored) return stored;
  const operation: SeatPurchaseOperation = {
    organizationId,
    operationKey: crypto.randomUUID(),
    ...request,
    periodStart: period.periodStart,
    periodEnd: period.periodEnd,
    confirmedAt: new Date().toISOString(),
  };
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
  const remaining = operations.filter((operation) => operation.operationKey !== operationKey);
  if (remaining.length === operations.length) return;
  saveSeatPurchaseOperations(organizationId, remaining);
}
