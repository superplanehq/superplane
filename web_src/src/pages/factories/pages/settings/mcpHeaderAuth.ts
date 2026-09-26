const BEARER_PREFIX = /^bearer\s+/i;
export const MCP_HEADER_SECRET_KEY = "token";

export function bearerAuthorizationValue(token: string): string {
  const trimmed = token.trim();
  if (BEARER_PREFIX.test(trimmed)) {
    return `Bearer ${trimmed.replace(BEARER_PREFIX, "")}`;
  }
  return `Bearer ${trimmed}`;
}

export function catalogHeaderSecretName(serverName: string, uniquePart: string): string {
  return `${serverName}-mcp-${uniquePart}`;
}
