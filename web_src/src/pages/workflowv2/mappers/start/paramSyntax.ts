/** Mirrors `params.ParamType` in pkg/triggers/start/params/definition.go. */
export type ParamType = "string" | "number" | "boolean" | "select";

/** Mirrors `params.Definition` in pkg/triggers/start/params/definition.go. */
export type ParamDefinition = {
  path: string;
  type: ParamType;
  title: string;
  default: unknown;
  required: boolean;
  values: string[];
};

/** Mirrors `params.IsParamString` in pkg/triggers/start/params/parser.go. */
export function isParamString(value: string): boolean {
  return paramExprRe.test(value.trim());
}

/** Mirrors `params.HasParams` in pkg/triggers/start/params/parser.go. */
export function hasParams(payload: Record<string, unknown>): boolean {
  return (
    walkPayload(payload, "", (_path, leaf) => {
      if (typeof leaf === "string" && isParamString(leaf)) {
        return "stop";
      }
      return "continue";
    }) === "stop"
  );
}

/** Mirrors `params.ParseParams` in pkg/triggers/start/params/parser.go. */
export function parseParams(payload: Record<string, unknown>): ParamDefinition[] {
  const defs: ParamDefinition[] = [];
  let parseError: Error | undefined;

  walkPayload(payload, "", (path, leaf) => {
    if (parseError) {
      return "stop";
    }
    if (typeof leaf !== "string" || !isParamString(leaf)) {
      return "continue";
    }

    try {
      defs.push(parseParamString(path, leaf));
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      parseError = new Error(`${path}: ${message}`);
      return "stop";
    }
    return "continue";
  });

  if (parseError) {
    throw parseError;
  }
  return defs;
}

/** Mirrors `params.ParseParamString` in pkg/triggers/start/params/parser.go. */
export function parseParamString(path: string, expression: string): ParamDefinition {
  const trimmed = expression.trim();
  const match = paramExprRe.exec(trimmed);
  if (!match) {
    throw new Error("not a param() expression");
  }

  const args = splitArgs(match[1].trim());

  let paramType: ParamType | "" = "";
  let title = "";
  let defaultRaw = "";
  let defaultValue: unknown;
  let required = false;
  let values: string[] = [];

  for (const [key, raw] of Object.entries(args)) {
    switch (key) {
      case "type":
        paramType = parseTypeName(raw);
        break;
      case "title":
        title = parseQuotedString(raw, "title");
        break;
      case "default":
        defaultRaw = raw;
        break;
      case "required":
        required = parseBoolToken(raw, "required");
        break;
      case "values":
        values = parseSelectValues(raw);
        break;
      default:
        throw new Error(`unknown param() key "${key}"`);
    }
  }

  if (defaultRaw !== "") {
    if (!paramType) {
      throw new Error("param() missing type");
    }
    defaultValue = parseDefaultValue(paramType, defaultRaw);
  }

  return newDefinition(path, paramType, title, defaultValue, required, values);
}

/** Mirrors `params.NewDefinition` in pkg/triggers/start/params/definition.go. */
export function newDefinition(
  path: string,
  paramType: ParamType | "",
  title: string,
  defaultValue: unknown,
  required: boolean,
  values: string[],
): ParamDefinition {
  if (path === "") {
    throw new Error("param() path is required");
  }
  if (!paramType) {
    throw new Error("param() missing type");
  }

  switch (paramType) {
    case "select":
      if (values.length === 0) {
        throw new Error("select param requires values");
      }
      if (defaultValue !== undefined && defaultValue !== null) {
        if (typeof defaultValue !== "string") {
          throw new Error("select default must be a string");
        }
        if (!values.includes(defaultValue)) {
          throw new Error(`default "${defaultValue}" is not one of the select values`);
        }
      }
      break;
    case "string":
    case "number":
    case "boolean":
      if (values.length > 0) {
        throw new Error("values is only valid for select params");
      }
      break;
    default:
      throw new Error(`unsupported type "${paramType}"`);
  }

  return {
    path,
    type: paramType,
    title,
    default: defaultValue,
    required,
    values,
  };
}

