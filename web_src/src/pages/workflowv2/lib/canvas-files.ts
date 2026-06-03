import type { WorkflowFile } from "../workflow-files-types";

/** Git-first: repository files only; no virtual canvas.yaml/console.yaml injection. */
export function buildWorkflowFiles(): WorkflowFile[] {
  return [];
}
