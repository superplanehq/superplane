import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas } from "@/api-client";

import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./lib/workflow-spec-paths";
import { materializeCanvasSpec, materializeConsoleSpec } from "./lib/workflow-spec-files";
import { useSpecFileAutosave } from "./useSpecFileAutosave";

const sampleCanvas: CanvasesCanvas = {
  metadata: { id: "canvas-1", name: "Sample", description: "" },
  spec: {
    nodes: [{ id: "node-1", name: "Start", type: "TYPE_TRIGGER", component: "schedule", position: { x: 0, y: 0 } }],
    edges: [],
  },
};

function setup(overrides?: { isReadOnly?: boolean }) {
  const applyLocalWorkflowUpdate = vi.fn();
  const handleSaveWorkflow = vi.fn().mockResolvedValue({ status: "saved" as const });
  const mutate = vi.fn();
  const updateConsoleMutation = { mutate } as never;

  const { result } = renderHook(() =>
    useSpecFileAutosave({
      canvas: sampleCanvas,
      isReadOnly: overrides?.isReadOnly ?? false,
      applyLocalWorkflowUpdate,
      handleSaveWorkflow,
      updateConsoleMutation,
    }),
  );

  return { onSpecFileChange: result.current.onSpecFileChange, applyLocalWorkflowUpdate, handleSaveWorkflow, mutate };
}

describe("useSpecFileAutosave", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("materializes canvas.yaml locally immediately and auto-saves after the debounce", () => {
    const { onSpecFileChange, applyLocalWorkflowUpdate, handleSaveWorkflow } = setup();
    const nextYaml = materializeCanvasSpec({
      ...sampleCanvas,
      spec: { ...sampleCanvas.spec, nodes: [{ ...sampleCanvas.spec!.nodes![0]!, name: "Renamed" }] },
    });

    act(() => onSpecFileChange(CANVAS_YAML_PATH, nextYaml));
    expect(applyLocalWorkflowUpdate).toHaveBeenCalledTimes(1);
    expect(handleSaveWorkflow).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(400));

    expect(applyLocalWorkflowUpdate).toHaveBeenCalledTimes(1);
    expect(handleSaveWorkflow).toHaveBeenCalledTimes(1);
    const saved = handleSaveWorkflow.mock.calls[0]![0] as CanvasesCanvas;
    expect(saved.spec?.nodes?.[0]?.name).toBe("Renamed");
  });

  it("updates console.yaml locally immediately and auto-saves after the debounce", () => {
    const onEffectiveConsoleChange = vi.fn();
    const applyLocalWorkflowUpdate = vi.fn();
    const handleSaveWorkflow = vi.fn().mockResolvedValue({ status: "saved" as const });
    const mutate = vi.fn();
    const updateConsoleMutation = { mutate } as never;

    const { result } = renderHook(() =>
      useSpecFileAutosave({
        canvas: sampleCanvas,
        isReadOnly: false,
        applyLocalWorkflowUpdate,
        handleSaveWorkflow,
        updateConsoleMutation,
        onEffectiveConsoleChange,
      }),
    );

    const consoleYaml = materializeConsoleSpec({
      panels: [{ id: "p1", type: "markdown", content: { body: "hi" } }],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
      canvasId: "canvas-1",
    });

    act(() => result.current.onSpecFileChange(CONSOLE_YAML_PATH, consoleYaml));
    expect(onEffectiveConsoleChange).toHaveBeenCalledTimes(1);
    expect(mutate).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(400));

    expect(mutate).toHaveBeenCalledTimes(1);
    const payload = mutate.mock.calls[0]![0] as { panels: unknown[]; layout: unknown[] };
    expect(payload.panels).toHaveLength(1);
    expect(payload.layout).toHaveLength(1);
  });

  it("debounces rapid edits into a single save", () => {
    const { onSpecFileChange, handleSaveWorkflow } = setup();
    const yamlText = materializeCanvasSpec(sampleCanvas);

    act(() => {
      onSpecFileChange(CANVAS_YAML_PATH, yamlText);
      vi.advanceTimersByTime(100);
      onSpecFileChange(CANVAS_YAML_PATH, yamlText);
      vi.advanceTimersByTime(100);
      onSpecFileChange(CANVAS_YAML_PATH, yamlText);
    });
    act(() => vi.advanceTimersByTime(400));

    expect(handleSaveWorkflow).toHaveBeenCalledTimes(1);
  });

  it("ignores invalid YAML", () => {
    const { onSpecFileChange, applyLocalWorkflowUpdate, handleSaveWorkflow } = setup();

    act(() => onSpecFileChange(CANVAS_YAML_PATH, "::: not valid yaml :::"));
    act(() => vi.advanceTimersByTime(400));

    expect(applyLocalWorkflowUpdate).not.toHaveBeenCalled();
    expect(handleSaveWorkflow).not.toHaveBeenCalled();
  });

  it("does not save when read-only", () => {
    const { onSpecFileChange, handleSaveWorkflow, mutate } = setup({ isReadOnly: true });
    const yamlText = materializeCanvasSpec(sampleCanvas);

    act(() => onSpecFileChange(CANVAS_YAML_PATH, yamlText));
    act(() => vi.advanceTimersByTime(400));

    expect(handleSaveWorkflow).not.toHaveBeenCalled();
    expect(mutate).not.toHaveBeenCalled();
  });
});