/** Test/story fixture; same payload as `issueExamplePayload()` in pkg/triggers/start/params/apply_test.go. */
export function issueExamplePayload(): Record<string, unknown> {
  return {
    body: {
      name: "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
      size: "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
    },
  };
}

const paramExprRe = /^param\((.*)\)$/s;

type WalkControl = "continue" | "stop";

function joinPath(prefix: string, segment: string): string {
  if (prefix === "") {
    return segment;
  }
  if (segment.startsWith("[")) {
    return prefix + segment;
  }
  return `${prefix}.${segment}`;
}

function walkPayload(
  value: unknown,
  path: string,
  visit: (leafPath: string, leafValue: unknown) => WalkControl,
): WalkControl {
  if (value !== null && typeof value === "object") {
    if (Array.isArray(value)) {
      for (let i = 0; i < value.length; i++) {
        if (walkPayload(value[i], joinPath(path, `[${i}]`), visit) === "stop") {
          return "stop";
        }
      }
      return "continue";
    }

    const record = value as Record<string, unknown>;
    for (const key of Object.keys(record)) {
      if (walkPayload(record[key], joinPath(path, key), visit) === "stop") {
        return "stop";
      }
    }
    return "continue";
  }

  return visit(path, value);
}

function splitArgs(inner: string): Record<string, string> {
  if (inner === "") {
    throw new Error("param() has no arguments");
  }

  const out: Record<string, string> = {};
  for (const part of inner.split(",")) {
    const trimmed = part.trim();
    if (trimmed === "") {
      throw new Error("param() has empty argument");
    }
    const colonIndex = trimmed.indexOf(":");
    if (colonIndex < 0) {
      throw new Error(`invalid param() argument "${trimmed}": missing ':'`);
    }
    const key = trimmed.slice(0, colonIndex).trim();
    if (key === "") {
      throw new Error(`invalid param() argument "${trimmed}": empty key`);
    }
    if (key in out) {
      throw new Error(`duplicate param() key "${key}"`);
    }
    out[key] = trimmed.slice(colonIndex + 1).trim();
  }
  return out;
}

function parseTypeName(raw: string): ParamType {
  const name = raw.trim();
  if (name === "string" || name === "number" || name === "boolean" || name === "select") {
    return name;
  }
  throw new Error(`unsupported type "${name}"`);
}

function parseBoolToken(raw: string, field: string): boolean {
  const token = raw.trim();
  if (token === "true") {
    return true;
  }
  if (token === "false") {
    return false;
  }
  throw new Error(`${field}: expected true or false, got "${raw}"`);
}

function validateQuotedCharset(content: string): void {
  if (/['",]/.test(content)) {
    throw new Error("quoted value must not contain ', \", or comma");
  }
}

function parseQuotedString(raw: string, field: string): string {
  const trimmed = raw.trim();
  if (trimmed.length < 2 || trimmed[0] !== "'" || trimmed[trimmed.length - 1] !== "'") {
    throw new Error(`${field}: expected single-quoted string, got "${raw}"`);
  }
  const content = trimmed.slice(1, -1);
  validateQuotedCharset(content);
  return content;
}

function parseSelectValues(raw: string): string[] {
  const content = parseQuotedString(raw, "values");
  const parts = content.split("|");
  if (parts.length === 0) {
    throw new Error("select values must not be empty");
  }

  const out: string[] = [];
  for (const part of parts) {
    const option = part.trim();
    if (option === "") {
      throw new Error("select option must not be empty");
    }
    if (option.includes("|")) {
      throw new Error("select option must not contain |");
    }
    validateQuotedCharset(option);
    out.push(option);
  }
  return out;
}

function parseDefaultValue(paramType: ParamType, raw: string): unknown {
  const trimmed = raw.trim();
  switch (paramType) {
    case "boolean":
      return parseBoolToken(trimmed, "default");
    case "number": {
      const value = Number(trimmed);
      if (!Number.isFinite(value)) {
        throw new Error(`default: expected number, got "${raw}"`);
      }
      return value;
    }
    case "string":
    case "select":
      return parseQuotedString(trimmed, "default");
    default:
      throw new Error(`unsupported type "${paramType}"`);
  }
}
