import type { CanvasesCanvas } from "@/api-client";

export function resolveEditableWorkflowSnapshot({
  isEditing,
  renderedWorkflow,
  detailWorkflow,
}: {
  isEditing: boolean;
  renderedWorkflow: CanvasesCanvas | null | undefined;
  detailWorkflow: CanvasesCanvas | null | undefined;
}): CanvasesCanvas | null | undefined {
  if (isEditing && renderedWorkflow?.spec) {
    return renderedWorkflow;
  }

  return detailWorkflow ?? renderedWorkflow ?? null;
}
