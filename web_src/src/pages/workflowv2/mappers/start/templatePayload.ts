export type StartTemplateParameterType = "string" | "number" | "boolean" | "select";

export interface StartTemplateParameterOption {
  label: string;
  value: string;
}

export interface StartTemplateParameter {
  name: string;
  title?: string;
  type: StartTemplateParameterType;
  options?: StartTemplateParameterOption[];
  defaultString?: unknown;
  defaultNumber?: unknown;
  defaultBoolean?: unknown;
}

export function parameterDisplayLabel(param: StartTemplateParameter): string {
  const title = typeof param.title === "string" ? param.title.trim() : "";
  return title || param.name;
}

export function selectOptionValues(param: StartTemplateParameter): string[] {
  if (param.type !== "select" || !param.options) {
    return [];
  }
  return param.options.map((opt) => opt.value).filter((value) => value !== "");
}

export function isValidSelectParameterValue(param: StartTemplateParameter, value: string): boolean {
  const allowed = selectOptionValues(param);
  if (allowed.length === 0) {
    return true;
  }
  return allowed.includes(value);
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
    case "select":
    case "string": {
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

export function payloadRecordForParameters(payload: Record<string, unknown> | string): Record<string, unknown> {
  if (typeof payload !== "string") {
    return payload;
  }
  try {
    const parsed = JSON.parse(payload) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Expression placeholders make the payload invalid JSON until run time.
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
    case "select":
    case "string":
      return raw == null ? "" : String(raw);
  }
}

export function initialParameterValue(param: StartTemplateParameter): string | number | boolean {
  const configuredDefault = parameterDefaultValue(param);
  if (configuredDefault !== undefined) {
    return coerceParameterValue(param, configuredDefault) as string | number | boolean;
  }
  if (param.type === "select") {
    const firstOption = selectOptionValues(param)[0];
    return firstOption ?? "";
  }
  return param.type === "boolean" ? false : param.type === "number" ? 0 : "";
}
