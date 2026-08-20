import type { FactoriesFactoryLine, FactoryLineStep } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { uniqueCanvasName } from "@/pages/home/uniqueCanvasName";

import { duplicateLineName } from "./lineCardActions";

const FACTORY_LINE_NAME_ALREADY_EXISTS = "factory line with the same name already exists";
const MAX_NAME_RETRY_ATTEMPTS = 20;

export function isFactoryLineNameAlreadyExistsError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(FACTORY_LINE_NAME_ALREADY_EXISTS);
}

export type DuplicateFactoryLineDeps = {
  line: FactoriesFactoryLine;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  /** Names already taken in the factory (used to pick "X copy", "X copy (2)", …). */
  existingNames?: Iterable<string>;
};

/**
 * Clones a factory line: same steps (apps, entrypoints, max parallelism), a
 * unique "{name} copy" name. Work-order history and metrics are never
 * copied — `CreateFactoryLine` always starts a line with a clean slate.
 */
export async function duplicateFactoryLine(deps: DuplicateFactoryLineDeps): Promise<FactoriesFactoryLine> {
  const preferredName = duplicateLineName(deps.line.name);
  const steps = deps.line.steps ?? [];
  const taken = new Set(
    [...(deps.existingNames ?? [])].map((name) => name.trim()).filter((name): name is string => Boolean(name)),
  );
  let name = uniqueCanvasName(preferredName, taken);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      return await deps.createLine({ name, steps });
    } catch (error) {
      if (!isFactoryLineNameAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(name);
      name = uniqueCanvasName(preferredName, taken);
    }
  }

  throw new Error("Failed to create line");
}
