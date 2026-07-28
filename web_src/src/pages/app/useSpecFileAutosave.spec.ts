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
      pages: [
        {
          id: "main",
          name: "Main",
          panels: [{ id: "p1", type: "markdown", content: { body: "hi" } }],
          layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
        },
      ],
      canvasId: "canvas-1",
    });

    act(() => result.current.onSpecFileChange(CONSOLE_YAML_PATH, consoleYaml));
    expect(onEffectiveConsoleChange).toHaveBeenCalledTimes(1);
    expect(mutate).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(400));

    expect(mutate).toHaveBeenCalledTimes(1);
    const payload = mutate.mock.calls[0]![0] as { pages: { panels: unknown[]; layout: unknown[] }[] };
    expect(payload.pages).toHaveLength(1);
    expect(payload.pages[0]!.panels).toHaveLength(1);
    expect(payload.pages[0]!.layout).toHaveLength(1);
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

  it("surfaces console YAML parse errors via onSpecParseError", () => {
    // Without the callback we silently drop the save; with it, the
    // Files tab can render a toast so the user knows what happened
    // instead of thinking the edit was persisted.
    const onSpecParseError = vi.fn();
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
        onSpecParseError,
      }),
    );

    // Unknown top-level field — a structural error that both the
    // strict and lenient parsers reject.
    const badYaml = "apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  bogus: true\n";

    act(() => result.current.onSpecFileChange(CONSOLE_YAML_PATH, badYaml));
    act(() => vi.advanceTimersByTime(400));

    expect(mutate).not.toHaveBeenCalled();
    expect(onSpecParseError).toHaveBeenCalled();
    expect(onSpecParseError.mock.calls[0]![0]).toBe(CONSOLE_YAML_PATH);
    // Deduplicated: even though the local-apply and persist paths both
    // parse the same bad content, the callback fires once per distinct
    // error per path.
    expect(onSpecParseError).toHaveBeenCalledTimes(1);
  });

  it("stages grandfathered over-cap console content via the lenient parser", async () => {
    // A migrated console with more than MAX_CONSOLE_PANELS_PER_PAGE
    // panels on a single page must still stage through the Files tab;
    // the backend commit path handles the cap check.
    const { MAX_CONSOLE_PANELS_PER_PAGE } = await import("./console/consoleYaml");
    const overCap = MAX_CONSOLE_PANELS_PER_PAGE + 3;
    const overCapPanels = Array.from({ length: overCap }, (_v, i) => ({
      id: `p${i}`,
      type: "markdown",
      content: { body: `panel-${i}` },
    }));
    const overCapLayout = Array.from({ length: overCap }, (_v, i) => ({
      i: `p${i}`,
      x: 0,
      y: i,
      w: 4,
      h: 2,
    }));

    const consoleYaml = materializeConsoleSpec({
      pages: [{ id: "main", name: "Main", panels: overCapPanels, layout: overCapLayout }],
      canvasId: "canvas-1",
    });

    const onSpecParseError = vi.fn();
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
        onSpecParseError,
      }),
    );

    act(() => result.current.onSpecFileChange(CONSOLE_YAML_PATH, consoleYaml));
    act(() => vi.advanceTimersByTime(400));

    expect(onSpecParseError).not.toHaveBeenCalled();
    expect(mutate).toHaveBeenCalledTimes(1);
    const payload = mutate.mock.calls[0]![0] as { pages: { panels: unknown[] }[] };
    expect(payload.pages[0]!.panels).toHaveLength(overCap);
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
