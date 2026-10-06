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

interface MonacoEnvironmentHost {
  MonacoEnvironment?: {
    getWorker?: (workerId: string, label: string) => Worker;
  };
}

export function workerUrlForLabel(workers: MonacoWorkers, label: string): string {
  const key = WORKER_KEY_BY_LABEL[label] ?? "editor";
  return workers[key];
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
  if (import.meta.env.DEV) {
    return new Worker(scriptUrl, { type: "module" });
  }

  if (sharesPageOrigin(scriptUrl)) {
    return new Worker(scriptUrl);
  }

  return new Worker(bootstrapUrl(scriptUrl));
}

function sharesPageOrigin(scriptUrl: string): boolean {
  return new URL(scriptUrl, pageUrl()).origin === pageOrigin();
}

function bootstrapUrl(scriptUrl: string): string {
  const absoluteScriptUrl = new URL(scriptUrl, pageUrl()).href;
  const bootstrap = new Blob([`importScripts(${JSON.stringify(absoluteScriptUrl)})`], {
    type: "text/javascript",
  });
  return URL.createObjectURL(bootstrap);
}

function pageUrl(): string {
  return globalThis.location.href;
}

function pageOrigin(): string {
  return globalThis.location.origin;
}
