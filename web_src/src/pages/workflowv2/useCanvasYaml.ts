import { useCallback, useMemo } from "react";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import type { CanvasNode } from "@/ui/CanvasPage";

interface UseCanvasYamlParams {
  nodes: CanvasNode[];
  getYamlExportPayload: (nodes: CanvasNode[]) => { yamlText: string; filename: string } | null;
}

export function useCanvasYaml({ nodes, getYamlExportPayload }: UseCanvasYamlParams) {
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
  }, [yamlPayload]);

  return {
    yamlPayload,
    handleYamlViewCopy,
    handleYamlViewDownload,
  };
}
