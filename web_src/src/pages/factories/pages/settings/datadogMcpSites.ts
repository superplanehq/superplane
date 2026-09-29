export type DatadogMCPSite = {
  id: string;
  label: string;
  url: string;
};

const DATADOG_MCP_SITE_DEFINITIONS = [
  { id: "datadoghq.com", label: "US1 (datadoghq.com)" },
  { id: "us3.datadoghq.com", label: "US3 (us3.datadoghq.com)" },
  { id: "us5.datadoghq.com", label: "US5 (us5.datadoghq.com)" },
  { id: "datadoghq.eu", label: "EU (datadoghq.eu)" },
  { id: "ap1.datadoghq.com", label: "AP1 (ap1.datadoghq.com)" },
  { id: "ap2.datadoghq.com", label: "AP2 (ap2.datadoghq.com)" },
  { id: "uk1.datadoghq.com", label: "UK1 (uk1.datadoghq.com)" },
] as const;

export const DATADOG_MCP_DEFAULT_SITE_ID = "datadoghq.com";

export function datadogMCPURL(siteId: string): string {
  if (siteId === "datadoghq.com") {
    return "https://mcp.datadoghq.com/v1/mcp";
  }
  if (siteId === "datadoghq.eu") {
    return "https://mcp.datadoghq.eu/v1/mcp";
  }
  return `https://mcp.${siteId}/v1/mcp`;
}

export const DATADOG_MCP_SITES: DatadogMCPSite[] = DATADOG_MCP_SITE_DEFINITIONS.map((site) => ({
  id: site.id,
  label: site.label,
  url: datadogMCPURL(site.id),
}));

export function datadogMCPSiteForURL(raw?: string): DatadogMCPSite | undefined {
  const host = datadogMCPHost(raw);
  if (!host) {
    return undefined;
  }
  return DATADOG_MCP_SITES.find((site) => new URL(site.url).host === host);
}

export function isDatadogMCPURL(raw?: string): boolean {
  return datadogMCPSiteForURL(raw) !== undefined;
}

function datadogMCPHost(raw?: string): string | undefined {
  const trimmed = raw?.trim();
  if (!trimmed) {
    return undefined;
  }
  try {
    const parsed = new URL(trimmed);
    if (parsed.protocol !== "https:") {
      return undefined;
    }
    const path = parsed.pathname.replace(/\/+$/, "");
    if (path !== "/v1/mcp") {
      return undefined;
    }
    return parsed.host.toLowerCase();
  } catch {
    return undefined;
  }
}
