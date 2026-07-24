import { useMemo } from "react";
import type { ConfigurationField, SuperplaneComponentsNode } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";
import { ConfigurationFieldRenderer } from "./index";
import type { FieldRendererProps, ValidationError } from "./types";
import { coerceRunParameterValues, normalizeRunParameterDefinitions } from "./runParameters";
import { useSyncRunParameterValues } from "./useSyncRunParameterValues";

interface RunParametersFieldRendererProps extends FieldRendererProps {
  organizationId: string;
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

interface RunParametersFieldContentProps {
  field: ConfigurationField;
  appId: string | undefined;
  nodeId: string | undefined;
  isLoading: boolean;
  error: unknown;
  targetNodeResolved: boolean;
  parameterDefinitions: ConfigurationField[];
  parameterValues: Record<string, unknown>;
  baseFieldPath: string;
  onChange: (value: Record<string, unknown>) => void;
  allValues?: Record<string, unknown>;
  organizationId: string;
  allowExpressions: boolean;
  autocompleteExampleObj: FieldRendererProps["autocompleteExampleObj"];
  validationErrors?: ValidationError[] | Set<string>;
  readOnly: boolean;
}

function RunParametersFieldContent({
  field,
  appId,
  nodeId,
  isLoading,
  error,
  targetNodeResolved,
  parameterDefinitions,
  parameterValues,
  baseFieldPath,
  onChange,
  allValues,
  organizationId,
  allowExpressions,
  autocompleteExampleObj,
  validationErrors,
  readOnly,
}: RunParametersFieldContentProps) {
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

  if (!targetNodeResolved) {
    return (
      <div data-testid={toTestId(`run-parameters-field-${field.name}`)} className="space-y-2">
        <p className="text-xs text-gray-500 dark:text-gray-400">
          The selected node was not found in the target app. Choose a different node before configuring run parameters.
        </p>
      </div>
    );
  }

  if (parameterDefinitions.length === 0) {
    return (
      <div
        data-testid={toTestId(`run-parameters-field-${field.name}`)}
        className="rounded-md border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/40"
      >
        <p className="text-xs text-gray-600 dark:text-gray-400">
          The trigger you selected does not define any parameters. If parameters are needed in your flow, define them in
          the trigger configuration first. Without parameters, the run will still be triggered, but no additional values
          will be passed.
        </p>
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

export function RunParametersFieldRenderer({
  field,
  value,
  onChange,
  allValues,
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
  } = useCanvas(organizationId, appId ?? "", {
    enabled: Boolean(appId),
  });

  const targetNode = useMemo(() => findTargetNode(canvas?.spec?.nodes, nodeId), [canvas?.spec?.nodes, nodeId]);
  const targetNodeResolved = Boolean(targetNode);
  const parameterDefinitions = useMemo(
    () => normalizeRunParameterDefinitions(targetNode?.configuration?.parameters),
    [targetNode],
  );
  const parameterValues = useMemo(() => coerceRunParameterValues(value), [value]);
  const baseFieldPath = fieldPath || field.name || "parameters";

  useSyncRunParameterValues({
    readOnly,
    isLoading,
    error,
    appId,
    nodeId,
    targetNodeResolved,
    parameterDefinitions,
    parameterValues,
    onChange,
  });

  return (
    <RunParametersFieldContent
      field={field}
      appId={appId}
      nodeId={nodeId}
      isLoading={isLoading}
      error={error}
      targetNodeResolved={targetNodeResolved}
      parameterDefinitions={parameterDefinitions}
      parameterValues={parameterValues}
      baseFieldPath={baseFieldPath}
      onChange={onChange}
      allValues={allValues}
      organizationId={organizationId}
      allowExpressions={allowExpressions}
      autocompleteExampleObj={autocompleteExampleObj}
      validationErrors={validationErrors}
      readOnly={readOnly}
    />
  );
}
