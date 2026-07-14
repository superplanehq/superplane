export function buildAgentStagingAutoOpenKey(canvasId: string, message?: string): string {
  return `${canvasId}:${message ?? ""}`;
}

const openedAgentStagingKeys = new Set<string>();

export function claimAgentStagingAutoOpen(stagingKey: string): boolean {
  if (openedAgentStagingKeys.has(stagingKey)) {
    return false;
  }

  openedAgentStagingKeys.add(stagingKey);
  return true;
}

export function releaseAgentStagingAutoOpen(stagingKey: string): void {
  openedAgentStagingKeys.delete(stagingKey);
}
