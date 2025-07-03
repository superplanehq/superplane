import { pathToFileURL, fileURLToPath } from 'node:url';
import { readFile } from 'node:fs/promises';
import { i as isNodeBuiltin, h as hasCJSSyntax } from './vendor-index.0b5b3600.js';
import { normalizeModuleId } from 'vite-node/utils';
import './vendor-index.29282562.js';
import { g as getWorkerState } from './vendor-global.97e4527c.js';
import 'acorn';
import 'node:module';
import 'node:fs';
import 'pathe';
import 'node:assert';
import 'node:process';
import 'node:path';
import 'node:v8';
import 'node:util';
import 'std-env';
import '@vitest/runner/utils';
import '@vitest/utils';

var ModuleFormat = /* @__PURE__ */ ((ModuleFormat2) => {
  ModuleFormat2["Builtin"] = "builtin";
  ModuleFormat2["Commonjs"] = "commonjs";
  ModuleFormat2["Json"] = "json";
  ModuleFormat2["Module"] = "module";
  ModuleFormat2["Wasm"] = "wasm";
  return ModuleFormat2;
})(ModuleFormat || {});

const ESM_RE = /([\s;}]|^)(import[\w,{}\s*]*from|import\s*['"*{]|export\b\s*(?:[*{]|default|class|type|function|const|var|let|async function)|import\.meta\b)/m;
function hasESMSyntax(code) {
  return ESM_RE.test(code);
}
const cache = /* @__PURE__ */ new Map();
async function getPotentialSource(filepath, result) {
  var _a;
  if (!result.url.startsWith("file://") || result.format === "module")
    return null;
  let source = (_a = cache.get(result.url)) == null ? void 0 : _a.source;
  if (source == null)
    source = await readFile(filepath, "utf8");
  return source;
}
function detectESM(url, source) {
  const cached = cache.get(url);
  if (cached)
    return cached.isPseudoESM;
  if (!source)
    return false;
  return hasESMSyntax(source) && !hasCJSSyntax(source);
}
const resolve = async (url, context, next) => {
  var _a;
  const { parentURL } = context;
  const state = getWorkerState();
  const resolver = state == null ? void 0 : state.rpc.resolveId;
  const environment = state == null ? void 0 : state.ctx.environment;
  if (!parentURL || isNodeBuiltin(url) || !resolver || !environment)
    return next(url, context, next);
  const id = normalizeModuleId(url);
  const importer = normalizeModuleId(parentURL);
  const resolved = await resolver(id, importer, environment.transformMode ?? ((_a = environment.environment) == null ? void 0 : _a.transformMode) ?? "ssr");
  let result;
  let filepath;
  if (resolved) {
    const resolvedUrl = pathToFileURL(resolved.id).toString();
    filepath = resolved.id;
    result = {
      url: resolvedUrl,
      shortCircuit: true
    };
  } else {
    const { url: resolvedUrl, format } = await next(url, context, next);
    filepath = fileURLToPath(resolvedUrl);
    result = {
      url: resolvedUrl,
      format,
      shortCircuit: true
    };
  }
  const source = await getPotentialSource(filepath, result);
  const isPseudoESM = detectESM(result.url, source);
  if (typeof source === "string")
    cache.set(result.url, { isPseudoESM, source });
  if (isPseudoESM)
    result.format = ModuleFormat.Module;
  return result;
};
const load = async (url, context, next) => {
  const result = await next(url, context, next);
  const cached = cache.get(url);
  if ((cached == null ? void 0 : cached.isPseudoESM) && result.format !== "module") {
    return {
      source: cached.source,
      format: ModuleFormat.Module
    };
  }
  return result;
};

export { load, resolve };
