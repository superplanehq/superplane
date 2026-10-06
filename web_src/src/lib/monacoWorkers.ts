export type MonacoWorkerConstructor = new () => Worker;

export interface MonacoWorkers {
  editor: MonacoWorkerConstructor;
  json: MonacoWorkerConstructor;
  css: MonacoWorkerConstructor;
  html: MonacoWorkerConstructor;
  typescript: MonacoWorkerConstructor;
}

export type MonacoWorkerUrls = Record<keyof MonacoWorkers, string>;

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

function workerKeyForLabel(label: string): keyof MonacoWorkers {
  return WORKER_KEY_BY_LABEL[label] ?? "editor";
}

export function workerConstructorForLabel(workers: MonacoWorkers, label: string): MonacoWorkerConstructor {
  return workers[workerKeyForLabel(label)];
}

function isSecurityError(error: unknown): boolean {
  return error instanceof Error && error.name === "SecurityError";
}

function absoluteWorkerUrl(scriptUrl: string, pageLocation: Pick<Location, "href">): string {
  return new URL(scriptUrl, pageLocation.href).href;
}

function classicWorker(scriptUrl: string, pageLocation: Pick<Location, "href">): Worker {
  const source = `importScripts(${JSON.stringify(absoluteWorkerUrl(scriptUrl, pageLocation))});`;
  const blob = new Blob([source], { type: "application/javascript" });
  return new Worker(URL.createObjectURL(blob));
}

function startWorker(create: MonacoWorkerConstructor, scriptUrl: string): Worker {
  try {
    return new create();
  } catch (error) {
    if (!isSecurityError(error)) {
      throw error;
    }
    return classicWorker(scriptUrl, location);
  }
}

export function setupMonacoEditor<Editor>(
  editor: Editor,
  loader: MonacoLoader<Editor>,
  workers: MonacoWorkers,
  urls: MonacoWorkerUrls,
): void {
  loader.config({ monaco: editor });

  const host = globalThis as typeof globalThis & MonacoEnvironmentHost;
  const current = host.MonacoEnvironment;
  host.MonacoEnvironment = {
    ...current,
    getWorker(_workerId, label) {
      const key = workerKeyForLabel(label);
      return startWorker(workers[key], urls[key]);
    },
  };
}
