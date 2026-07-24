import { useEffect } from "react";
import type { ConfigurationField } from "@/api-client";
import { getSyncedRunParameterValues } from "./runParameters";

interface UseSyncRunParameterValuesOptions {
  readOnly: boolean;
  isLoading: boolean;
  error: unknown;
  appId: string | undefined;
  nodeId: string | undefined;
  targetNodeResolved: boolean;
  parameterDefinitions: ConfigurationField[];
  parameterValues: Record<string, unknown>;
  onChange: (value: Record<string, unknown>) => void;
}

export function useSyncRunParameterValues({
  readOnly,
  isLoading,
  error,
  appId,
  nodeId,
  targetNodeResolved,
  parameterDefinitions,
  parameterValues,
  onChange,
}: UseSyncRunParameterValuesOptions) {
  useEffect(() => {
    if (readOnly || isLoading || error || !appId || !nodeId || !targetNodeResolved) {
      return;
    }

    const nextValues = getSyncedRunParameterValues(parameterValues, parameterDefinitions, targetNodeResolved);
    if (nextValues !== null) {
      onChange(nextValues);
    }
  }, [appId, error, isLoading, nodeId, onChange, parameterDefinitions, parameterValues, readOnly, targetNodeResolved]);
}
