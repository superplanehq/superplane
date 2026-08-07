import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const loaderConfig = vi.hoisted(() => vi.fn());
const workerConstructors = vi.hoisted(() => {
  class EditorWorker {}
  class JsonWorker {}
  class CssWorker {}
  class HtmlWorker {}
  class TypeScriptWorker {}

  return { EditorWorker, JsonWorker, CssWorker, HtmlWorker, TypeScriptWorker };
});

vi.mock("@monaco-editor/react", () => ({ loader: { config: loaderConfig } }));
vi.mock("monaco-editor", () => ({ editor: {} }));
vi.mock("monaco-editor/esm/vs/editor/editor.worker?worker", () => ({ default: workerConstructors.EditorWorker }));
vi.mock("monaco-editor/esm/vs/language/json/json.worker?worker", () => ({ default: workerConstructors.JsonWorker }));
vi.mock("monaco-editor/esm/vs/language/css/css.worker?worker", () => ({ default: workerConstructors.CssWorker }));
vi.mock("monaco-editor/esm/vs/language/html/html.worker?worker", () => ({ default: workerConstructors.HtmlWorker }));
vi.mock("monaco-editor/esm/vs/language/typescript/ts.worker?worker", () => ({
  default: workerConstructors.TypeScriptWorker,
}));

import * as monaco from "monaco-editor";
import { configureMonaco } from "./configureMonaco";

describe("configureMonaco", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    globalThis.MonacoEnvironment = undefined;
    configureMonaco();
  });

  afterEach(() => {
    globalThis.MonacoEnvironment = undefined;
  });

  it("gives the React loader the bundled Monaco instance", () => {
    expect(loaderConfig).toHaveBeenCalledWith({ monaco });
  });

  it.each([
    ["json", workerConstructors.JsonWorker],
    ["css", workerConstructors.CssWorker],
    ["scss", workerConstructors.CssWorker],
    ["less", workerConstructors.CssWorker],
    ["html", workerConstructors.HtmlWorker],
    ["handlebars", workerConstructors.HtmlWorker],
    ["razor", workerConstructors.HtmlWorker],
    ["typescript", workerConstructors.TypeScriptWorker],
    ["javascript", workerConstructors.TypeScriptWorker],
    ["python", workerConstructors.EditorWorker],
  ])("uses the bundled worker for %s", (label, WorkerConstructor) => {
    const worker = globalThis.MonacoEnvironment?.getWorker?.("worker-id", label);

    expect(worker).toBeInstanceOf(WorkerConstructor);
  });
});
