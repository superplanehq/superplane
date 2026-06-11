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

type FieldRowsContext = {
  fields: ConfigurationField[];
  values: Record<string, unknown>;
  rootConfiguration: Record<string, unknown>;
  parentPath: string;
  depth: number;
  rows: ConfigurationDisplayRow[];
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

function appendNotConnectedIntegrationRow(rows: ConfigurationDisplayRow[], typeLabel: string): void {
  rows.push({
    key: "integration.notConnected",
    label: "Integration",
    kind: "integration",
    displayText: typeLabel,
    integrationStatus: "Not connected",
    integrationStatusVariant: "pending",
  });
}

function findSelectedIntegration(
  integrationsOfType: OrganizationsIntegration[],
  integrationRef?: ComponentsIntegrationRef,
): OrganizationsIntegration | undefined {
  if (!integrationRef?.id) {
    return undefined;
  }

  return integrationsOfType.find((integration) => integration.metadata?.id === integrationRef.id);
}

function appendConnectedIntegrationRows(
  rows: ConfigurationDisplayRow[],
  typeLabel: string,
  selectedIntegration: OrganizationsIntegration,
): void {
  const status = selectedIntegration.status?.state ?? "unknown";
  const statusLabel = status.charAt(0).toUpperCase() + status.slice(1);
  const statusVariant: ConfigurationDisplayRow["integrationStatusVariant"] =
    status === "ready" || status === "error" ? status : "pending";

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
  const selectedIntegration = findSelectedIntegration(integrationsOfType, integrationRef);
  if (!selectedIntegration) {
    appendNotConnectedIntegrationRow(rows, typeLabel);
    return;
  }

  appendConnectedIntegrationRows(rows, typeLabel, selectedIntegration);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function shouldSkipField(field: ConfigurationField, ctx: FieldRowsContext): boolean {
  if (!field.name || field.name === "customName") {
    return true;
  }
  return !isFieldVisible(field, { ...ctx.rootConfiguration, ...ctx.values });
}

function appendObjectFieldRows(
  field: ConfigurationField,
  rawValue: Record<string, unknown>,
  fieldPath: string,
  ctx: FieldRowsContext,
): void {
  const objectSchema = field.typeOptions?.object?.schema;
  if (!objectSchema) {
    return;
  }

  const schemaDefaults = parseDefaultValues(objectSchema);
  const mergedValues = { ...schemaDefaults, ...rawValue };
  if (ctx.depth === 0) {
    ctx.rows.push({
      key: `${fieldPath}.__group`,
      label: formatConfigurationLabel(field),
      kind: "text",
      displayText: "",
      depth: ctx.depth,
    });
  }

  appendFieldRows({
    ...ctx,
    fields: objectSchema,
    values: mergedValues,
    parentPath: fieldPath,
    depth: ctx.depth + 1,
  });
}

function appendListFieldRows(
  field: ConfigurationField,
  rawValue: unknown[],
  fieldPath: string,
  ctx: FieldRowsContext,
): void {
  const listItemSchema = field.typeOptions?.list?.itemDefinition?.schema;
  if (!listItemSchema) {
    return;
  }

  ctx.rows.push({
    key: `${fieldPath}.__group`,
    label: formatConfigurationLabel(field),
    kind: rawValue.length === 0 ? "empty" : "list",
    displayText:
      rawValue.length === 0 ? EMPTY_DISPLAY_VALUE : `${rawValue.length} item${rawValue.length === 1 ? "" : "s"}`,
    depth: ctx.depth,
  });

  const itemLabel = field.typeOptions?.list?.itemLabel ?? "Item";
  rawValue.forEach((item, index) => {
    if (!isRecord(item)) {
      return;
    }

    ctx.rows.push({
      key: `${fieldPath}[${index}].__header`,
      label: `${itemLabel} ${index + 1}`,
      kind: "text",
      displayText: "",
      depth: ctx.depth + 1,
    });
    appendFieldRows({
      ...ctx,
      fields: listItemSchema,
      values: item,
      parentPath: `${fieldPath}[${index}]`,
      depth: ctx.depth + 2,
    });
  });
}

function appendScalarFieldRow(
  field: ConfigurationField,
  rawValue: unknown,
  fieldPath: string,
  ctx: FieldRowsContext,
): void {
  const formatted = formatConfigurationValue(field, rawValue);
  ctx.rows.push({
    key: fieldPath,
    label: formatConfigurationLabel(field),
    depth: ctx.depth,
    ...formatted,
  });
}

function hasObjectFieldSchema(field: ConfigurationField): boolean {
  return field.type === "object" && Boolean(field.typeOptions?.object?.schema);
}

function isListFieldValue(field: ConfigurationField, rawValue: unknown): rawValue is unknown[] {
  const listItemType = field.typeOptions?.list?.itemDefinition?.type;
  return (
    field.type === "list" &&
    Array.isArray(rawValue) &&
    Boolean(field.typeOptions?.list?.itemDefinition?.schema) &&
    listItemType === "object"
  );
}

function processField(field: ConfigurationField, ctx: FieldRowsContext): void {
  const fieldPath = ctx.parentPath ? `${ctx.parentPath}.${field.name}` : field.name!;
  const rawValue = ctx.values[field.name!];

  if (hasObjectFieldSchema(field)) {
    appendObjectFieldRows(field, isRecord(rawValue) ? rawValue : {}, fieldPath, ctx);
    return;
  }

  if (isListFieldValue(field, rawValue)) {
    appendListFieldRows(field, rawValue, fieldPath, ctx);
    return;
  }

  appendScalarFieldRow(field, rawValue, fieldPath, ctx);
}

function appendFieldRows(ctx: FieldRowsContext): void {
  for (const field of ctx.fields) {
    if (shouldSkipField(field, ctx)) {
      continue;
    }

    processField(field, ctx);
  }
}

export function buildConfigurationDisplayModel(input: BuildConfigurationDisplayModelInput): ConfigurationDisplayModel {
  const rows: ConfigurationDisplayRow[] = [];

  appendCustomNameRow(rows, input.configuration, input.configurationFields);
  appendIntegrationRows(rows, input);
  appendFieldRows({
    fields: input.configurationFields,
    values: input.configuration,
    rootConfiguration: input.configuration,
    parentPath: "",
    depth: 0,
    rows,
  });

  return { rows };
}
