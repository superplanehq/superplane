import { useMemo } from "react";
import type { AuthorizationDomainType, ConfigurationField, SuperplaneComponentsNode } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";
import { ConfigurationFieldRenderer } from "./index";
import { ObjectFieldRenderer } from "./ObjectFieldRenderer";
import type { FieldRendererProps, ValidationError } from "./types";
import { normalizeRunParameterDefinitions } from "./runParameters";

interface RunParametersFieldRendererProps extends FieldRendererProps {
  domainId?: string;
  domainType?: AuthorizationDomainType;
  organizationId?: string;
  allowExpressions?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}

function resolveTargetNodeId(allValues?: Record<string, unknown>): string | undefined {
  const nodeId = allValues?.node;
  if (typeof nodeId !== "string" || nodeId.trim().length === 0) {
    return undefined;
  }

  return nodeId;
}

function resolveTargetAppId(allValues?: Record<string, unknown>): string | undefined {
  const appId = allValues?.app;
  if (typeof appId !== "string" || appId.trim().length === 0) {
    return undefined;
  }

  return appId;
}

function findTargetNode(
  nodes: SuperplaneComponentsNode[] | undefined,
  nodeId: string | undefined,
): SuperplaneComponentsNode | undefined {
  if (!nodeId || !nodes?.length) {
    return undefined;
  }

  return nodes.find((node) => node.id === nodeId);
}

export function RunParametersFieldRenderer({
  field,
  value,
  onChange,
  allValues,
  domainId,
  domainType,
  organizationId,
  allowExpressions = false,
  autocompleteExampleObj,
  readOnly = false,
  validationErrors,
  fieldPath,
}: RunParametersFieldRendererProps) {
  const appId = useMemo(() => resolveTargetAppId(allValues), [allValues]);
  const nodeId = useMemo(() => resolveTargetNodeId(allValues), [allValues]);

  const {
    data: canvas,
    isLoading,
    error,
  } = useCanvas(organizationId ?? "", appId ?? "", {
    enabled: Boolean(organizationId && appId),
  });

  const parameterDefinitions = useMemo(() => {
    const targetNode = findTargetNode(canvas?.spec?.nodes, nodeId);
    return normalizeRunParameterDefinitions(targetNode?.configuration?.parameters);
  }, [canvas?.spec?.nodes, nodeId]);

  const parameterValues = useMemo(() => {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return value as Record<string, unknown>;
    }

    return {};
  }, [value]);

  const fallbackObjectField = useMemo(
    (): ConfigurationField => ({
      name: field.name ?? "parameters",
      label: field.label,
      description: field.description,
      type: "object",
      required: field.required,
    }),
    [field.description, field.label, field.name, field.required],
  );

  const baseFieldPath = fieldPath || field.name || "parameters";

  if (!organizationId) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">Run parameters field requires organization context.</div>
    );
  }

  if (!appId || !nodeId) {
    return (
      <div data-testid={toTestId(`run-parameters-field-${field.name}`)} className="space-y-2">
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Choose the target app and node before configuring run parameters.
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load run parameters: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div data-testid={toTestId(`run-parameters-field-${field.name}`)}>
        <p className="text-xs text-gray-500 dark:text-gray-400">Loading run parameters...</p>
      </div>
    );
  }

  if (parameterDefinitions.length === 0) {
    return (
      <div data-testid={toTestId(`run-parameters-field-${field.name}`)}>
        <ObjectFieldRenderer
          field={fallbackObjectField}
          value={value}
          onChange={onChange}
          allValues={allValues}
          domainId={domainId}
          domainType={domainType}
          organizationId={organizationId}
          allowExpressions={allowExpressions}
          autocompleteExampleObj={autocompleteExampleObj}
          readOnly={readOnly}
        />
      </div>
    );
  }

  return (
    <div
      data-testid={toTestId(`run-parameters-field-${field.name}`)}
      className="space-y-4 rounded-md border border-gray-200 dark:border-gray-700 p-3"
    >
      {parameterDefinitions.map((parameterField) => {
        const parameterName = parameterField.name!;
        return (
          <ConfigurationFieldRenderer
            key={parameterName}
            field={parameterField}
            value={parameterValues[parameterName]}
            onChange={(nextValue) => {
              onChange({
                ...parameterValues,
                [parameterName]: nextValue,
              });
            }}
            allValues={allValues}
            domainId={domainId}
            domainType={domainType}
            organizationId={organizationId}
            allowExpressions={allowExpressions}
            autocompleteExampleObj={autocompleteExampleObj}
            validationErrors={validationErrors}
            fieldPath={`${baseFieldPath}.${parameterName}`}
            readOnly={readOnly}
          />
        );
      })}
    </div>
  );
}
