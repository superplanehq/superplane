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
