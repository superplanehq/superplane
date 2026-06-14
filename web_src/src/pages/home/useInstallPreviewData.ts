import { useCallback, useEffect, useState } from "react";
import type { InstallParam } from "../install/types";

interface UseInstallPreviewDataOptions {
  repo: string;
  preloadedIntegrations?: string[];
  preloadedParams?: InstallParam[];
}

export function useInstallPreviewData({ repo, preloadedIntegrations, preloadedParams }: UseInstallPreviewDataOptions) {
  const hasPreloaded = Boolean(preloadedIntegrations || preloadedParams);
  const [installParams, setInstallParams] = useState<InstallParam[]>(preloadedParams ?? []);
  const [paramValues, setParamValues] = useState<Record<string, string>>(() => {
    const defaults: Record<string, string> = {};
    for (const p of preloadedParams ?? []) {
      if (p.default) defaults[p.name] = p.default;
    }
    return defaults;
  });
  const [previewLoading, setPreviewLoading] = useState(!hasPreloaded);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [detectedIntegrations, setDetectedIntegrations] = useState<string[]>(preloadedIntegrations ?? []);

  useEffect(() => {
    if (hasPreloaded) return;
    fetch(`/apps/install/preview?repo=${encodeURIComponent(repo)}`, { credentials: "include" })
      .then((r) => {
        if (!r.ok) throw new Error(`Failed to load app configuration (${r.status})`);
        return r.json();
      })
      .then((data) => {
        if (data.installParams && data.installParams.length > 0) {
          setInstallParams(data.installParams);
          const defaults: Record<string, string> = {};
          for (const p of data.installParams) {
            if (p.default) defaults[p.name] = p.default;
          }
          setParamValues(defaults);
        }
        if (data.integrations && data.integrations.length > 0) {
          setDetectedIntegrations(data.integrations);
        }
      })
      .catch((err) => {
        setPreviewError(err instanceof Error ? err.message : "Failed to load app configuration");
      })
      .finally(() => setPreviewLoading(false));
  }, [repo, hasPreloaded]);

  const resetParamValues = useCallback(() => {
    const defaults: Record<string, string> = {};
    for (const p of installParams) {
      if (p.default) defaults[p.name] = p.default;
    }
    setParamValues(defaults);
  }, [installParams]);

  return {
    installParams,
    paramValues,
    setParamValues,
    resetParamValues,
    previewLoading,
    previewError,
    detectedIntegrations,
  };
}
