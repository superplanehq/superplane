import type { CanvasStagingRecord } from "./types";

export type { CanvasStagingRecord } from "./types";
export { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./types";

const DB_NAME = "superplane-canvas-staging";
const DB_VERSION = 1;
const STORE_NAME = "staging";

function stagingKey(canvasId: string, branch: string): string {
  return `${canvasId}:${branch}`;
}

export async function openStagingDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onerror = () => {
      reject(request.error ?? new Error("Failed to open canvas staging database"));
    };

    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "id" });
      }
    };

    request.onsuccess = () => {
      resolve(request.result);
    };
  });
}

type StoredStagingRecord = CanvasStagingRecord & { id: string };

export async function getStaging(canvasId: string, branch: string): Promise<CanvasStagingRecord | null> {
  const db = await openStagingDB();

  return new Promise((resolve, reject) => {
    const transaction = db.transaction(STORE_NAME, "readonly");
    const store = transaction.objectStore(STORE_NAME);
    const request = store.get(stagingKey(canvasId, branch));

    request.onerror = () => {
      db.close();
      reject(request.error ?? new Error("Failed to read canvas staging"));
    };

    request.onsuccess = () => {
      db.close();
      const record = request.result as StoredStagingRecord | undefined;
      if (!record) {
        resolve(null);
        return;
      }

      resolve({
        canvasId: record.canvasId,
        branch: record.branch,
        baseHeadSha: record.baseHeadSha,
        files: record.files,
        deletedPaths: record.deletedPaths,
        updatedAt: record.updatedAt,
      });
    };
  });
}

export async function putStaging(record: CanvasStagingRecord): Promise<void> {
  const db = await openStagingDB();

  return new Promise((resolve, reject) => {
    const transaction = db.transaction(STORE_NAME, "readwrite");
    const store = transaction.objectStore(STORE_NAME);
    const payload: StoredStagingRecord = {
      ...record,
      id: stagingKey(record.canvasId, record.branch),
      updatedAt: record.updatedAt || Date.now(),
    };
    const request = store.put(payload);

    request.onerror = () => {
      db.close();
      reject(request.error ?? new Error("Failed to write canvas staging"));
    };

    request.onsuccess = () => {
      db.close();
      resolve();
    };
  });
}

export async function clearStaging(canvasId: string, branch: string): Promise<void> {
  const db = await openStagingDB();

  return new Promise((resolve, reject) => {
    const transaction = db.transaction(STORE_NAME, "readwrite");
    const store = transaction.objectStore(STORE_NAME);
    const request = store.delete(stagingKey(canvasId, branch));

    request.onerror = () => {
      db.close();
      reject(request.error ?? new Error("Failed to clear canvas staging"));
    };

    request.onsuccess = () => {
      db.close();
      resolve();
    };
  });
}

export function hasStagingFiles(record: CanvasStagingRecord | null | undefined): boolean {
  if (!record) {
    return false;
  }

  return Object.keys(record.files).length > 0 || (record.deletedPaths?.length ?? 0) > 0;
}

/** Whether IndexedDB staging still applies to the current branch HEAD (lenient while HEAD is loading). */
export function stagingMatchesBranchHead(
  record: CanvasStagingRecord | null | undefined,
  branchHeadSha?: string,
): boolean {
  if (!hasStagingFiles(record)) {
    return false;
  }

  if (!branchHeadSha) {
    return true;
  }

  if (!record?.baseHeadSha) {
    return true;
  }

  return record.baseHeadSha === branchHeadSha;
}
