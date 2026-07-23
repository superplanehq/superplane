import { useMemo } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { toTestId } from "@/lib/testID";
import { ConfigurationFieldRenderer } from "./index";
import type { FieldRendererProps, ValidationError } from "./types";
import { normalizeRunParameterDefinitions } from "./runParameters";

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

  const parameterDefinitions = useMemo(() => {
    return normalizeRunParameterDefinitions(targetNode?.configuration?.parameters);
  }, [targetNode]);

  const parameterValues = useMemo(() => {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return value as Record<string, unknown>;
    }

    return {};
  }, [value]);

  const baseFieldPath = fieldPath || field.name || "parameters";

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
      <div
        data-testid={toTestId(`run-parameters-field-${field.name}`)}
        className="rounded-md border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/40"
      >
        <p className="text-xs text-gray-600 dark:text-gray-400">
        The trigger you selected does not define any parameters.
        If parameters are needed in your flow, define them in the trigger configuration first.
        Without parameters, the run will still be triggered, but no additional values will be passed.
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
