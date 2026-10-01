const MONACO_WORKER_ASSET_NAME = /^(?:editor|json|css|html|ts)\.worker-[^/]+\.js$/;
const MONACO_PACKAGE_MODULE = /[\\/]node_modules[\\/]monaco-editor[\\/]/;
const VIRTUAL_MODULE = /^\0/;

export function isMonacoWorkerAsset(filename: string): boolean {
  const assetName = filename.split("/").pop() ?? "";
  return MONACO_WORKER_ASSET_NAME.test(assetName);
}

export function monacoWorkerAppOriginRuntime(filename: string): string | undefined {
  if (!isMonacoWorkerAsset(filename)) {
    return undefined;
  }

  const assetPath = filename.startsWith("/") ? filename : `/${filename}`;
  return `new URL(${JSON.stringify(assetPath)}, self.location.origin).href`;
}

export function monacoChunkFileName(chunk: {
  name: string;
  facadeModuleId?: string | null;
  moduleIds: readonly string[];
}): string {
  if (isMonacoChunk(chunk)) {
    return chunk.name === "monaco-editor" ? "assets/[name]-[hash].js" : "assets/monaco-editor-[name]-[hash].js";
  }

  return "assets/[name]-[hash].js";
}

function isMonacoChunk(chunk: { name: string; facadeModuleId?: string | null; moduleIds: readonly string[] }): boolean {
  if (chunk.name === "monaco-editor") {
    return true;
  }

  if (chunk.facadeModuleId && MONACO_PACKAGE_MODULE.test(chunk.facadeModuleId)) {
    return true;
  }

  const packageModules = chunk.moduleIds.filter((moduleId) => !VIRTUAL_MODULE.test(moduleId));
  return packageModules.length > 0 && packageModules.every((moduleId) => MONACO_PACKAGE_MODULE.test(moduleId));
}
