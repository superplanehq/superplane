import { afterEach, describe, expect, it, vi } from "bun:test";

import {
  setupMonacoEditor,
  workerConstructorForLabel,
  type MonacoWorkerUrls,
  type MonacoWorkers,
} from "./monacoWorkers";

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

const workerUrls: MonacoWorkerUrls = {
  editor: "/editor.worker.js",
  json: "/json.worker.js",
  css: "/css.worker.js",
  html: "/html.worker.js",
  typescript: "/ts.worker.js",
};

function monacoEnvironment(): { getWorker: (workerId: string, label: string) => Worker } {
  const environment = (
    globalThis as { MonacoEnvironment?: { getWorker?: (workerId: string, label: string) => Worker } }
  ).MonacoEnvironment;
  if (!environment?.getWorker) {
    throw new Error("MonacoEnvironment.getWorker was not set");
  }
  return { getWorker: environment.getWorker };
}

function installWorkers(constructors: MonacoWorkers, urls: MonacoWorkerUrls) {
  setupMonacoEditor({ name: "local-monaco" }, { config() {} }, constructors, urls);
  return monacoEnvironment().getWorker;
}

function setPageUrl(url: string) {
  const happyDOM = (window as Window & { happyDOM?: { setURL?: (next: string) => void } }).happyDOM;
  if (typeof happyDOM?.setURL !== "function") {
    throw new Error("page URL cannot be set");
  }
  happyDOM.setURL(url);
}

function workerThatThrows(error: Error): MonacoWorkers {
  const create = function BlockedWorker() {
    throw error;
  };
  return {
    editor: create,
    json: create,
    css: create,
    html: create,
    typescript: create,
  } as unknown as MonacoWorkers;
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
  const pageUrl = window.location.href;

  afterEach(() => {
    delete (globalThis as { MonacoEnvironment?: unknown }).MonacoEnvironment;
    setPageUrl(pageUrl);
    vi.restoreAllMocks();
  });

  it("gives the loader the local editor and returns mapped workers from getWorker", () => {
    const configured: unknown[] = [];
    const loader = {
      config(options: { monaco: typeof editor }) {
        configured.push(options.monaco);
      },
    };

    setupMonacoEditor(editor, loader, workers, workerUrls);
    setupMonacoEditor(editor, loader, workers, workerUrls);

    expect(configured).toEqual([editor, editor]);
    const { getWorker } = monacoEnvironment();
    expect(getWorker("workerMain.js", "json")).toBeInstanceOf(JsonWorker);
    expect(getWorker("workerMain.js", "css")).toBeInstanceOf(CssWorker);
    expect(getWorker("workerMain.js", "html")).toBeInstanceOf(HtmlWorker);
    expect(getWorker("workerMain.js", "javascript")).toBeInstanceOf(TypeScriptWorker);
    expect(getWorker("workerMain.js", "typescript")).toBeInstanceOf(TypeScriptWorker);
    expect(getWorker("workerMain.js", "yaml")).toBeInstanceOf(EditorWorker);
  });

  it("creates a same-origin worker with the bundler constructor", () => {
    const createObjectURL = vi.spyOn(URL, "createObjectURL");
    const getWorker = installWorkers(workers, workerUrls);

    expect(getWorker("workerMain.js", "json")).toBeInstanceOf(JsonWorker);
    expect(createObjectURL).not.toHaveBeenCalled();
  });

  it("loads a cross-origin worker through a same-origin importScripts blob", async () => {
    const page = "https://app.superplane.com/superplane/workspaces/super-s1pniwje/task/741";
    const scriptUrl = "https://assets.superplane.com/releases/hash/assets/json.worker.js";
    setPageUrl(page);

    const blobs: Blob[] = [];
    const workerCalls: Array<{ url: string; options?: WorkerOptions }> = [];
    vi.spyOn(URL, "createObjectURL").mockImplementation((value) => {
      blobs.push(value as Blob);
      return "blob:monaco-worker";
    });
    vi.spyOn(URL, "revokeObjectURL");
    vi.spyOn(globalThis, "Worker").mockImplementation((url: string | URL, options?: WorkerOptions) => {
      workerCalls.push({ url: String(url), options });
      return { url: String(url) } as unknown as Worker;
    });

    const getWorker = installWorkers(workerThatThrows(new DOMException("blocked worker", "SecurityError")), {
      ...workerUrls,
      json: scriptUrl,
    });

    expect(getWorker("workerMain.js", "json")).toEqual({ url: "blob:monaco-worker" });
    expect(workerCalls).toEqual([{ url: "blob:monaco-worker", options: undefined }]);
    expect(URL.revokeObjectURL).not.toHaveBeenCalled();
    await expect(blobs[0]?.text()).resolves.toBe(`importScripts(${JSON.stringify(scriptUrl)});`);
  });

  it("resolves a worker URL against the page location before importScripts", async () => {
    const page = "https://app.superplane.com/superplane/workspaces/super-s1pniwje/task/741";
    setPageUrl(page);

    const blobs: Blob[] = [];
    vi.spyOn(URL, "createObjectURL").mockImplementation((value) => {
      blobs.push(value as Blob);
      return "blob:monaco-worker";
    });
    vi.spyOn(globalThis, "Worker").mockImplementation(() => ({}) as Worker);

    const getWorker = installWorkers(workerThatThrows(new DOMException("blocked worker", "SecurityError")), {
      ...workerUrls,
      editor: "/releases/hash/assets/editor.worker.js",
    });

    getWorker("workerMain.js", "yaml");

    await expect(blobs[0]?.text()).resolves.toBe(
      `importScripts(${JSON.stringify("https://app.superplane.com/releases/hash/assets/editor.worker.js")});`,
    );
  });

  it("rethrows a constructor error that is not a SecurityError", () => {
    const createObjectURL = vi.spyOn(URL, "createObjectURL");
    const getWorker = installWorkers(workerThatThrows(new TypeError("worker failed")), workerUrls);

    expect(() => getWorker("workerMain.js", "json")).toThrow(TypeError);
    expect(() => getWorker("workerMain.js", "json")).toThrow("worker failed");
    expect(createObjectURL).not.toHaveBeenCalled();
  });
});
