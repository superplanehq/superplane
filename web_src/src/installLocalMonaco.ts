import { loader } from "@monaco-editor/react";

import { createLocalMonacoInit, type Cancelable } from "./lib/localMonacoInit";

interface MonacoLoader {
  init: () => Cancelable<unknown>;
}

const monacoLoader = loader as unknown as MonacoLoader;
const originalInit = monacoLoader.init.bind(monacoLoader);

monacoLoader.init = createLocalMonacoInit(
  () => originalInit(),
  () => import("./monacoEditor").then(() => undefined),
);
