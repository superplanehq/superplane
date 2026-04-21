import { useCallback, useMemo } from "react";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { analytics } from "@/lib/analytics";
import type { CanvasNode } from "@/ui/CanvasPage";

interface UseCanvasYamlParams {
  canvasId: string;
  organizationId: string;
  nodes: CanvasNode[];
  getYamlExportPayload: (nodes: CanvasNode[]) => { yamlText: string; filename: string } | null;
}

export function useCanvasYaml({ canvasId, organizationId, nodes, getYamlExportPayload }: UseCanvasYamlParams) {
  const yamlPayload = useMemo(() => getYamlExportPayload(nodes), [getYamlExportPayload, nodes]);

  const handleYamlViewCopy = useCallback(async () => {
    if (!yamlPayload) return;
    try {
      await navigator.clipboard.writeText(yamlPayload.yamlText);
      showSuccessToast("YAML copied to clipboard");
    } catch {
      showErrorToast("Failed to copy YAML to clipboard");
    }
  }, [yamlPayload]);

  const handleYamlViewDownload = useCallback(() => {
    if (!yamlPayload) return;
    const blob = new Blob([yamlPayload.yamlText], { type: "text/yaml;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = yamlPayload.filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    showSuccessToast("Canvas exported as YAML");
    analytics.yamlExport(canvasId, organizationId);
  }, [yamlPayload, canvasId, organizationId]);

  return {
    yamlPayload,
    handleYamlViewCopy,
    handleYamlViewDownload,
  };
}
