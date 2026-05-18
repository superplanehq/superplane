/**
 * Canvas node metadata is stored as a Struct/map and may arrive as camelCase
 * or snake_case depending on serialization. Normalize reads here.
 */

function metadataRecord(metadata: unknown): Record<string, unknown> | undefined {
  if (!metadata || typeof metadata !== "object") return undefined;
  return metadata as Record<string, unknown>;
}

export function getCloudflarePoolName(metadata: unknown): string | undefined {
  const m = metadataRecord(metadata);
  if (!m) return undefined;
  const raw = m.poolName ?? m.pool_name;
  return typeof raw === "string" && raw.trim() ? raw.trim() : undefined;
}

export function getCloudflareMonitorId(metadata: unknown): string | undefined {
  const m = metadataRecord(metadata);
  if (!m) return undefined;
  const raw = m.monitorId ?? m.monitor_id;
  return typeof raw === "string" && raw.trim() ? raw.trim() : undefined;
}

export function getCloudflareMonitorDescription(metadata: unknown): string | undefined {
  const m = metadataRecord(metadata);
  if (!m) return undefined;
  const raw = m.monitorDescription ?? m.monitor_description;
  return typeof raw === "string" && raw.trim() ? raw.trim() : undefined;
}

/** Prefer Cloudflare monitor description when node metadata matches the monitor id (see Setup resolver). */
export function getCloudflareMonitorDisplayLabel(metadata: unknown, monitorId: string): string {
  const id = monitorId.trim();
  if (!id) return "-";

  const resolvedId = getCloudflareMonitorId(metadata);
  const resolvedDesc = getCloudflareMonitorDescription(metadata);
  if (resolvedId === id && resolvedDesc) {
    return resolvedDesc;
  }

  return id;
}

export function getCloudflareTunnelName(metadata: unknown): string | undefined {
  const m = metadataRecord(metadata);
  if (!m) return undefined;
  const raw = m.tunnelName ?? m.tunnel_name;
  return typeof raw === "string" && raw.trim() ? raw.trim() : undefined;
}
