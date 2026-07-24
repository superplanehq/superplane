import { useEffect } from "react";
import type { ConfigurationField } from "@/api-client";
import { getSyncedRunParameterValues } from "./runParameters";

interface UseSyncRunParameterValuesOptions {
  readOnly: boolean;
  isLoading: boolean;
  error: unknown;
  appId: string | undefined;
  nodeId: string | undefined;
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
  parameterDefinitions,
  parameterValues,
  onChange,
}: UseSyncRunParameterValuesOptions) {
  useEffect(() => {
    if (readOnly || isLoading || error || !appId || !nodeId) {
      return;
    }

    const nextValues = getSyncedRunParameterValues(parameterValues, parameterDefinitions);
    if (nextValues !== null) {
      onChange(nextValues);
    }
  }, [appId, error, isLoading, nodeId, onChange, parameterDefinitions, parameterValues, readOnly]);
}
