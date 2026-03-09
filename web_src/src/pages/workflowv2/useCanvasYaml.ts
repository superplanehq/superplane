import { useCallback, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { CanvasesCanvas, ComponentsComponent, TriggersTrigger, WidgetsWidget } from "@/api-client/types.gen";
import { canvasKeys } from "@/hooks/useCanvasData";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import type { CanvasNode } from "@/ui/CanvasPage";

interface UseCanvasYamlParams {
  canvas: CanvasesCanvas | undefined;
  organizationId: string | undefined;
  canvasId: string | undefined;
  nodes: CanvasNode[];
  allComponents: ComponentsComponent[];
  allTriggers: TriggersTrigger[];
  widgets: WidgetsWidget[];
  canAutoSave: boolean;
  isReadOnly: boolean;
  getYamlExportPayload: (nodes: CanvasNode[]) => { yamlText: string; filename: string } | null;
  saveWorkflowSnapshot: (currentWorkflow: CanvasesCanvas) => void;
  handleSaveWorkflow: (workflowToSave?: CanvasesCanvas, options?: { showToast?: boolean }) => Promise<boolean>;
  markUnsavedChange: (kind: "structural" | "position") => void;
}

export function useCanvasYaml({
  canvas,
  organizationId,
  canvasId,
  nodes,
  allComponents,
  allTriggers,
  widgets,
  canAutoSave,
  isReadOnly,
  getYamlExportPayload,
  saveWorkflowSnapshot,
  handleSaveWorkflow,
  markUnsavedChange,
}: UseCanvasYamlParams) {
  const queryClient = useQueryClient();
  const [yamlServerError, setYamlServerError] = useState<string | null>(null);

  const yamlPayload = useMemo(() => getYamlExportPayload(nodes), [getYamlExportPayload, nodes]);

  const yamlAutocompleteExampleObj = useMemo(() => {
    const workflowNodes = canvas?.spec?.nodes || [];
    const exampleObj: Record<string, unknown> = {};
    for (const node of workflowNodes) {
      if (!node.name?.trim()) continue;
      const name = node.name.trim();
      if (node.type === "TYPE_TRIGGER") {
        const meta = allTriggers.find((t) => t.name === node.trigger?.name);
        if (meta?.exampleData) exampleObj[name] = meta.exampleData;
      } else {
        const meta = allComponents.find((c) => c.name === node.component?.name);
        if (meta?.exampleOutput) exampleObj[name] = meta.exampleOutput;
      }
    }
    return Object.keys(exampleObj).length > 0 ? exampleObj : null;
  }, [canvas, allComponents, allTriggers]);

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

  const handleYamlChange = useCallback(
    async (parsed: { metadata?: Record<string, unknown>; spec?: Record<string, unknown> }) => {
      if (!canvas || !organizationId || !canvasId) return;

      setYamlServerError(null);
      saveWorkflowSnapshot(canvas);

      const updatedWorkflow = {
        ...canvas,
        metadata: {
          ...canvas.metadata,
          ...(parsed.metadata || {}),
        },
        spec: {
          ...canvas.spec,
          nodes: (parsed.spec as { nodes?: unknown[] })?.nodes || [],
          edges: (parsed.spec as { edges?: unknown[] })?.edges || [],
        },
      };

      // When auto-save is enabled, only update the local query cache after the
      // backend confirms the save. This prevents corrupted state from persisting
      // locally when the server rejects the payload (e.g. proto validation errors).
      // When auto-save is off, the optimistic update is safe because the user will
      // explicitly trigger save later and can see/fix issues before committing.
      if (canAutoSave) {
        const saved = await handleSaveWorkflow(updatedWorkflow as CanvasesCanvas, { showToast: false });
        if (saved) {
          queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);
        }
      } else {
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  return {
    yamlPayload,
    yamlServerError,
    setYamlServerError,
    yamlAutocompleteExampleObj,
    handleYamlViewCopy,
    handleYamlViewDownload,
    handleYamlChange: isReadOnly ? undefined : handleYamlChange,
    allComponents,
    allTriggers,
    widgets,
  };
}
