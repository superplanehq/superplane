/**
 * Coerce arbitrary input into a string suitable for Monaco Editor's
 * `value`/`defaultValue` props.
 *
 * Monaco's underlying `editor.createModel(value, language, uri)` expects a
 * string. Passing a non-string (e.g. a plain object) blows up deep inside
 * Monaco with "$.create is not a function" because the model factory tries to
 * call `.create()` on what it assumes is a text-buffer factory. See
 * https://github.com/microsoft/monaco-editor/issues/4559.
 *
 * Persisted configuration values are typed as `unknown` and can legitimately
 * end up holding non-string data (e.g. an object stored under what is now a
 * text-typed field). Always run user-provided values through this helper
 * before handing them to Monaco.
 */
export function coerceMonacoValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (value === undefined || value === null) {
    return "";
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  try {
    return JSON.stringify(value, null, 2) ?? "";
  } catch {
    return "";
  }
}
