import { afterEach, describe, expect, it } from "bun:test";

import { setupMonacoEditor, workerConstructorForLabel, type MonacoWorkers } from "./monacoWorkers";

class EditorWorker {}
class JsonWorker {}
class CssWorker {}
class HtmlWorker {}
class TypeScriptWorker {}

const workers = {
  editor: EditorWorker,
  json: JsonWorker,
  css: CssWorker,
  html: HtmlWorker,
  typescript: TypeScriptWorker,
} as unknown as MonacoWorkers;

function monacoEnvironment(): { getWorker: (workerId: string, label: string) => Worker } {
  const environment = (
    globalThis as { MonacoEnvironment?: { getWorker?: (workerId: string, label: string) => Worker } }
  ).MonacoEnvironment;
  if (!environment?.getWorker) {
    throw new Error("MonacoEnvironment.getWorker was not set");
  }
  return { getWorker: environment.getWorker };
}

describe("workerConstructorForLabel", () => {
  it("maps json, css, html, javascript, and typescript to their workers", () => {
    expect(workerConstructorForLabel(workers, "json")).toBe(workers.json);
    expect(workerConstructorForLabel(workers, "css")).toBe(workers.css);
    expect(workerConstructorForLabel(workers, "html")).toBe(workers.html);
    expect(workerConstructorForLabel(workers, "javascript")).toBe(workers.typescript);
    expect(workerConstructorForLabel(workers, "typescript")).toBe(workers.typescript);
  });

  it("uses the editor worker for an unknown label such as yaml", () => {
    expect(workerConstructorForLabel(workers, "yaml")).toBe(workers.editor);
  });
});

describe("setupMonacoEditor", () => {
  const editor = { name: "local-monaco" };

  afterEach(() => {
    delete (globalThis as { MonacoEnvironment?: unknown }).MonacoEnvironment;
  });

  it("gives the loader the local editor and returns mapped workers from getWorker", () => {
    const configured: unknown[] = [];
    const loader = {
      config(options: { monaco: typeof editor }) {
        configured.push(options.monaco);
      },
    };

    setupMonacoEditor(editor, loader, workers);
    setupMonacoEditor(editor, loader, workers);

    expect(configured).toEqual([editor, editor]);
    const { getWorker } = monacoEnvironment();
    expect(getWorker("workerMain.js", "json")).toBeInstanceOf(JsonWorker);
    expect(getWorker("workerMain.js", "css")).toBeInstanceOf(CssWorker);
    expect(getWorker("workerMain.js", "html")).toBeInstanceOf(HtmlWorker);
    expect(getWorker("workerMain.js", "javascript")).toBeInstanceOf(TypeScriptWorker);
    expect(getWorker("workerMain.js", "typescript")).toBeInstanceOf(TypeScriptWorker);
    expect(getWorker("workerMain.js", "yaml")).toBeInstanceOf(EditorWorker);
  });
});
