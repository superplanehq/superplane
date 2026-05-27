export type ManualRunParamType = "string" | "number" | "boolean" | "select";

export interface ManualRunParamDefinition {
  type: ManualRunParamType;
  title: string;
  default?: string | number | boolean;
  required: boolean;
  values?: string[];
}

export interface ManualRunParamField {
  path: string;
  def: ManualRunParamDefinition;
}

const PARAM_PREFIX = "param(";

export function parseParamString(
  value: string,
): { def: ManualRunParamDefinition; isParam: boolean } | { error: string } {
  const trimmed = value.trim();
  if (!trimmed.startsWith(PARAM_PREFIX) || !trimmed.endsWith(")")) {
    return { def: { type: "string", title: "", required: false }, isParam: false };
  }

  const body = trimmed.slice(PARAM_PREFIX.length, -1).trim();
  let options: Record<string, string>;
  try {
    options = parseOptionPairs(body);
  } catch (err) {
    return { error: err instanceof Error ? err.message : "invalid param() syntax" };
  }

  let type: ManualRunParamType | undefined;
  let title = "";
  let required = false;
  let defaultValue: string | undefined;
  let values: string[] | undefined;

  for (const [key, rawVal] of Object.entries(options)) {
    switch (key) {
      case "type": {
        const t = rawVal.trim() as ManualRunParamType;
        if (t !== "string" && t !== "number" && t !== "boolean" && t !== "select") {
          return { error: `unknown param type ${JSON.stringify(rawVal)}` };
        }
        type = t;
        break;
      }
      case "title":
        title = rawVal;
        break;
      case "default":
        defaultValue = rawVal;
        break;
      case "required":
        required = rawVal.trim() === "true";
        break;
      case "values":
        values = rawVal
          .split("|")
          .map((v) => v.trim())
          .filter(Boolean);
        break;
      default:
        return { error: `unknown param option ${JSON.stringify(key)}` };
    }
  }

  if (!type) {
    return { error: "param() missing type" };
  }
  if (type === "select" && (!values || values.length === 0)) {
    return { error: "select param requires values" };
  }
  if (!title) {
    return { error: "param() missing title" };
  }

  const def: ManualRunParamDefinition = { type, title, required };
  if (defaultValue !== undefined) {
    def.default = coerceDefault(type, defaultValue);
  }
  if (values) {
    def.values = values;
  }

  return { def, isParam: true };
}

export function hasManualRunParams(payload: Record<string, unknown>): boolean {
  return extractManualRunParams(payload).length > 0;
}

export function extractManualRunParams(payload: Record<string, unknown>): ManualRunParamField[] {
  const fields: ManualRunParamField[] = [];
  walkPayload(payload, "", fields);
  return fields;
}

function walkPayload(value: unknown, prefix: string, fields: ManualRunParamField[]): void {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    const record = value as Record<string, unknown>;
    const keys = Object.keys(record).sort();
    for (const key of keys) {
      const path = prefix ? `${prefix}.${key}` : key;
      walkPayload(record[key], path, fields);
    }
    return;
  }

  if (Array.isArray(value)) {
    value.forEach((item, index) => {
      const path = prefix ? `${prefix}[${index}]` : `[${index}]`;
      walkPayload(item, path, fields);
    });
    return;
  }

  if (typeof value !== "string" || !value.trim().startsWith(PARAM_PREFIX)) {
    return;
  }

  const parsed = parseParamString(value);
  if ("error" in parsed) {
    return;
  }
  if (!parsed.isParam || !prefix) {
    return;
  }
  fields.push({ path: prefix, def: parsed.def });
}

export function mergeManualRunPayload(
  template: Record<string, unknown>,
  values: Record<string, unknown>,
): { payload?: Record<string, unknown>; error?: string } {
  const fields = extractManualRunParams(template);
  for (const field of fields) {
    if (field.def.required && !(field.path in values)) {
      return { error: `missing required parameter ${JSON.stringify(field.path)}` };
    }
  }

  const out = deepClone(template) as Record<string, unknown>;

  for (const field of fields) {
    const raw = values[field.path];
    let resolved: unknown;
    if (raw !== undefined) {
      const coerced = coerceSubmittedValue(field.def, raw);
      if ("error" in coerced) {
        return { error: `field ${JSON.stringify(field.path)}: ${coerced.error}` };
      }
      resolved = coerced.value;
    } else if (field.def.default !== undefined) {
      resolved = field.def.default;
    } else if (field.def.required) {
      return { error: `missing required parameter ${JSON.stringify(field.path)}` };
    } else {
      continue;
    }

    const setResult = setAtPath(out, field.path, resolved);
    if (setResult) {
      return { error: setResult };
    }
  }

  return { payload: out };
}

