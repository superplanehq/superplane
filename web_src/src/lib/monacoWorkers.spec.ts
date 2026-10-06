import { afterEach, describe, expect, it } from "bun:test";

import { setupMonacoEditor, workerUrlForLabel, type MonacoWorkers } from "./monacoWorkers";

const PAGE_URL = "http://localhost/";
const ASSET_HOST = "https://assets.superplane.com";

const workers: MonacoWorkers = {
  editor: `${ASSET_HOST}/releases/sha/assets/editor.worker.js`,
  json: `${ASSET_HOST}/releases/sha/assets/json.worker.js`,
  css: `${ASSET_HOST}/releases/sha/assets/css.worker.js`,
  html: `${ASSET_HOST}/releases/sha/assets/html.worker.js`,
  typescript: `${ASSET_HOST}/releases/sha/assets/ts.worker.js`,
};

type StartedWorker = {
  scriptUrl: string | URL;
  options?: WorkerOptions;
};

type RecordedBlob = {
  parts: BlobPart[];
  type?: string;
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

function setDevelopmentMode(enabled: boolean): void {
  (import.meta.env as { DEV: boolean }).DEV = enabled;
}

function installWorkerDoubles() {
  const started: StartedWorker[] = [];
  const blobs: RecordedBlob[] = [];
  const objectUrls: string[] = [];
  const previousWorker = globalThis.Worker;
  const previousBlob = globalThis.Blob;
  const previousCreateObjectURL = URL.createObjectURL;
  const previousRevokeObjectURL = URL.revokeObjectURL;
  const revoked: string[] = [];
  let sequence = 0;

  class WorkerDouble {
    constructor(
      public scriptUrl: string | URL,
      public options?: WorkerOptions,
    ) {
      started.push({ scriptUrl, options });
    }
  }

  class BlobDouble {
    constructor(
      public parts: BlobPart[] = [],
      options?: BlobPropertyBag,
    ) {
      blobs.push({ parts, type: options?.type });
    }
  }

  globalThis.Worker = WorkerDouble as unknown as typeof Worker;
  globalThis.Blob = BlobDouble as unknown as typeof Blob;
  URL.createObjectURL = () => {
    const objectUrl = `blob:monaco-worker-${sequence}`;
    sequence += 1;
    objectUrls.push(objectUrl);
    return objectUrl;
  };
  URL.revokeObjectURL = (objectUrl: string) => {
    revoked.push(objectUrl);
  };

  return {
    started,
    blobs,
    objectUrls,
    revoked,
    restore() {
      globalThis.Worker = previousWorker;
      globalThis.Blob = previousBlob;
      URL.createObjectURL = previousCreateObjectURL;
      URL.revokeObjectURL = previousRevokeObjectURL;
    },
  };
}

function blobSource(blob: RecordedBlob): string {
  return blob.parts.map(String).join("");
}

function importScriptsSource(scriptUrl: string): string {
  return `importScripts(${JSON.stringify(scriptUrl)})`;
}

describe("workerUrlForLabel", () => {
  it("maps json, css, html, javascript, and typescript to their workers", () => {
    expect(workerUrlForLabel(workers, "json")).toBe(workers.json);
    expect(workerUrlForLabel(workers, "css")).toBe(workers.css);
    expect(workerUrlForLabel(workers, "html")).toBe(workers.html);
    expect(workerUrlForLabel(workers, "javascript")).toBe(workers.typescript);
    expect(workerUrlForLabel(workers, "typescript")).toBe(workers.typescript);
  });

  it("uses the editor worker for an unknown label such as yaml", () => {
    expect(workerUrlForLabel(workers, "yaml")).toBe(workers.editor);
  });
});

describe("setupMonacoEditor", () => {
  const editor = { name: "local-monaco" };
  const previousDev = (import.meta.env as { DEV: boolean }).DEV;
  let restoreDoubles: (() => void) | undefined;

  afterEach(() => {
    delete (globalThis as { MonacoEnvironment?: unknown }).MonacoEnvironment;
    setDevelopmentMode(previousDev);
    restoreDoubles?.();
    restoreDoubles = undefined;
    window.history.replaceState(null, "", PAGE_URL);
  });

  it("gives the loader the local editor", () => {
    const configured: unknown[] = [];
    const loader = {
      config(options: { monaco: typeof editor }) {
        configured.push(options.monaco);
      },
    };

    setupMonacoEditor(editor, loader, workers);
    setupMonacoEditor(editor, loader, workers);

    expect(configured).toEqual([editor, editor]);
  });

  it("starts a cross-origin worker from a same-origin bootstrap script", () => {
    setDevelopmentMode(false);
    window.history.replaceState(null, "", PAGE_URL);
    const doubles = installWorkerDoubles();
    restoreDoubles = doubles.restore;
    const loader = { config() {} };

    setupMonacoEditor(editor, loader, workers);
    const { getWorker } = monacoEnvironment();
    const labels = ["editor", "json", "css", "html", "typescript", "javascript", "yaml"] as const;
    for (const label of labels) {
      getWorker("workerMain.js", label);
    }

    const scriptUrls = labels.map((label) => workerUrlForLabel(workers, label));
    const uniqueScriptUrls = [...new Set(scriptUrls)];
    expect(doubles.objectUrls).toHaveLength(uniqueScriptUrls.length);
    expect(doubles.started.map((worker) => String(worker.scriptUrl))).toEqual(
      scriptUrls.map((scriptUrl) => doubles.objectUrls[uniqueScriptUrls.indexOf(scriptUrl)]),
    );
    expect(doubles.started.every((worker) => worker.options === undefined)).toBe(true);
    for (const scriptUrl of uniqueScriptUrls) {
      expect(doubles.started.map((worker) => String(worker.scriptUrl))).not.toContain(scriptUrl);
    }
    expect(doubles.blobs.map(blobSource)).toEqual(uniqueScriptUrls.map(importScriptsSource));
    expect(doubles.revoked).toEqual([]);
  });

  it("reuses one bootstrap URL when the same worker restarts", () => {
    setDevelopmentMode(false);
    window.history.replaceState(null, "", PAGE_URL);
    const doubles = installWorkerDoubles();
    restoreDoubles = doubles.restore;

    setupMonacoEditor(editor, loaderConfig(), workers);
    const { getWorker } = monacoEnvironment();
    getWorker("workerMain.js", "json");
    getWorker("workerMain.js", "json");
    getWorker("workerMain.js", "javascript");

    expect(doubles.objectUrls).toEqual(["blob:monaco-worker-0", "blob:monaco-worker-1"]);
    expect(doubles.started.map((worker) => String(worker.scriptUrl))).toEqual([
      "blob:monaco-worker-0",
      "blob:monaco-worker-0",
      "blob:monaco-worker-1",
    ]);
    expect(doubles.blobs.map(blobSource)).toEqual([
      importScriptsSource(workers.json),
      importScriptsSource(workers.typescript),
    ]);
    expect(doubles.revoked).toEqual([]);
  });

  it("starts a same-origin production worker directly", () => {
    setDevelopmentMode(false);
    window.history.replaceState(null, "", PAGE_URL);
    const doubles = installWorkerDoubles();
    restoreDoubles = doubles.restore;
    const sameOriginWorkers: MonacoWorkers = {
      editor: "/assets/editor.worker.js",
      json: "http://localhost/assets/json.worker.js",
      css: "http://localhost/assets/css.worker.js",
      html: "http://localhost/assets/html.worker.js",
      typescript: "http://localhost/assets/ts.worker.js",
    };

    setupMonacoEditor(editor, loaderConfig(), sameOriginWorkers);
    const { getWorker } = monacoEnvironment();
    getWorker("workerMain.js", "yaml");
    getWorker("workerMain.js", "json");

    expect(doubles.started).toEqual([
      { scriptUrl: sameOriginWorkers.editor, options: undefined },
      { scriptUrl: sameOriginWorkers.json, options: undefined },
    ]);
    expect(doubles.blobs).toEqual([]);
    expect(doubles.objectUrls).toEqual([]);
  });

  it("starts a module worker in development", () => {
    setDevelopmentMode(true);
    const doubles = installWorkerDoubles();
    restoreDoubles = doubles.restore;

    setupMonacoEditor(editor, loaderConfig(), workers);
    const { getWorker } = monacoEnvironment();
    getWorker("workerMain.js", "typescript");
    getWorker("workerMain.js", "css");

    expect(doubles.started).toEqual([
      { scriptUrl: workers.typescript, options: { type: "module" } },
      { scriptUrl: workers.css, options: { type: "module" } },
    ]);
    expect(doubles.blobs).toEqual([]);
    expect(doubles.objectUrls).toEqual([]);
  });
});

function loaderConfig() {
  return { config() {} };
}
