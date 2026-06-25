// FirewallNodeMetadata is persisted on Update/Delete Firewall Rule nodes so the
// collapsed UI can show the targeted firewall rule name.
export interface FirewallNodeMetadata {
  firewallName?: string;
}

// firewallLastSegment returns the final path segment of a firewall rule or
// network reference (e.g. a selfLink), used to show a short name in the UI.
// Returns undefined for empty values or unresolved expressions.
export function firewallLastSegment(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes("{{")) return undefined;
  const idx = trimmed.lastIndexOf("/");
  return idx >= 0 ? trimmed.slice(idx + 1).replace(/[?#].*$/, "") : trimmed;
}
