import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import cssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker";
import htmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker";
import jsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker";
import tsWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker";

import "monaco-editor/esm/vs/editor/editor.main";
import { setupMonacoEditor } from "./lib/monacoWorkers";

setupMonacoEditor(monaco, loader, {
  editor: editorWorker,
  json: jsonWorker,
  css: cssWorker,
  html: htmlWorker,
  typescript: tsWorker,
});
