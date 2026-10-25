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

// Holds operations when the browser blocks storage, so a retry in the same
// page still reuses the key.
const memoryOperations = new Map<string, SeatPurchaseOperation>();

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

function parseOperation(raw: string | null, organizationId: string): SeatPurchaseOperation | undefined {
  if (!raw) return undefined;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!isStoredOperation(parsed, organizationId)) return undefined;
    return {
      organizationId: parsed.organizationId,
      operationKey: parsed.operationKey,
      additionalSeats: parsed.additionalSeats,
      expectedPurchasedSeats: parsed.expectedPurchasedSeats,
      periodStart: parsed.periodStart,
      periodEnd: parsed.periodEnd,
      confirmedAt: parsed.confirmedAt,
    };
  } catch {
    return undefined;
  }
}

/** Returns the unresolved purchase operation of the organization, if any. */
export function loadSeatPurchaseOperation(organizationId: string): SeatPurchaseOperation | undefined {
  if (!organizationId) return undefined;
  try {
    return parseOperation(window.localStorage.getItem(storageKey(organizationId)), organizationId);
  } catch {
    return memoryOperations.get(organizationId);
  }
}

function saveSeatPurchaseOperation(operation: SeatPurchaseOperation) {
  const stored: StoredSeatPurchaseOperation = { version: STORAGE_VERSION, ...operation };
  try {
    window.localStorage.setItem(storageKey(operation.organizationId), JSON.stringify(stored));
    memoryOperations.delete(operation.organizationId);
  } catch {
    memoryOperations.set(operation.organizationId, operation);
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
 * new key, which is stored before the request is sent.
 */
export function confirmSeatPurchaseOperation(
  organizationId: string,
  request: SeatPurchaseRequest,
  period: SeatPurchasePeriod,
): SeatPurchaseOperation {
  const stored = loadSeatPurchaseOperation(organizationId);
  if (stored && sameRequest(stored, request)) return stored;
  const operation: SeatPurchaseOperation = {
    organizationId,
    operationKey: crypto.randomUUID(),
    ...request,
    periodStart: period.periodStart,
    periodEnd: period.periodEnd,
    confirmedAt: new Date().toISOString(),
  };
  saveSeatPurchaseOperation(operation);
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
  if (memoryOperations.get(organizationId)?.operationKey === operationKey) memoryOperations.delete(organizationId);
  try {
    if (loadSeatPurchaseOperation(organizationId)?.operationKey !== operationKey) return;
    window.localStorage.removeItem(storageKey(organizationId));
  } catch {
    // The memory record is already gone. A blocked storage has no record.
  }
}
