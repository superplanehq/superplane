export type StartTemplateParameterType = "string" | "number" | "boolean";

export interface StartTemplateParameter {
  name: string;
  type: StartTemplateParameterType;
  defaultString?: unknown;
  defaultNumber?: unknown;
  defaultBoolean?: unknown;
}

export interface StartTemplate {
  name: string;
  payload: Record<string, unknown>;
  parameters?: StartTemplateParameter[];
}

export interface StartConfiguration {
  templates?: StartTemplate[];
}

export function parameterDefaultValue(param: StartTemplateParameter): unknown | undefined {
  switch (param.type) {
    case "number": {
      const value = param.defaultNumber;
      return value === null || value === undefined ? undefined : value;
    }
    case "boolean": {
      const value = param.defaultBoolean;
      return value === null || value === undefined ? undefined : value;
    }
    default: {
      const value = param.defaultString;
      if (value === null || value === undefined) return undefined;
      if (typeof value === "string" && value === "") return undefined;
      return value;
    }
  }
}

export function payloadForTemplateRun(template: StartTemplate): Record<string, unknown> {
  const payload = template.payload;
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload;
  }
  return {};
}

export function coerceParameterValue(param: StartTemplateParameter, raw: unknown): unknown {
  switch (param.type) {
    case "number":
      if (typeof raw === "number") return raw;
      if (raw === "" || raw == null) return 0;
      return Number(raw);
    case "boolean":
      if (typeof raw === "boolean") return raw;
      return raw === true || raw === "true" || raw === "1";
    default:
      return raw == null ? "" : String(raw);
  }
}

export function initialParameterValue(
  param: StartTemplateParameter,
  payload: Record<string, unknown>,
): string | number | boolean {
  const fromPayload = payload[param.name];
  if (fromPayload !== undefined) {
    return coerceParameterValue(param, fromPayload) as string | number | boolean;
  }
  const configuredDefault = parameterDefaultValue(param);
  if (configuredDefault !== undefined) {
    return coerceParameterValue(param, configuredDefault) as string | number | boolean;
  }
  return param.type === "boolean" ? false : param.type === "number" ? 0 : "";
}
