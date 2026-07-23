export type StartTemplateParameterType = "string" | "text" | "number" | "boolean" | "select";

export interface StartTemplateParameterOption {
  label: string;
  value: string;
}

export interface StartTemplateParameter {
  name: string;
  title?: string;
  placeholder?: string;
  type: StartTemplateParameterType;
  options?: StartTemplateParameterOption[];
  defaultString?: unknown;
  defaultNumber?: unknown;
  defaultBoolean?: unknown;
}

export function parameterDisplayLabel(param: StartTemplateParameter): string {
  const title = typeof param.title === "string" ? param.title.trim() : "";
  const name = param.name.trim();
  if (title && name && title.toLowerCase() === name.toLowerCase()) {
    return title;
  }
  return title || name;
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

export function parameterPlaceholder(param: StartTemplateParameter): string {
  const placeholder = typeof param.placeholder === "string" ? param.placeholder.trim() : "";
  return placeholder;
}

/** Placeholder for run-form inputs; omitted when it would repeat the field label. */
export function parameterInputPlaceholder(param: StartTemplateParameter, label: string): string | undefined {
  const placeholder = parameterPlaceholder(param);
  if (!placeholder || placeholder.toLowerCase() === label.toLowerCase()) {
    return undefined;
  }
  return placeholder;
}

/** Title for the manual-run modal opened from a Start trigger template. */
export function startRunModalTitle(nodeName: string | undefined, templateName: string): string {
  const trimmedNodeName = nodeName?.trim();
  if (trimmedNodeName) {
    return trimmedNodeName;
  }
  const trimmedTemplateName = templateName.trim();
  if (trimmedTemplateName) {
    return trimmedTemplateName;
  }
  return "Run";
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
    case "string":
    case "text": {
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
    case "text":
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
  if (param.type === "boolean") return false;
  if (param.type === "number") return 0;
  return "";
}

export function validateSubmittedParameterValue(param: StartTemplateParameter, coerced: unknown): string | null {
  if (param.type === "number" && typeof coerced === "number" && Number.isNaN(coerced)) {
    return `"${parameterDisplayLabel(param)}" must be a valid number`;
  }
  if (param.type === "select" && !isValidSelectParameterValue(param, String(coerced ?? ""))) {
    return `"${parameterDisplayLabel(param)}" must be one of the configured options`;
  }
  return null;
}

export function buildParameterFormPayload(
  parameters: StartTemplateParameter[] | undefined,
  parameterValues: Record<string, string | number | boolean>,
): { payload: Record<string, unknown> } | { error: string } {
  const payload: Record<string, unknown> = {};
  for (const param of parameters ?? []) {
    if (!param.name || !param.type) continue;
    const coerced = coerceParameterValue(param, parameterValues[param.name]);
    const validationError = validateSubmittedParameterValue(param, coerced);
    if (validationError) {
      return { error: validationError };
    }
    payload[param.name] = coerced;
  }
  return { payload };
}

export function parseJsonEventPayload(eventData: string): { payload: Record<string, unknown> } | { error: string } {
  try {
    const candidate = JSON.parse(eventData) as unknown;
    if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) {
      return { error: "Payload must be a JSON object" };
    }
    return { payload: candidate as Record<string, unknown> };
  } catch {
    return { error: "Invalid JSON format" };
  }
}
