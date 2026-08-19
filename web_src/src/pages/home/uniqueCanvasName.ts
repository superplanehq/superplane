import { getApiErrorMessage } from "@/lib/errors";

const CANVAS_NAME_ALREADY_EXISTS = "Canvas with the same name already exists";

/** Returns baseName, or "baseName (2)", "(3)", … until the name is free. */
export function uniqueCanvasName(baseName: string, existingNames: Iterable<string>): string {
  const base = baseName.trim() || "App";
  const taken = new Set([...existingNames].map((name) => name.trim()).filter((name): name is string => Boolean(name)));

  if (!taken.has(base)) {
    return base;
  }

  let suffix = 2;
  let candidate = `${base} (${suffix})`;
  while (taken.has(candidate)) {
    suffix += 1;
    candidate = `${base} (${suffix})`;
  }
  return candidate;
}

export function isCanvasNameAlreadyExistsError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(CANVAS_NAME_ALREADY_EXISTS);
}
