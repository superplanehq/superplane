import { afterEach, describe, expect, it, vi } from "vitest";

describe("monaco bootstrap", () => {
  afterEach(() => {
    delete (globalThis as typeof globalThis & { MonacoEnvironment?: unknown }).MonacoEnvironment;
    vi.resetModules();
    vi.doUnmock("@monaco-editor/react");
    vi.doUnmock("monaco-editor");
    vi.doUnmock("monaco-editor/esm/vs/language/css/css.worker?worker");
    vi.doUnmock("monaco-editor/esm/vs/language/html/html.worker?worker");
    vi.doUnmock("monaco-editor/esm/vs/language/json/json.worker?worker");
    vi.doUnmock("monaco-editor/esm/vs/language/typescript/ts.worker?worker");
    vi.doUnmock("monaco-editor/esm/vs/editor/editor.worker?worker");
  });

  it("configures Monaco to use bundled workers instead of the CDN loader", async () => {
    const loaderConfig = vi.fn();
    const monacoInstance = { editor: {} };

    const cssWorker = vi.fn(function MockCssWorker(this: object) {
      return { kind: "css-worker" };
    });
    const htmlWorker = vi.fn(function MockHtmlWorker(this: object) {
      return { kind: "html-worker" };
    });
    const jsonWorker = vi.fn(function MockJsonWorker(this: object) {
      return { kind: "json-worker" };
    });
    const tsWorker = vi.fn(function MockTsWorker(this: object) {
      return { kind: "ts-worker" };
    });
    const editorWorker = vi.fn(function MockEditorWorker(this: object) {
      return { kind: "editor-worker" };
    });

    vi.doMock("@monaco-editor/react", () => ({
      loader: { config: loaderConfig },
    }));
    vi.doMock("monaco-editor", () => monacoInstance);
    vi.doMock("monaco-editor/esm/vs/language/css/css.worker?worker", () => ({ default: cssWorker }));
    vi.doMock("monaco-editor/esm/vs/language/html/html.worker?worker", () => ({ default: htmlWorker }));
    vi.doMock("monaco-editor/esm/vs/language/json/json.worker?worker", () => ({ default: jsonWorker }));
    vi.doMock("monaco-editor/esm/vs/language/typescript/ts.worker?worker", () => ({ default: tsWorker }));
    vi.doMock("monaco-editor/esm/vs/editor/editor.worker?worker", () => ({ default: editorWorker }));

    await import("@/lib/monaco");

    expect(loaderConfig).toHaveBeenCalledWith({ monaco: monacoInstance });

    const monacoEnvironment = (
      globalThis as typeof globalThis & {
        MonacoEnvironment?: { getWorker: (_workerId: string, label: string) => unknown };
      }
    ).MonacoEnvironment;

    expect(monacoEnvironment).toBeDefined();
    expect(monacoEnvironment?.getWorker("workerMain.js", "json")).toEqual({ kind: "json-worker" });
    expect(monacoEnvironment?.getWorker("workerMain.js", "css")).toEqual({ kind: "css-worker" });
    expect(monacoEnvironment?.getWorker("workerMain.js", "html")).toEqual({ kind: "html-worker" });
    expect(monacoEnvironment?.getWorker("workerMain.js", "typescript")).toEqual({ kind: "ts-worker" });
    expect(monacoEnvironment?.getWorker("workerMain.js", "yaml")).toEqual({ kind: "editor-worker" });
  });
});
