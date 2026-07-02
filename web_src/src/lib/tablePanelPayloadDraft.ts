export interface PayloadDraftEntry {
  rowId: string;
  path: string;
  template: string;
}

export function draftToPayloadShape(entries: PayloadDraftEntry[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const entry of entries) {
    if (!entry.path) continue;
    out[entry.path] = entry.template;
  }
  return out;
}

/**
 * Serialize a draft list back into the persisted `Record<string, string>`.
 * Trailing blank rows are dropped, and entries with an empty path are skipped
 * so the on-disk shape stays clean.
 */
export function draftToPayload(entries: PayloadDraftEntry[]): Record<string, string> | undefined {
  const shape = draftToPayloadShape(entries);
  return Object.keys(shape).length > 0 ? shape : undefined;
}

export function payloadShallowEqual(a: Record<string, string>, b: Record<string, string>): boolean {
  const ak = Object.keys(a);
  const bk = Object.keys(b);
  if (ak.length !== bk.length) return false;
  for (const k of ak) {
    if (!(k in b) || a[k] !== b[k]) return false;
  }
  return true;
}
