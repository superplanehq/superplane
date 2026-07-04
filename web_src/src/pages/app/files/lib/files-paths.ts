import type { PendingFileChange } from "../types";

export function nextUntitledPath(paths: Set<string>): string {
  if (!paths.has("untitled.txt")) return "untitled.txt";

  let index = 1;
  while (paths.has(`untitled-${index}.txt`)) {
    index += 1;
  }

  return `untitled-${index}.txt`;
}

export function buildRenderableTreePaths(repositoryPaths: string[], changes: PendingFileChange[]): string[] {
  const deletedPaths = new Set(changes.filter((change) => change.type === "deleted").map((change) => change.path));
  const paths = repositoryPaths.filter((path) => !deletedPaths.has(path));

  for (const change of changes) {
    if (change.type === "deleted") continue;
    if (getPathValidationError([...paths, change.path])) continue;
    paths.push(change.path);
  }

  return Array.from(new Set(paths)).sort();
}

export function buildFinalRepositoryPaths(repositoryPaths: string[], changes: PendingFileChange[]): string[] {
  const paths = new Set(repositoryPaths);

  for (const change of changes) {
    if (change.type === "deleted") {
      paths.delete(change.path);
      continue;
    }

    paths.add(change.path);
  }

  return Array.from(paths).sort();
}

export function getPathValidationError(paths: string[]): string | undefined {
  const filePaths = new Set<string>();
  const directoryPaths = new Set<string>();

  for (const path of paths) {
    const normalizedPath = normalizeFilePath(path);
    if (!normalizedPath || normalizedPath.endsWith("/")) {
      return "File path is required.";
    }

    const segments = normalizedPath.split("/");
    if (segments.some((segment) => segment === "" || segment === "." || segment === ".." || segment === ".git")) {
      return `Invalid file path "${path}".`;
    }

    for (let index = 1; index < segments.length; index += 1) {
      const directoryPath = segments.slice(0, index).join("/");
      if (filePaths.has(directoryPath)) {
        return `Path "${normalizedPath}" collides with existing file "${directoryPath}".`;
      }
      directoryPaths.add(directoryPath);
    }

    if (directoryPaths.has(normalizedPath)) {
      return `Path "${normalizedPath}" collides with an existing directory.`;
    }

    if (filePaths.has(normalizedPath)) {
      return `Path "${normalizedPath}" is already used.`;
    }

    filePaths.add(normalizedPath);
  }
}

export function normalizeFilePath(path: string): string {
  return path.trim().replace(/\\/g, "/").replace(/^\/+/, "");
}
