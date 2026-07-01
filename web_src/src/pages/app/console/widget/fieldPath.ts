/**
 * Resolve a dot/bracket field path against a value tree.
 *
 * Examples:
 *   getValueAtPath({ a: { b: [{ c: 1 }] } }, "a.b[0].c") === 1
 *   getValueAtPath({ foo: { bar: "x" } }, "foo.bar") === "x"
 *   getValueAtPath(null, "a.b") === undefined
 *
 * Edge cases:
 *   - `[*]` is rejected by the caller — wildcards are handled by filter logic.
 *   - Numeric keys ("0") are accepted on objects (treated as property lookup).
 */
export function getValueAtPath(input: unknown, path: string): unknown {
  if (input == null || !path) return input;
  const segments = parsePath(path);
  let cursor: unknown = input;
  for (const segment of segments) {
    if (cursor == null) return undefined;
    if (Array.isArray(cursor)) {
      // Allow numeric index on arrays; reject string keys.
      const idx = Number(segment);
      if (!Number.isInteger(idx)) return undefined;
      cursor = cursor[idx];
      continue;
    }
    if (typeof cursor === "object") {
      cursor = (cursor as Record<string, unknown>)[segment];
      continue;
    }
    return undefined;
  }
  return cursor;
}

/**
 * Split a field path into its constituent segments. Accepts both dot
 * (`a.b.c`) and bracket (`a[0].b`) notation. Bracket indices are stringified
 * before being returned so callers can treat the whole list as `string[]`.
 */
export function parsePath(path: string): string[] {
  const out: string[] = [];
  let buffer = "";
  let i = 0;
  while (i < path.length) {
    const ch = path[i];
    if (ch === ".") {
      if (buffer.length > 0) {
        out.push(buffer);
        buffer = "";
      }
      i++;
      continue;
    }
    if (ch === "[") {
      if (buffer.length > 0) {
        out.push(buffer);
        buffer = "";
      }
      const end = path.indexOf("]", i + 1);
      if (end === -1) throw new Error(`Unterminated '[' in path: ${path}`);
      const segment = path
        .slice(i + 1, end)
        .trim()
        .replace(/^['"]|['"]$/g, "");
      out.push(segment);
      i = end + 1;
      continue;
    }
    buffer += ch;
    i++;
  }
  if (buffer.length > 0) out.push(buffer);
  return out;
}

/**
 * Interpolate `{field}` placeholders in a template string using a row record.
 * Unknown fields render as empty strings.
 */
export function interpolate(template: string, row: unknown): string {
  return template.replace(/\{([^}]+)\}/g, (_, raw) => {
    const value = getValueAtPath(row, raw.trim());
    if (value == null) return "";
    return String(value);
  });
}
