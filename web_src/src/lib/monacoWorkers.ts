export type MonacoWorkerConstructor = new () => Worker;

export interface MonacoWorkers {
  editor: MonacoWorkerConstructor;
  json: MonacoWorkerConstructor;
  css: MonacoWorkerConstructor;
  html: MonacoWorkerConstructor;
  typescript: MonacoWorkerConstructor;
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

export function workerConstructorForLabel(workers: MonacoWorkers, label: string): MonacoWorkerConstructor {
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
      return new (workerConstructorForLabel(workers, label))();
    },
  };
}
