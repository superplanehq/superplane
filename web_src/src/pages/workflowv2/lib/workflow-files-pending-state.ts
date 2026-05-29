import type { PendingFileChange } from "../workflow-files-types";

export function applyPendingContentUpdate(
  current: Record<string, PendingFileChange>,
  selectedPath: string,
  value: string,
  originalContent: string | undefined,
): Record<string, PendingFileChange> {
  const currentChange = current[selectedPath];
  if (currentChange?.type === "added") {
    return { ...current, [selectedPath]: { ...currentChange, content: value } };
  }

  if (originalContent === undefined) {
    return current;
  }

  if (value === originalContent) {
    const { [selectedPath]: _removed, ...remaining } = current;
    return remaining;
  }

  return {
    ...current,
    [selectedPath]: { type: "modified", path: selectedPath, content: value },
  };
}

export function applyPendingDelete(
  current: Record<string, PendingFileChange>,
  path: string,
): Record<string, PendingFileChange> {
  const currentChange = current[path];
  if (currentChange?.type === "added") {
    const { [path]: _removed, ...remaining } = current;
    return remaining;
  }

  return {
    ...current,
    [path]: { type: "deleted", path },
  };
}

export function mergeLoadedContentAfterPublish(
  current: Record<string, string>,
  pendingChanges: PendingFileChange[],
): Record<string, string> {
  const next = { ...current };
  for (const change of pendingChanges) {
    if (change.type === "deleted") {
      delete next[change.path];
      continue;
    }

    next[change.path] = change.content;
  }

  return next;
}
