function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function defaultNumberValue(parameter: Record<string, unknown>): unknown {
  const value = parameter.defaultNumber;
  return value === null || value === undefined ? 0 : value;
}

function defaultBooleanValue(parameter: Record<string, unknown>): unknown {
  const value = parameter.defaultBoolean;
  return value === null || value === undefined ? false : value;
}

function firstSelectOptionValue(rawOptions: unknown): string {
  if (!Array.isArray(rawOptions) || rawOptions.length === 0) {
    return "";
  }
  const first = rawOptions[0];
  if (isRecord(first) && typeof first.value === "string" && first.value !== "") {
    return first.value;
  }
  return "";
}

function defaultSelectValue(parameter: Record<string, unknown>): unknown {
  const value = parameter.defaultString;
  if (value === null || value === undefined || value === "") {
    return firstSelectOptionValue(parameter.options);
  }
  return value;
}

function defaultStringValue(parameter: Record<string, unknown>): unknown {
  const value = parameter.defaultString;
  if (value === null || value === undefined) return "";
  return value;
}

function templateParameterValue(parameter: Record<string, unknown>): unknown {
  switch (parameter.type) {
    case "number":
      return defaultNumberValue(parameter);
    case "boolean":
      return defaultBooleanValue(parameter);
    case "select":
      return defaultSelectValue(parameter);
    default:
      return defaultStringValue(parameter);
  }
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
