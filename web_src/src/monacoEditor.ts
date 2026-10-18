import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
import editorWorkerUrl from "monaco-editor/esm/vs/editor/editor.worker?worker&url";
import cssWorkerUrl from "monaco-editor/esm/vs/language/css/css.worker?worker&url";
import htmlWorkerUrl from "monaco-editor/esm/vs/language/html/html.worker?worker&url";
import jsonWorkerUrl from "monaco-editor/esm/vs/language/json/json.worker?worker&url";
import tsWorkerUrl from "monaco-editor/esm/vs/language/typescript/ts.worker?worker&url";

import "monaco-editor/esm/vs/editor/editor.main";
import { setupMonacoEditor } from "./lib/monacoWorkers";

setupMonacoEditor(monaco, loader, {
  editor: editorWorkerUrl,
  json: jsonWorkerUrl,
  css: cssWorkerUrl,
  html: htmlWorkerUrl,
  typescript: tsWorkerUrl,
});
