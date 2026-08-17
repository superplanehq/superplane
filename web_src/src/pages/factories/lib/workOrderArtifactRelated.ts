import type { WorkOrderArtifactLike } from "./workOrderArtifact";

// The API and timeline both carry artifacts with `data` typed as
// `unknown`, but the resolver only needs `{ type, data }`. Do the
// narrowing once at the call site so consumers can pass the array
// straight through.
export function toWorkOrderArtifactLikes(
  artifacts: ReadonlyArray<{ type?: string | null; data?: unknown }> | undefined,
): WorkOrderArtifactLike[] {
  return (artifacts ?? []).map((artifact) => ({
    type: artifact.type ?? undefined,
    data: toArtifactDataRecord(artifact.data),
  }));
}

/**
 * Narrows the API's `data?: unknown` into the string-keyed shape every
 * artifact consumer expects, without lying about non-object payloads.
 */
export function toArtifactDataRecord(data: unknown): Record<string, unknown> | undefined {
  return data && typeof data === "object" ? (data as Record<string, unknown>) : undefined;
}
