function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function templateParameterValue(parameter: Record<string, unknown>): unknown {
  const parameterType = parameter.type;
  if (parameterType === "number") {
    const value = parameter.defaultNumber;
    return value === null || value === undefined ? 0 : value;
  }
  if (parameterType === "boolean") {
    const value = parameter.defaultBoolean;
    return value === null || value === undefined ? false : value;
  }
  const value = parameter.defaultString;
  if (value === null || value === undefined) return "";
  return value;
}

export function buildTemplateParametersAutocompleteObject(
  allValues: Record<string, unknown>,
): Record<string, unknown> | null {
  const rawParameters = allValues.parameters;
  if (!Array.isArray(rawParameters) || rawParameters.length === 0) {
    return null;
  }

  const parameters: Record<string, unknown> = {};
  for (const rawParameter of rawParameters) {
    if (!isRecord(rawParameter)) {
      continue;
    }

    const name = rawParameter.name;
    if (typeof name !== "string" || name.trim() === "") {
      continue;
    }

    parameters[name] = templateParameterValue(rawParameter);
  }

  if (Object.keys(parameters).length === 0) {
    return null;
  }

  return parameters;
}