export function defaultFormValues(fields: ManualRunParamField[]): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const field of fields) {
    if (field.def.default !== undefined) {
      values[field.path] = field.def.default;
    } else if (field.def.type === "boolean") {
      values[field.path] = false;
    } else if (field.def.type === "select" && field.def.values?.length) {
      values[field.path] = field.def.values[0];
    }
  }
  return values;
}

function coerceDefault(type: ManualRunParamType, raw: string): string | number | boolean {
  switch (type) {
    case "number": {
      const n = Number(raw);
      return Number.isFinite(n) ? n : raw;
    }
    case "boolean":
      return raw === "true";
    default:
      return raw;
  }
}

function coerceSubmittedValue(def: ManualRunParamDefinition, raw: unknown): { value: unknown } | { error: string } {
  switch (def.type) {
    case "string":
      if (typeof raw === "string") return { value: raw };
      if (typeof raw === "number" || typeof raw === "boolean") return { value: String(raw) };
      return { error: "expected string" };
    case "number": {
      if (typeof raw === "number" && Number.isFinite(raw)) return { value: raw };
      if (typeof raw === "string" && raw.trim() !== "") {
        const n = Number(raw);
        if (Number.isFinite(n)) return { value: n };
      }
      return { error: "expected number" };
    }
    case "boolean":
      if (typeof raw === "boolean") return { value: raw };
      if (raw === "true" || raw === "false") return { value: raw === "true" };
      return { error: "expected boolean" };
    case "select": {
      if (typeof raw !== "string") return { error: "expected string" };
      if (def.values?.includes(raw)) return { value: raw };
      return { error: `value ${JSON.stringify(raw)} is not one of the allowed options` };
    }
    default:
      return { error: "unknown param type" };
  }
}

function parseOptionPairs(body: string): Record<string, string> {
  const parts = splitOutsideQuotes(body, ",");
  const opts: Record<string, string> = {};
  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed) continue;
    const idx = indexOfKeyValueColon(trimmed);
    if (idx < 0) {
      throw new Error(`invalid param option ${JSON.stringify(trimmed)}`);
    }
    const key = trimmed.slice(0, idx).trim();
    const val = unquote(trimmed.slice(idx + 1).trim());
    if (!key) {
      throw new Error(`invalid param option ${JSON.stringify(trimmed)}`);
    }
    opts[key] = val;
  }
  return opts;
}

function splitOutsideQuotes(input: string, separator: string): string[] {
  const parts: string[] = [];
  let current = "";
  let inQuote = false;
  for (let i = 0; i < input.length; i++) {
    const c = input[i];
    if (c === "'") {
      inQuote = !inQuote;
      current += c;
      continue;
    }
    if (c === separator && !inQuote) {
      parts.push(current);
      current = "";
      continue;
    }
    current += c;
  }
  if (current) {
    parts.push(current);
  }
  return parts;
}

function indexOfKeyValueColon(part: string): number {
  let inQuote = false;
  for (let i = 0; i < part.length; i++) {
    if (part[i] === "'") {
      inQuote = !inQuote;
      continue;
    }
    if (part[i] === ":" && !inQuote) {
      return i;
    }
  }
  return -1;
}

function unquote(value: string): string {
  const trimmed = value.trim();
  if (trimmed.length >= 2 && trimmed.startsWith("'") && trimmed.endsWith("'")) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function deepClone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

function setAtPath(root: Record<string, unknown>, path: string, value: unknown): string | undefined {
  const segments = path.split(".");
  let current: unknown = root;
  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    const isLast = i === segments.length - 1;
    if (!current || typeof current !== "object" || Array.isArray(current)) {
      return `path ${JSON.stringify(path)} not found at segment ${JSON.stringify(segment)}`;
    }
    const node = current as Record<string, unknown>;
    if (isLast) {
      node[segment] = value;
      return undefined;
    }
    if (!(segment in node)) {
      return `path ${JSON.stringify(path)} not found at segment ${JSON.stringify(segment)}`;
    }
    current = node[segment];
  }
  return `path ${JSON.stringify(path)} not found`;
}
