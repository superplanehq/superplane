const MONACO_WORKER_ASSET_NAME = /^(?:editor|json|css|html|ts)\.worker-[A-Za-z0-9._-]+\.js$/;
const MONACO_WORKER_HOST = /(?:^|[\\/])(?:editor|json|css|html|ts)\.worker\.js(?:$|[?#])/;
const MONACO_PACKAGE_MODULE = /[\\/]node_modules[\\/]monaco-editor[\\/]/;
const VIRTUAL_MODULE = /^\0/;
const RELEASE_ASSET_BASE = /^\/releases\/[0-9a-f]{40}\/$/;
const RELEASE_ASSET_NAME = /^[A-Za-z0-9._-]{1,200}\.(?:js|mjs|wasm)$/;

export function isMonacoWorkerAsset(filename: string): boolean {
  const assetName = filename.split("/").pop() ?? "";
  return MONACO_WORKER_ASSET_NAME.test(assetName);
}

export function isMonacoWorkerHost(hostId: string): boolean {
  return MONACO_WORKER_HOST.test(hostId);
}

export function releasePrefixFromAssetBase(assetBaseUrl: string | undefined): string | undefined {
  const trimmed = assetBaseUrl?.trim();
  if (!trimmed) {
    return undefined;
  }

  let pathname: string;
  try {
    pathname = new URL(trimmed).pathname;
  } catch {
    return undefined;
  }

  if (!pathname.endsWith("/")) {
    pathname = `${pathname}/`;
  }

  if (!RELEASE_ASSET_BASE.test(pathname)) {
    return undefined;
  }

  return pathname;
}

export function assetBaseMarker(assetBaseUrl: string | undefined): string | undefined {
  if (!releasePrefixFromAssetBase(assetBaseUrl)) {
    return undefined;
  }

  return `${assetBaseUrl?.trim()}\n`;
}

export function renderMonacoBuiltUrl(
  filename: string,
  hostId: string,
  hostType: string,
  assetBaseUrl?: string,
): string | undefined {
  if (hostType !== "js") {
    return undefined;
  }

  const workerRuntime = monacoWorkerAppOriginRuntime(filename, assetBaseUrl);
  if (workerRuntime) {
    return workerRuntime;
  }

  if (!isMonacoWorkerHost(hostId)) {
    return undefined;
  }

  return appOriginRuntime(releaseAssetPath(filename, assetBaseUrl));
}

export function monacoWorkerAppOriginRuntime(filename: string, assetBaseUrl?: string): string | undefined {
  if (!isMonacoWorkerAsset(filename)) {
    return undefined;
  }

  return appOriginRuntime(releaseAssetPath(filename, assetBaseUrl));
}

function releaseAssetPath(filename: string, assetBaseUrl: string | undefined): string | undefined {
  const assetName = filename.split("/").pop() ?? "";
  if (!RELEASE_ASSET_NAME.test(assetName)) {
    return undefined;
  }

  const releasePrefix = releasePrefixFromAssetBase(assetBaseUrl);
  if (!releasePrefix) {
    return `/assets/${assetName}`;
  }

  return `${releasePrefix}assets/${assetName}`;
}

function appOriginRuntime(assetPath: string | undefined): string | undefined {
  if (!assetPath) {
    return undefined;
  }

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
