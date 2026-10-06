import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import editorWorkerUrl from "monaco-editor/esm/vs/editor/editor.worker?worker&url";
import cssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker";
import cssWorkerUrl from "monaco-editor/esm/vs/language/css/css.worker?worker&url";
import htmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker";
import htmlWorkerUrl from "monaco-editor/esm/vs/language/html/html.worker?worker&url";
import jsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker";
import jsonWorkerUrl from "monaco-editor/esm/vs/language/json/json.worker?worker&url";
import tsWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker";
import tsWorkerUrl from "monaco-editor/esm/vs/language/typescript/ts.worker?worker&url";

import "monaco-editor/esm/vs/editor/editor.main";
import { setupMonacoEditor } from "./lib/monacoWorkers";

setupMonacoEditor(
  monaco,
  loader,
  {
    editor: editorWorker,
    json: jsonWorker,
    css: cssWorker,
    html: htmlWorker,
    typescript: tsWorker,
  },
  {
    editor: editorWorkerUrl,
    json: jsonWorkerUrl,
    css: cssWorkerUrl,
    html: htmlWorkerUrl,
    typescript: tsWorkerUrl,
  },
);
