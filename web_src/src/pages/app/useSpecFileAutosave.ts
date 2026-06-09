import { useCallback, useEffect, useRef } from "react";

import type { CanvasesCanvas } from "@/api-client";
import type { UpdateCanvasConsoleMutationResult } from "@/hooks/useCanvasData";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH, isWorkflowSpecPath } from "./lib/workflow-spec-paths";
import { parseCanvasYamlForImport, parseConsoleYamlForSave } from "./lib/workflow-spec-files";

const SPEC_FILE_AUTOSAVE_DEBOUNCE_MS = 400;

type UseSpecFileAutosaveParams = {
  canvas?: CanvasesCanvas | null;
  isReadOnly: boolean;
  applyLocalWorkflowUpdate: (workflow: CanvasesCanvas) => void;
  handleSaveWorkflow: (
    workflowToSave?: CanvasesCanvas,
    options?: { showToast?: boolean },
  ) => Promise<{ status: "saved" | "replaced" | "stale" } | undefined | void>;
  updateConsoleMutation: UpdateCanvasConsoleMutationResult;
};

/**
 * Auto-saves edits made to the virtual spec files (canvas.yaml / console.yaml)
 * in the Files tab. Unlike regular repository files, these are materialized
 * into the live canvas/console state and persisted immediately (debounced)
 * instead of waiting for an explicit publish.
 */
export function useSpecFileAutosave({
  canvas,
  isReadOnly,
  applyLocalWorkflowUpdate,
  handleSaveWorkflow,
  updateConsoleMutation,
}: UseSpecFileAutosaveParams) {
  const canvasRef = useRef(canvas);
  canvasRef.current = canvas;
  const isReadOnlyRef = useRef(isReadOnly);
  isReadOnlyRef.current = isReadOnly;
  const applyLocalWorkflowUpdateRef = useRef(applyLocalWorkflowUpdate);
  applyLocalWorkflowUpdateRef.current = applyLocalWorkflowUpdate;
  const handleSaveWorkflowRef = useRef(handleSaveWorkflow);
  handleSaveWorkflowRef.current = handleSaveWorkflow;
  const updateConsoleMutationRef = useRef(updateConsoleMutation);
  updateConsoleMutationRef.current = updateConsoleMutation;

  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      for (const timer of timers.values()) {
        clearTimeout(timer);
      }
      timers.clear();
    };
  }, []);

  const flushCanvasSpec = useCallback((content: string) => {
    const current = canvasRef.current;
    if (!current) return;

    const parsed = parseCanvasYamlForImport(content);
    if (!parsed.ok) return;

    const updatedWorkflow: CanvasesCanvas = {
      ...current,
      spec: { ...current.spec, ...parsed.spec },
    };

    applyLocalWorkflowUpdateRef.current(updatedWorkflow);
    void handleSaveWorkflowRef.current(updatedWorkflow, { showToast: false });
  }, []);

  const flushConsoleSpec = useCallback((content: string) => {
    const parsed = parseConsoleYamlForSave(content);
    if (!parsed.ok) return;

    updateConsoleMutationRef.current.mutate({ panels: parsed.panels, layout: parsed.layout });
  }, []);

  const onSpecFileChange = useCallback(
    (path: string, content: string) => {
      if (isReadOnlyRef.current || !isWorkflowSpecPath(path)) return;

      const existing = timersRef.current.get(path);
      if (existing) clearTimeout(existing);

      const timer = setTimeout(() => {
        timersRef.current.delete(path);
        if (path === CANVAS_YAML_PATH) {
          flushCanvasSpec(content);
          return;
        }
        if (path === CONSOLE_YAML_PATH) {
          flushConsoleSpec(content);
        }
      }, SPEC_FILE_AUTOSAVE_DEBOUNCE_MS);

      timersRef.current.set(path, timer);
    },
    [flushCanvasSpec, flushConsoleSpec],
  );

  return { onSpecFileChange };
}
