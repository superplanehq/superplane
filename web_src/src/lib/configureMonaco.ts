import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import CssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker";
import HtmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker";
import JsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker";
import TypeScriptWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker";

function createMonacoWorker(label: string): Worker {
  if (label === "json") {
    return new JsonWorker();
  }
  if (label === "css" || label === "scss" || label === "less") {
    return new CssWorker();
  }
  if (label === "html" || label === "handlebars" || label === "razor") {
    return new HtmlWorker();
  }
  if (label === "typescript" || label === "javascript") {
    return new TypeScriptWorker();
  }
  return new EditorWorker();
}

export function configureMonaco(): void {
  globalThis.MonacoEnvironment = {
    getWorker: (_workerId, label) => createMonacoWorker(label),
  };

  loader.config({ monaco });
}
