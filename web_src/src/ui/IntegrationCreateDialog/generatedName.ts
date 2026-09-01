import { useMemo, useState } from "react";

import { getApiErrorMessage } from "@/lib/errors";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";

const MAX_NAME_ATTEMPTS = 20;

/**
 * Resolves the name a new connection gets.
 *
 * GitHub connections are named after the owner instead of by the user, so the
 * list of connections stays readable when a team connects several owners. Every
 * other integration keeps the name the user typed.
 */
export function useGeneratedIntegrationName(args: {
  isGitHub: boolean;
  typedName: string;
  owner: unknown;
  existingNames: Set<string>;
}) {
  /** Name the API accepted, which can differ from the first candidate. */
  const [createdName, setCreatedName] = useState<string | undefined>(undefined);

  const baseName = useMemo(() => {
    const owner = typeof args.owner === "string" ? args.owner.trim().toLowerCase() : "";
    return owner ? `github-${owner}` : "github";
  }, [args.owner]);

  const name = useMemo(() => {
    if (!args.isGitHub) return args.typedName;
    return createdName ?? getNextIntegrationName(baseName, args.existingNames);
  }, [args.existingNames, args.isGitHub, args.typedName, baseName, createdName]);

  return { name, baseName, setCreatedName };
}

export function isNameTakenError(error: unknown): boolean {
  return /already exists/i.test(getApiErrorMessage(error, ""));
}

/**
 * Creates an integration under a generated name.
 *
 * The list of taken names comes from a cached query, so it can miss a
 * connection that another tab or an abandoned setup left behind. A name
 * conflict is not something the user can act on when the name is generated,
 * so retry with the next suffix instead of showing the error.
 */
export async function createWithGeneratedName<T>(args: {
  baseName: string;
  takenNames: Set<string>;
  create: (name: string) => Promise<T>;
}): Promise<{ result: T; name: string }> {
  const taken = new Set(args.takenNames);
  let lastError: unknown;

  for (let attempt = 0; attempt < MAX_NAME_ATTEMPTS; attempt += 1) {
    const name = getNextIntegrationName(args.baseName, taken);
    try {
      return { result: await args.create(name), name };
    } catch (error) {
      if (!isNameTakenError(error)) throw error;
      lastError = error;
      taken.add(name);
    }
  }

  throw lastError;
}
