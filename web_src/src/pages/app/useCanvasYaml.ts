import { useCallback, useMemo } from "react";
import type { CanvasesCanvas, ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { analytics } from "@/lib/analytics";
import type { CanvasNode } from "@/ui/CanvasPage";
import type { CanvasYamlModalProps } from "./CanvasYamlModal";

interface UseCanvasYamlParams {
  canvasId: string;
  organizationId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isImporting: boolean;
  nodes: CanvasNode[];
  getYamlExportPayload: (nodes: CanvasNode[]) => { yamlText: string; filename: string } | null;
  canvas?: CanvasesCanvas | null;
  isReadOnly: boolean;
  handleSaveWorkflow: (
    workflowToSave?: CanvasesCanvas,
    options?: { showToast?: boolean },
  ) => Promise<{ status: "saved" | "replaced" | "stale" } | undefined | void>;
  onWorkflowImported: (workflow: CanvasesCanvas) => void;
}

export function useCanvasYaml({
  canvasId,
  organizationId,
  open,
  onOpenChange,
  isImporting,
  nodes,
  getYamlExportPayload,
  canvas,
  isReadOnly,
  handleSaveWorkflow,
  onWorkflowImported,
}: UseCanvasYamlParams) {
  const yamlPayload = useMemo(() => getYamlExportPayload(nodes), [getYamlExportPayload, nodes]);

  const importYamlGuardError = !canvas || !organizationId || !canvasId ? "Canvas data is not available" : null;

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

  const handleImportYaml = useCallback(
    async (data: { nodes: unknown[]; edges: unknown[] }) => {
      if (importYamlGuardError) throw new Error(importYamlGuardError);

      const updatedWorkflow = {
        ...canvas!,
        spec: {
          ...canvas!.spec,
          nodes: data.nodes as ComponentsNode[],
          edges: data.edges as ComponentsEdge[],
        },
      };

      const result = await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      if (result?.status !== "saved") {
        throw new Error(getImportFailureMessage(result?.status));
      }

      onWorkflowImported(updatedWorkflow);
    },
    [importYamlGuardError, canvas, handleSaveWorkflow, onWorkflowImported],
  );

  return {
    yamlPayload,
    modalProps: {
      open,
      onOpenChange,
      yamlText: yamlPayload?.yamlText,
      filename: yamlPayload?.filename,
      onCopy: handleYamlViewCopy,
      onDownload: handleYamlViewDownload,
      onImport: isReadOnly ? undefined : handleImportYaml,
      isImporting,
    } satisfies CanvasYamlModalProps,
  };
}

function getImportFailureMessage(status?: "replaced" | "stale") {
  if (status === "stale") {
    return "The canvas changed while importing. Refresh and try again.";
  }

  if (status === "replaced") {
    return "A newer canvas save replaced this import. Try importing again.";
  }

  return "YAML import could not be saved. Try again.";
}
