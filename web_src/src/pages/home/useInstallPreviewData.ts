import { useCallback, useEffect, useState } from "react";
import type { InstallParam } from "../install/types";

interface UseInstallPreviewDataOptions {
  repo: string;
  /** When true, skips the preview fetch (caller already loaded data). */
  skipFetch?: boolean;
  preloadedIntegrations?: string[];
  preloadedParams?: InstallParam[];
}

function buildDefaultValues(params: InstallParam[]): Record<string, string> {
  const defaults: Record<string, string> = {};
  for (const p of params) {
    if (p.default) defaults[p.name] = p.default;
  }
  return defaults;
}

export function useInstallPreviewData({
  repo,
  skipFetch,
  preloadedIntegrations,
  preloadedParams,
}: UseInstallPreviewDataOptions) {
  const hasPreloaded = skipFetch ?? Boolean(preloadedIntegrations || preloadedParams);
  const [installParams, setInstallParams] = useState<InstallParam[]>(preloadedParams ?? []);
  const [paramValues, setParamValues] = useState<Record<string, string>>(() =>
    buildDefaultValues(preloadedParams ?? []),
  );
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
          setParamValues(buildDefaultValues(data.installParams));
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
    setParamValues(buildDefaultValues(installParams));
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
