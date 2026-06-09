import type { ComponentsIntegrationRef, ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { isFieldVisible, parseDefaultValues } from "@/lib/components";
import { EMPTY_DISPLAY_VALUE, formatConfigurationLabel, formatConfigurationValue } from "./formatConfigurationValue";
import type { ConfigurationDisplayModel, ConfigurationDisplayRow } from "./types";

export type BuildConfigurationDisplayModelInput = {
  configuration: Record<string, unknown>;
  configurationFields: ConfigurationField[];
  integrationName?: string;
  integrationRef?: ComponentsIntegrationRef;
  integrations?: OrganizationsIntegration[];
  allowIntegrations?: boolean;
};

function resolveIntegrationInstanceName(integration: OrganizationsIntegration): string {
  const instanceName = integration.metadata?.name;
  const typeName = integration.metadata?.integrationName;
  if (!instanceName) {
    return "Unnamed integration";
  }
  if (instanceName.toLowerCase() === typeName?.toLowerCase()) {
    return getIntegrationTypeDisplayName(undefined, typeName) || instanceName;
  }
  return instanceName;
}

function appendCustomNameRow(
  rows: ConfigurationDisplayRow[],
  configuration: Record<string, unknown>,
  configurationFields: ConfigurationField[],
): void {
  const runTitleField = configurationFields.find((field) => field.name === "customName");
  if (!runTitleField?.name || !isFieldVisible(runTitleField, configuration)) {
    return;
  }

  const formatted = formatConfigurationValue(runTitleField, configuration[runTitleField.name]);
  rows.push({
    key: "customName",
    label: formatConfigurationLabel(runTitleField),
    ...formatted,
  });
}

function appendIntegrationRows(rows: ConfigurationDisplayRow[], input: BuildConfigurationDisplayModelInput): void {
  const { integrationName, integrationRef, integrations = [], allowIntegrations = true } = input;
  if (!integrationName) {
    return;
  }

  const typeLabel = getIntegrationTypeDisplayName(undefined, integrationName) || integrationName;

  if (!allowIntegrations) {
    rows.push({
      key: "integration.permission",
      label: "Integration",
      kind: "text",
      displayText: "You don't have permission to view integrations.",
    });
    return;
  }

  const integrationsOfType = integrations.filter(
    (integration) => integration.metadata?.integrationName === integrationName,
  );
  if (integrationsOfType.length === 0) {
    rows.push({
      key: "integration.notConnected",
      label: "Integration",
      kind: "integration",
      displayText: `${typeLabel} — not connected`,
      integrationStatus: "Not connected",
      integrationStatusVariant: "pending",
    });
    return;
  }

  const selectedIntegration =
    integrationsOfType.find((integration) => integration.metadata?.id === integrationRef?.id) ?? integrationsOfType[0];

  const status = selectedIntegration.status?.state ?? "unknown";
  const statusLabel = status.charAt(0).toUpperCase() + status.slice(1);
  const statusVariant = status === "ready" ? "ready" : status === "error" ? "error" : ("pending" as const);

  rows.push({
    key: "integration.type",
    label: "Type",
    kind: "text",
    displayText: typeLabel,
  });
  rows.push({
    key: "integration.instance",
    label: "Instance",
    kind: "text",
    displayText: resolveIntegrationInstanceName(selectedIntegration),
  });
  rows.push({
    key: "integration.status",
    label: "Connection",
    kind: "integration",
    displayText: statusLabel,
    integrationStatus: statusLabel,
    integrationStatusVariant: statusVariant,
  });

  if (status === "error" && selectedIntegration.status?.stateDescription) {
    rows.push({
      key: "integration.statusDescription",
      label: "Status details",
      kind: "text",
      displayText: selectedIntegration.status.stateDescription,
    });
  }
}

function appendFieldRows(
  fields: ConfigurationField[],
  values: Record<string, unknown>,
  rootConfiguration: Record<string, unknown>,
  parentPath: string,
  depth: number,
  rows: ConfigurationDisplayRow[],
): void {
  for (const field of fields) {
    if (!field.name || field.name === "customName") {
      continue;
    }
    if (!isFieldVisible(field, { ...rootConfiguration, ...values })) {
      continue;
    }

    const fieldPath = parentPath ? `${parentPath}.${field.name}` : field.name;
    const rawValue = values[field.name];
    const objectSchema = field.typeOptions?.object?.schema;
    const listItemSchema = field.typeOptions?.list?.itemDefinition?.schema;
    const listItemType = field.typeOptions?.list?.itemDefinition?.type;

    if (field.type === "object" && objectSchema && isRecord(rawValue)) {
      const schemaDefaults = parseDefaultValues(objectSchema);
      const mergedValues = { ...schemaDefaults, ...rawValue };
      if (depth === 0) {
        rows.push({
          key: `${fieldPath}.__group`,
          label: formatConfigurationLabel(field),
          kind: "text",
          displayText: "",
          depth,
        });
      }
      appendFieldRows(objectSchema, mergedValues, rootConfiguration, fieldPath, depth + 1, rows);
      continue;
    }

    if (field.type === "list" && Array.isArray(rawValue) && listItemSchema && listItemType === "object") {
      rows.push({
        key: `${fieldPath}.__group`,
        label: formatConfigurationLabel(field),
        kind: rawValue.length === 0 ? "empty" : "list",
        displayText:
          rawValue.length === 0 ? EMPTY_DISPLAY_VALUE : `${rawValue.length} item${rawValue.length === 1 ? "" : "s"}`,
        depth,
      });

      rawValue.forEach((item, index) => {
        if (!isRecord(item)) {
          return;
        }
        const itemLabel = field.typeOptions?.list?.itemLabel ?? "Item";
        rows.push({
          key: `${fieldPath}[${index}].__header`,
          label: `${itemLabel} ${index + 1}`,
          kind: "text",
          displayText: "",
          depth: depth + 1,
        });
        appendFieldRows(listItemSchema, item, rootConfiguration, `${fieldPath}[${index}]`, depth + 2, rows);
      });
      continue;
    }

    const formatted = formatConfigurationValue(field, rawValue);
    rows.push({
      key: fieldPath,
      label: formatConfigurationLabel(field),
      depth,
      ...formatted,
    });
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function buildConfigurationDisplayModel(input: BuildConfigurationDisplayModelInput): ConfigurationDisplayModel {
  const rows: ConfigurationDisplayRow[] = [];

  appendCustomNameRow(rows, input.configuration, input.configurationFields);
  appendIntegrationRows(rows, input);
  appendFieldRows(input.configurationFields, input.configuration, input.configuration, "", 0, rows);

  return { rows };
}
