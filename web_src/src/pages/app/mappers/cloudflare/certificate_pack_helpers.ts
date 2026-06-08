/** Payload shape shared by Order and Delete Certificate Pack outputs (nested fields vary). */
export interface CertificatePackOutput {
  zoneId?: string;
  zoneName?: string;
  packId?: string;
  hosts?: string[];
  pack?: {
    id?: string;
    certificate_authority?: string;
    hosts?: string[];
    status?: string;
    type?: string;
    validation_method?: string;
    validity_days?: number;
  };
  deleted?: boolean;
}

export function certificatePackZoneLabel(result: CertificatePackOutput): string | undefined {
  const zoneName = result.zoneName?.trim();
  if (zoneName) return zoneName;
  const zoneId = result.zoneId?.trim();
  return zoneId || undefined;
}

/** Prefer top-level hosts (delete payload); otherwise nested pack.hosts (order payload). */
export function certificatePackHostsLabel(result: CertificatePackOutput): string | undefined {
  const joinHosts = (hosts: string[] | undefined) =>
    hosts
      ?.map((h) => h.trim())
      .filter(Boolean)
      .join(", ");
  const top = joinHosts(result.hosts);
  if (top) return top;
  const nested = joinHosts(result.pack?.hosts);
  return nested || undefined;
}

export function certificatePackId(result: CertificatePackOutput): string | undefined {
  return result.packId?.trim() || result.pack?.id?.trim() || undefined;
}
