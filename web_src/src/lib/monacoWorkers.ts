export interface MonacoWorkers {
  editor: string;
  json: string;
  css: string;
  html: string;
  typescript: string;
}

export interface MonacoLoader<Editor> {
  config(options: { monaco: Editor }): void;
}

const WORKER_KEY_BY_LABEL: Record<string, keyof MonacoWorkers> = {
  json: "json",
  css: "css",
  scss: "css",
  less: "css",
  html: "html",
  handlebars: "html",
  razor: "html",
  typescript: "typescript",
  javascript: "typescript",
};

const ASSET_PATH_MARKER = "/assets/";
const RELEASE_ASSET_PATH = /^\/releases\/[0-9a-f]{7,64}\/assets\/[A-Za-z0-9._~-]+$/;

interface MonacoEnvironmentHost {
  MonacoEnvironment?: {
    getWorker?: (workerId: string, label: string) => Worker;
  };
}

export function workerUrlForLabel(workers: MonacoWorkers, label: string): string {
  const key = WORKER_KEY_BY_LABEL[label] ?? "editor";
  return workers[key];
}

export function workerScriptUrl(scriptUrl: string, pageUrl: string): string {
  const page = new URL(pageUrl);
  const worker = new URL(scriptUrl, page);
  if (worker.origin === page.origin) {
    return scriptUrl;
  }

  if (RELEASE_ASSET_PATH.test(worker.pathname)) {
    return new URL(worker.pathname, page.origin).href;
  }

  const markerIndex = worker.pathname.indexOf(ASSET_PATH_MARKER);
  if (markerIndex === -1) {
    return worker.href;
  }

  return new URL(worker.pathname.slice(markerIndex), page.origin).href;
}

export function setupMonacoEditor<Editor>(editor: Editor, loader: MonacoLoader<Editor>, workers: MonacoWorkers): void {
  loader.config({ monaco: editor });

  const host = globalThis as typeof globalThis & MonacoEnvironmentHost;
  const current = host.MonacoEnvironment;
  host.MonacoEnvironment = {
    ...current,
    getWorker(_workerId, label) {
      return createMonacoWorker(workerUrlForLabel(workers, label));
    },
  };
}

function createMonacoWorker(scriptUrl: string): Worker {
  return new Worker(workerScriptUrl(scriptUrl, pageUrl()), { type: "module" });
}

function pageUrl(): string {
  return globalThis.location.href;
}
