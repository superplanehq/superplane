import { getApiErrorMessage } from "@/lib/errors";

import { uniqueWorkspaceKeyFromName } from "../../lib/workspaceKey";

const FACTORY_KEY_ALREADY_EXISTS = "workspace key already exists in this organization";
const MAX_KEY_RETRY_ATTEMPTS = 26;

export function isFactoryKeyAlreadyExistsError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(FACTORY_KEY_ALREADY_EXISTS);
}

export type SaveWorkspaceKeyDeps<T> = {
  /** Real workspace/repository name the key is derived from. */
  name: string;
  /** Keys already known to be in use, used to pick the first candidate. */
  takenKeys?: Iterable<string>;
  save: (key: string) => Promise<T>;
};

/**
 * Saves a workspace with a free key derived from its (real) name. The
 * organization's key list can be stale, so a key rejected by the API is
 * added to the taken set and the candidate walks to the next variant.
 */
export async function saveWithFreeWorkspaceKey<T>(deps: SaveWorkspaceKeyDeps<T>): Promise<T> {
  const taken = new Set([...(deps.takenKeys ?? [])].map((key) => key.trim().toLowerCase()).filter(Boolean));
  let key = uniqueWorkspaceKeyFromName(deps.name, taken);

  for (let attempt = 0; attempt < MAX_KEY_RETRY_ATTEMPTS; attempt += 1) {
    try {
      return await deps.save(key);
    } catch (error) {
      if (!isFactoryKeyAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(key);
      key = uniqueWorkspaceKeyFromName(deps.name, taken);
    }
  }

  throw new Error("Failed to find a free workspace key");
}
