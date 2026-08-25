import { getApiErrorMessage } from "@/lib/errors";

import { uniqueWorkspaceName } from "./workspaceNames";

const FACTORY_NAME_ALREADY_EXISTS = "factory with the same name already exists";
const MAX_NAME_RETRY_ATTEMPTS = 20;

export function isFactoryNameAlreadyExistsError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(FACTORY_NAME_ALREADY_EXISTS);
}

export type SaveWorkspaceNameDeps<T> = {
  /** Preferred name, for example the name derived from the repository. */
  name: string;
  /** Names already known to be in use, used to pick the first candidate. */
  takenNames?: Iterable<string>;
  save: (name: string) => Promise<T>;
};

/**
 * Saves a workspace with a free name. The organization list can be stale, so a
 * name rejected by the API is added to the taken names and the suffix counts up.
 */
export async function saveWithFreeWorkspaceName<T>(deps: SaveWorkspaceNameDeps<T>): Promise<T> {
  const taken = new Set([...(deps.takenNames ?? [])].map((name) => name.trim()).filter(Boolean));
  let name = uniqueWorkspaceName(deps.name, taken);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt += 1) {
    try {
      return await deps.save(name);
    } catch (error) {
      if (!isFactoryNameAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(name);
      name = uniqueWorkspaceName(deps.name, taken);
    }
  }

  throw new Error("Failed to find a free workspace name");
}
