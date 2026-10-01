import { loader } from "@monaco-editor/react";

const workerFactories = {
  json: () =>
    new Worker(new URL("monaco-editor/esm/vs/language/json/json.worker.js", import.meta.url), { type: "module" }),
  css: () =>
    new Worker(new URL("monaco-editor/esm/vs/language/css/css.worker.js", import.meta.url), { type: "module" }),
  html: () =>
    new Worker(new URL("monaco-editor/esm/vs/language/html/html.worker.js", import.meta.url), { type: "module" }),
  typescript: () =>
    new Worker(new URL("monaco-editor/esm/vs/language/typescript/ts.worker.js", import.meta.url), { type: "module" }),
  editor: () =>
    new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url), { type: "module" }),
};

export function getMonacoWorkerType(label: string): keyof typeof workerFactories {
  switch (label) {
    case "json":
      return "json";
    case "css":
    case "scss":
    case "less":
      return "css";
    case "html":
    case "handlebars":
    case "razor":
      return "html";
    case "typescript":
    case "javascript":
      return "typescript";
    default:
      return "editor";
  }
}

export async function setupMonaco(): Promise<void> {
  if (typeof window === "undefined") {
    return;
  }

  self.MonacoEnvironment = {
    getWorker: (_moduleId, label) => workerFactories[getMonacoWorkerType(label)](),
  };
  const monaco = await import("monaco-editor");
  loader.config({ monaco });
}
