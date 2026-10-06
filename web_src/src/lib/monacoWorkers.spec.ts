import { afterEach, describe, expect, it } from "bun:test";

import { setupMonacoEditor, workerScriptUrl, workerUrlForLabel, type MonacoWorkers } from "./monacoWorkers";

const ASSET_HOST = "https://assets.superplane.com";
const PRODUCTION_PAGE_URL = "https://app.superplane.com/superplane/workspaces/super-s1pniwje/task/741";
const LOCAL_PAGE_URL = "http://localhost/";

const workers: MonacoWorkers = {
  editor: `${ASSET_HOST}/releases/sha/assets/editor.worker.js`,
  json: `${ASSET_HOST}/releases/sha/assets/json.worker-abc.js`,
  css: `${ASSET_HOST}/releases/sha/assets/css.worker.js`,
  html: `${ASSET_HOST}/releases/sha/assets/html.worker.js`,
  typescript: `${ASSET_HOST}/releases/sha/assets/ts.worker.js`,
};

type StartedWorker = {
  scriptUrl: string | URL;
  options?: WorkerOptions;
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

function installWorkerDouble() {
  const started: StartedWorker[] = [];
  const previousWorker = globalThis.Worker;

  class WorkerDouble {
    constructor(scriptUrl: string | URL, options?: WorkerOptions) {
      started.push({ scriptUrl, options });
    }
  }

  globalThis.Worker = WorkerDouble as unknown as typeof Worker;

  return {
    started,
    restore() {
      globalThis.Worker = previousWorker;
    },
  };
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

describe("workerScriptUrl", () => {
  it("rewrites a cross-origin release URL to the page origin plus /assets/<file>", () => {
    expect(workerScriptUrl(workers.json, PRODUCTION_PAGE_URL)).toBe(
      "https://app.superplane.com/assets/json.worker-abc.js",
    );
  });

  it("keeps a same-origin URL unchanged", () => {
    const sameOriginUrl = "https://app.superplane.com/assets/json.worker-abc.js";
    expect(workerScriptUrl(sameOriginUrl, PRODUCTION_PAGE_URL)).toBe(sameOriginUrl);
  });

  it("keeps a cross-origin URL without /assets/ unchanged", () => {
    const crossOriginUrl = "https://cdn.superplane.com/editor.worker.js";
    expect(workerScriptUrl(crossOriginUrl, PRODUCTION_PAGE_URL)).toBe(crossOriginUrl);
  });
});

describe("setupMonacoEditor", () => {
  const editor = { name: "local-monaco" };
  let restoreDouble: (() => void) | undefined;

  afterEach(() => {
    delete (globalThis as { MonacoEnvironment?: unknown }).MonacoEnvironment;
    restoreDouble?.();
    restoreDouble = undefined;
    window.history.replaceState(null, "", LOCAL_PAGE_URL);
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

  it("starts a cross-origin worker as a module from the page origin", () => {
    window.history.replaceState(null, "", LOCAL_PAGE_URL);
    const doubles = installWorkerDouble();
    restoreDouble = doubles.restore;

    setupMonacoEditor(editor, loaderConfig(), workers);
    const { getWorker } = monacoEnvironment();
    const labels = ["editor", "json", "css", "html", "typescript", "javascript", "yaml"] as const;
    for (const label of labels) {
      getWorker("workerMain.js", label);
    }

    expect(doubles.started).toEqual(
      labels.map((label) => ({
        scriptUrl: workerScriptUrl(workerUrlForLabel(workers, label), LOCAL_PAGE_URL),
        options: { type: "module" },
      })),
    );
  });

  it("starts a same-origin worker as a module with its URL unchanged", () => {
    window.history.replaceState(null, "", LOCAL_PAGE_URL);
    const doubles = installWorkerDouble();
    restoreDouble = doubles.restore;
    const sameOriginWorkers: MonacoWorkers = {
      editor: "http://localhost/assets/editor.worker.js",
      json: "/assets/json.worker.js",
      css: "http://localhost/assets/css.worker.js",
      html: "http://localhost/assets/html.worker.js",
      typescript: "http://localhost/assets/ts.worker.js",
    };

    setupMonacoEditor(editor, loaderConfig(), sameOriginWorkers);
    const { getWorker } = monacoEnvironment();
    getWorker("workerMain.js", "yaml");
    getWorker("workerMain.js", "json");

    expect(doubles.started).toEqual([
      { scriptUrl: sameOriginWorkers.editor, options: { type: "module" } },
      { scriptUrl: sameOriginWorkers.json, options: { type: "module" } },
    ]);
  });
});

function loaderConfig() {
  return { config() {} };
}
