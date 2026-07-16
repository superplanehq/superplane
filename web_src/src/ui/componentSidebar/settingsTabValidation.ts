import type { ComponentsIntegrationRef, ConfigurationField } from "@/api-client";

export function buildAutosaveSnapshot(
  configuration: Record<string, unknown>,
  nodeName: string,
  integrationRef?: ComponentsIntegrationRef,
): string {
  return JSON.stringify({
    configuration,
    nodeName,
    integrationRef: integrationRef
      ? {
          id: integrationRef.id || "",
          name: integrationRef.name || "",
        }
      : null,
  });
}

export function shouldAutosaveOnChangeByFieldType(fieldType: ConfigurationField["type"] | undefined): boolean {
  if (!fieldType) {
    return false;
  }

  return ![
    "string",
    "text",
    "xml",
    "expression",
    "number",
    "url",
    "date",
    "datetime",
    "time",
    "cron",
    "git-ref",
    "app",
  ].includes(fieldType);
}
