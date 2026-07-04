import type { CanvasesCanvasVersion } from "@/api-client";

export function draftBranchName(version: CanvasesCanvasVersion): string {
  return version.metadata?.branchName ?? "";
}

export function draftVersionId(version: CanvasesCanvasVersion): string {
  return version.metadata?.id ?? "";
}

export function draftDisplayName(version: CanvasesCanvasVersion): string {
  return version.metadata?.displayName || draftBranchName(version) || "Draft";
}

export function draftOwnerId(version: CanvasesCanvasVersion): string | undefined {
  return version.metadata?.owner?.id;
}

export function draftOwnerName(version: CanvasesCanvasVersion): string {
  return version.metadata?.owner?.name || "Unknown";
}

export function draftUpdatedAt(version: CanvasesCanvasVersion): string | undefined {
  return version.metadata?.updatedAt || version.metadata?.createdAt;
}
