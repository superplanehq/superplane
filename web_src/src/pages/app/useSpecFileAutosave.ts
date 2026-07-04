import { useCallback, useEffect, useRef } from "react";

import type { CanvasesCanvas } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePanel, UpdateCanvasConsoleMutationResult } from "@/hooks/useCanvasData";

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
  onEffectiveConsoleChange?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void;
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
  onEffectiveConsoleChange,
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
  const onEffectiveConsoleChangeRef = useRef(onEffectiveConsoleChange);
  onEffectiveConsoleChangeRef.current = onEffectiveConsoleChange;

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

  const applyCanvasSpecLocal = useCallback((content: string) => {
    const current = canvasRef.current;
    if (!current) return;

    const parsed = parseCanvasYamlForImport(content);
    if (!parsed.ok) return;

    applyLocalWorkflowUpdateRef.current({
      ...current,
      spec: { ...current.spec, ...parsed.spec },
    });
  }, []);

  const applyConsoleSpecLocal = useCallback((content: string) => {
    const parsed = parseConsoleYamlForSave(content);
    if (!parsed.ok) return;

    onEffectiveConsoleChangeRef.current?.({ panels: parsed.panels, layout: parsed.layout });
  }, []);

  const persistCanvasSpec = useCallback((content: string) => {
    const current = canvasRef.current;
    if (!current) return;

    const parsed = parseCanvasYamlForImport(content);
    if (!parsed.ok) return;

    const updatedWorkflow: CanvasesCanvas = {
      ...current,
      spec: { ...current.spec, ...parsed.spec },
    };

    void handleSaveWorkflowRef.current(updatedWorkflow, { showToast: false });
  }, []);

  const persistConsoleSpec = useCallback((content: string) => {
    const parsed = parseConsoleYamlForSave(content);
    if (!parsed.ok) return;

    updateConsoleMutationRef.current.mutate({ panels: parsed.panels, layout: parsed.layout });
  }, []);

  const onSpecFileChange = useCallback(
    (path: string, content: string) => {
      if (isReadOnlyRef.current || !isWorkflowSpecPath(path)) return;

      if (path === CANVAS_YAML_PATH) {
        applyCanvasSpecLocal(content);
      } else if (path === CONSOLE_YAML_PATH) {
        applyConsoleSpecLocal(content);
      }

      const existing = timersRef.current.get(path);
      if (existing) clearTimeout(existing);

      const timer = setTimeout(() => {
        timersRef.current.delete(path);
        if (path === CANVAS_YAML_PATH) {
          persistCanvasSpec(content);
          return;
        }
        if (path === CONSOLE_YAML_PATH) {
          persistConsoleSpec(content);
        }
      }, SPEC_FILE_AUTOSAVE_DEBOUNCE_MS);

      timersRef.current.set(path, timer);
    },
    [applyCanvasSpecLocal, applyConsoleSpecLocal, persistCanvasSpec, persistConsoleSpec],
  );

  return { onSpecFileChange };
}
