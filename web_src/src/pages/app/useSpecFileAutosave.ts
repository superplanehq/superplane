import { useCallback, useEffect, useRef } from "react";

import type { CanvasesCanvas } from "@/api-client";
import type { ConsolePage, UpdateCanvasConsoleMutationResult } from "@/hooks/useCanvasData";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH, isWorkflowSpecPath } from "./lib/workflow-spec-paths";
import { parseCanvasYamlForImport, parseConsoleYamlForImport } from "./lib/workflow-spec-files";

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
  onEffectiveConsoleChange?: (next: { pages: ConsolePage[] }) => void;
  /**
   * Called when a spec file fails to parse. This is how the Files tab
   * surfaces silent-drop failures — without it, an invalid edit would
   * disappear on the next debounce without any user-visible signal.
   */
  onSpecParseError?: (path: string, error: string) => void;
};

/**
 * Auto-saves edits made to the virtual spec files (canvas.yaml / console.yaml)
 * in the Files tab. Unlike regular repository files, these are materialized
 * into the live canvas/console state and persisted immediately (debounced)
 * instead of waiting for an explicit publish.
 */
/**
 * Deduplicates parse-error reporting per path. A malformed keystroke
 * fires the autosave callbacks on every debounce; without this, users
 * would get the same toast repeatedly for the same broken state.
 */
function useParseErrorReporter(onSpecParseError?: (path: string, error: string) => void) {
  const onSpecParseErrorRef = useRef(onSpecParseError);
  onSpecParseErrorRef.current = onSpecParseError;
  const lastReportedErrorRef = useRef<Map<string, string>>(new Map());

  const reportParseError = useCallback((path: string, error: string) => {
    if (lastReportedErrorRef.current.get(path) === error) return;
    lastReportedErrorRef.current.set(path, error);
    onSpecParseErrorRef.current?.(path, error);
  }, []);

  const clearParseError = useCallback((path: string) => {
    lastReportedErrorRef.current.delete(path);
  }, []);

  return { reportParseError, clearParseError };
}

export function useSpecFileAutosave({
  canvas,
  isReadOnly,
  applyLocalWorkflowUpdate,
  handleSaveWorkflow,
  updateConsoleMutation,
  onEffectiveConsoleChange,
  onSpecParseError,
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

  const { reportParseError, clearParseError } = useParseErrorReporter(onSpecParseError);

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

  const applyCanvasSpecLocal = useCallback(
    (content: string) => {
      const current = canvasRef.current;
      if (!current) return;

      const parsed = parseCanvasYamlForImport(content);
      if (!parsed.ok) {
        reportParseError(CANVAS_YAML_PATH, parsed.error);
        return;
      }

      clearParseError(CANVAS_YAML_PATH);
      applyLocalWorkflowUpdateRef.current({
        ...current,
        spec: { ...current.spec, ...parsed.spec },
      });
    },
    [clearParseError, reportParseError],
  );

  const applyConsoleSpecLocal = useCallback(
    (content: string) => {
      // Use the lenient parser so grandfathered over-cap consoles keep
      // being previewable while the user edits them down. Structural
      // errors (malformed YAML, unknown fields, duplicate ids) still
      // surface. The backend enforces the cap via a delta check on
      // commit.
      const parsed = parseConsoleYamlForImport(content);
      if (!parsed.ok) {
        reportParseError(CONSOLE_YAML_PATH, parsed.error);
        return;
      }

      clearParseError(CONSOLE_YAML_PATH);
      onEffectiveConsoleChangeRef.current?.({ pages: parsed.pages });
    },
    [clearParseError, reportParseError],
  );

  const persistCanvasSpec = useCallback(
    (content: string) => {
      const current = canvasRef.current;
      if (!current) return;

      const parsed = parseCanvasYamlForImport(content);
      if (!parsed.ok) {
        reportParseError(CANVAS_YAML_PATH, parsed.error);
        return;
      }

      clearParseError(CANVAS_YAML_PATH);
      const updatedWorkflow: CanvasesCanvas = {
        ...current,
        spec: { ...current.spec, ...parsed.spec },
      };

      void handleSaveWorkflowRef.current(updatedWorkflow, { showToast: false });
    },
    [clearParseError, reportParseError],
  );

  const persistConsoleSpec = useCallback(
    (content: string) => {
      const parsed = parseConsoleYamlForImport(content);
      if (!parsed.ok) {
        reportParseError(CONSOLE_YAML_PATH, parsed.error);
        return;
      }

      clearParseError(CONSOLE_YAML_PATH);
      updateConsoleMutationRef.current.mutate({ pages: parsed.pages });
    },
    [clearParseError, reportParseError],
  );

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
