import { pathToFileURL, fileURLToPath } from 'node:url';
import vm from 'node:vm';
import { ModuleCacheMap, ViteNodeRunner, DEFAULT_REQUEST_STUBS } from 'vite-node/client';
import { isNodeBuiltin, isPrimitive, toArray, getCachedData, setCacheData, isInternalRequest } from 'vite-node/utils';
import { resolve, isAbsolute, dirname, join, basename, extname, normalize, relative } from 'pathe';
import { processError } from '@vitest/utils/error';
import { d as distDir } from './vendor-paths.84fc7a99.js';
import { g as getWorkerState } from './vendor-global.97e4527c.js';
import { existsSync, readdirSync, statSync, readFileSync } from 'node:fs';
import { getColors, getType } from '@vitest/utils';
import { b as getAllMockableProperties } from './vendor-base.9c08bbd0.js';
import { dirname as dirname$1 } from 'node:path';
import { createRequire, Module } from 'node:module';
import { CSS_LANGS_RE, KNOWN_ASSET_RE } from 'vite-node/constants';

const spyModulePath = resolve(distDir, "spy.js");
class RefTracker {
  idMap = /* @__PURE__ */ new Map();
  mockedValueMap = /* @__PURE__ */ new Map();
  getId(value) {
    return this.idMap.get(value);
  }
  getMockedValue(id) {
    return this.mockedValueMap.get(id);
  }
  track(originalValue, mockedValue) {
    const newId = this.idMap.size;
    this.idMap.set(originalValue, newId);
    this.mockedValueMap.set(newId, mockedValue);
    return newId;
  }
}
function isSpecialProp(prop, parentType) {
  return parentType.includes("Function") && typeof prop === "string" && ["arguments", "callee", "caller", "length", "name"].includes(prop);
}
class VitestMocker {
  constructor(executor) {
    this.executor = executor;
    const context = this.executor.options.context;
    if (context)
      this.primitives = vm.runInContext("({ Object, Error, Function, RegExp, Symbol, Array, Map })", context);
    else
      this.primitives = { Object, Error, Function, RegExp, Symbol: globalThis.Symbol, Array, Map };
    const Symbol2 = this.primitives.Symbol;
    this.filterPublicKeys = ["__esModule", Symbol2.asyncIterator, Symbol2.hasInstance, Symbol2.isConcatSpreadable, Symbol2.iterator, Symbol2.match, Symbol2.matchAll, Symbol2.replace, Symbol2.search, Symbol2.split, Symbol2.species, Symbol2.toPrimitive, Symbol2.toStringTag, Symbol2.unscopables];
  }
  static pendingIds = [];
  spyModule;
  resolveCache = /* @__PURE__ */ new Map();
  primitives;
  filterPublicKeys;
  get root() {
    return this.executor.options.root;
  }
  get mockMap() {
    return this.executor.options.mockMap;
  }
  get moduleCache() {
    return this.executor.moduleCache;
  }
  get moduleDirectories() {
    return this.executor.options.moduleDirectories || [];
  }
  async initializeSpyModule() {
    this.spyModule = await this.executor.executeId(spyModulePath);
  }
  deleteCachedItem(id) {
    const mockId = this.getMockPath(id);
    if (this.moduleCache.has(mockId))
      this.moduleCache.delete(mockId);
  }
  isAModuleDirectory(path) {
    return this.moduleDirectories.some((dir) => path.includes(dir));
  }
  getSuiteFilepath() {
    return this.executor.state.filepath || "global";
  }
  createError(message) {
    const Error2 = this.primitives.Error;
    return new Error2(message);
  }
  getMocks() {
    const suite = this.getSuiteFilepath();
    const suiteMocks = this.mockMap.get(suite);
    const globalMocks = this.mockMap.get("global");
    return {
      ...globalMocks,
      ...suiteMocks
    };
  }
  async resolvePath(rawId, importer) {
    let id;
    let fsPath;
    try {
      [id, fsPath] = await this.executor.originalResolveUrl(rawId, importer);
    } catch (error) {
      if (error.code === "ERR_MODULE_NOT_FOUND") {
        const { id: unresolvedId } = error[Symbol.for("vitest.error.not_found.data")];
        id = unresolvedId;
        fsPath = unresolvedId;
      } else {
        throw error;
      }
    }
    const external = !isAbsolute(fsPath) || this.isAModuleDirectory(fsPath) ? rawId : null;
    return {
      id,
      fsPath,
      external
    };
  }
  async resolveMocks() {
    if (!VitestMocker.pendingIds.length)
      return;
    await Promise.all(VitestMocker.pendingIds.map(async (mock) => {
      const { fsPath, external } = await this.resolvePath(mock.id, mock.importer);
      if (mock.type === "unmock")
        this.unmockPath(fsPath);
      if (mock.type === "mock")
        this.mockPath(mock.id, fsPath, external, mock.factory);
    }));
    VitestMocker.pendingIds = [];
  }
  async callFunctionMock(dep, mock) {
    var _a, _b;
    const cached = (_a = this.moduleCache.get(dep)) == null ? void 0 : _a.exports;
    if (cached)
      return cached;
    let exports;
    try {
      exports = await mock();
    } catch (err) {
      const vitestError = this.createError(
        '[vitest] There was an error when mocking a module. If you are using "vi.mock" factory, make sure there are no top level variables inside, since this call is hoisted to top of the file. Read more: https://vitest.dev/api/vi.html#vi-mock'
      );
      vitestError.cause = err;
      throw vitestError;
    }
    const filepath = dep.slice(5);
    const mockpath = ((_b = this.resolveCache.get(this.getSuiteFilepath())) == null ? void 0 : _b[filepath]) || filepath;
    if (exports === null || typeof exports !== "object")
      throw this.createError(`[vitest] vi.mock("${mockpath}", factory?: () => unknown) is not returning an object. Did you mean to return an object with a "default" key?`);
    const moduleExports = new Proxy(exports, {
      get: (target, prop) => {
        const val = target[prop];
        if (prop === "then") {
          if (target instanceof Promise)
            return target.then.bind(target);
        } else if (!(prop in target)) {
          if (this.filterPublicKeys.includes(prop))
            return void 0;
          const c = getColors();
          throw this.createError(
            `[vitest] No "${String(prop)}" export is defined on the "${mockpath}" mock. Did you forget to return it from "vi.mock"?
If you need to partially mock a module, you can use "vi.importActual" inside:

${c.green(`vi.mock("${mockpath}", async () => {
  const actual = await vi.importActual("${mockpath}")
  return {
    ...actual,
    // your mocked methods
  },
})`)}
`
          );
        }
        return val;
      }
    });
    this.moduleCache.set(dep, { exports: moduleExports });
    return moduleExports;
  }
  getMockPath(dep) {
    return `mock:${dep}`;
  }
  getDependencyMock(id) {
    return this.getMocks()[id];
  }
  normalizePath(path) {
    return this.moduleCache.normalizePath(path);
  }
  resolveMockPath(mockPath, external) {
    const path = external || mockPath;
    if (external || isNodeBuiltin(mockPath) || !existsSync(mockPath)) {
      const mockDirname = dirname(path);
      const mockFolder = join(this.root, "__mocks__", mockDirname);
      if (!existsSync(mockFolder))
        return null;
      const files = readdirSync(mockFolder);
      const baseOriginal = basename(path);
      for (const file of files) {
        const baseFile = basename(file, extname(file));
        if (baseFile === baseOriginal)
          return resolve(mockFolder, file);
      }
      return null;
    }
    const dir = dirname(path);
    const baseId = basename(path);
    const fullPath = resolve(dir, "__mocks__", baseId);
    return existsSync(fullPath) ? fullPath : null;
  }
  mockObject(object, mockExports = {}) {
    const finalizers = new Array();
    const refs = new RefTracker();
    const define = (container, key, value) => {
      try {
        container[key] = value;
        return true;
      } catch {
        return false;
      }
    };
    const mockPropertiesOf = (container, newContainer) => {
      const containerType = getType(container);
      const isModule = containerType === "Module" || !!container.__esModule;
      for (const { key: property, descriptor } of getAllMockableProperties(container, isModule, this.primitives)) {
        if (!isModule && descriptor.get) {
          try {
            Object.defineProperty(newContainer, property, descriptor);
          } catch (error) {
          }
          continue;
        }
        if (isSpecialProp(property, containerType))
          continue;
        const value = container[property];
        const refId = refs.getId(value);
        if (refId !== void 0) {
          finalizers.push(() => define(newContainer, property, refs.getMockedValue(refId)));
          continue;
        }
        const type = getType(value);
        if (Array.isArray(value)) {
          define(newContainer, property, []);
          continue;
        }
        const isFunction = type.includes("Function") && typeof value === "function";
        if ((!isFunction || value.__isMockFunction) && type !== "Object" && type !== "Module") {
          define(newContainer, property, value);
          continue;
        }
        if (!define(newContainer, property, isFunction ? value : {}))
          continue;
        if (isFunction) {
          const spyModule = this.spyModule;
          if (!spyModule)
            throw this.createError("[vitest] `spyModule` is not defined. This is Vitest error. Please open a new issue with reproduction.");
          const mock = spyModule.spyOn(newContainer, property).mockImplementation(() => void 0);
          mock.mockRestore = () => {
            mock.mockReset();
            mock.mockImplementation(() => void 0);
            return mock;
          };
          Object.defineProperty(newContainer[property], "length", { value: 0 });
        }
        refs.track(value, newContainer[property]);
        mockPropertiesOf(value, newContainer[property]);
      }
    };
    const mockedObject = mockExports;
    mockPropertiesOf(object, mockedObject);
    for (const finalizer of finalizers)
      finalizer();
    return mockedObject;
  }
  unmockPath(path) {
    const suitefile = this.getSuiteFilepath();
    const id = this.normalizePath(path);
    const mock = this.mockMap.get(suitefile);
    if (mock && id in mock)
      delete mock[id];
    this.deleteCachedItem(id);
  }
  mockPath(originalId, path, external, factory) {
    const suitefile = this.getSuiteFilepath();
    const id = this.normalizePath(path);
    const mocks = this.mockMap.get(suitefile) || {};
    const resolves = this.resolveCache.get(suitefile) || {};
    mocks[id] = factory || this.resolveMockPath(path, external);
    resolves[id] = originalId;
    this.mockMap.set(suitefile, mocks);
    this.resolveCache.set(suitefile, resolves);
    this.deleteCachedItem(id);
  }
  async importActual(rawId, importee) {
    const { id, fsPath } = await this.resolvePath(rawId, importee);
    const result = await this.executor.cachedRequest(id, fsPath, [importee]);
    return result;
  }
  async importMock(rawId, importee) {
    const { id, fsPath, external } = await this.resolvePath(rawId, importee);
    const normalizedId = this.normalizePath(fsPath);
    let mock = this.getDependencyMock(normalizedId);
    if (mock === void 0)
      mock = this.resolveMockPath(fsPath, external);
    if (mock === null) {
      const mod = await this.executor.cachedRequest(id, fsPath, [importee]);
      return this.mockObject(mod);
    }
    if (typeof mock === "function")
      return this.callFunctionMock(fsPath, mock);
    return this.executor.dependencyRequest(mock, mock, [importee]);
  }
  async requestWithMock(url, callstack) {
    const id = this.normalizePath(url);
    const mock = this.getDependencyMock(id);
    const mockPath = this.getMockPath(id);
    if (mock === null) {
      const cache = this.moduleCache.get(mockPath);
      if (cache.exports)
        return cache.exports;
      const exports = {};
      this.moduleCache.set(mockPath, { exports });
      const mod = await this.executor.directRequest(url, url, callstack);
      this.mockObject(mod, exports);
      return exports;
    }
    if (typeof mock === "function" && !callstack.includes(mockPath) && !callstack.includes(url)) {
      try {
        callstack.push(mockPath);
        return await this.callFunctionMock(mockPath, mock);
      } finally {
        const indexMock = callstack.indexOf(mockPath);
        callstack.splice(indexMock, 1);
      }
    }
    if (typeof mock === "string" && !callstack.includes(mock))
      return mock;
  }
  queueMock(id, importer, factory) {
    VitestMocker.pendingIds.push({ type: "mock", id, importer, factory });
  }
  queueUnmock(id, importer) {
    VitestMocker.pendingIds.push({ type: "unmock", id, importer });
  }
}

const _require = createRequire(import.meta.url);
const requiresCache = /* @__PURE__ */ new WeakMap();
class CommonjsExecutor {
  context;
  requireCache = /* @__PURE__ */ new Map();
  publicRequireCache = this.createProxyCache();
  moduleCache = /* @__PURE__ */ new Map();
  builtinCache = /* @__PURE__ */ Object.create(null);
  extensions = /* @__PURE__ */ Object.create(null);
  fs;
  Module;
  constructor(options) {
    this.context = options.context;
    this.fs = options.fileMap;
    const primitives = vm.runInContext("({ Object, Array, Error })", this.context);
    const executor = this;
    this.Module = class Module$1 {
      exports;
      isPreloading = false;
      id;
      filename;
      loaded;
      parent;
      children = [];
      path;
      paths = [];
      constructor(id = "", parent) {
        this.exports = primitives.Object.create(Object.prototype);
        this.path = dirname(id);
        this.id = id;
        this.filename = id;
        this.loaded = false;
        this.parent = parent;
      }
      get require() {
        const require = requiresCache.get(this);
        if (require)
          return require;
        const _require2 = Module$1.createRequire(this.id);
        requiresCache.set(this, _require2);
        return _require2;
      }
      _compile(code, filename) {
        const cjsModule = Module$1.wrap(code);
        const script = new vm.Script(cjsModule, {
          filename,
          importModuleDynamically: options.importModuleDynamically
        });
        script.identifier = filename;
        const fn = script.runInContext(executor.context);
        const __dirname = dirname(filename);
        executor.requireCache.set(filename, this);
        try {
          fn(this.exports, this.require, this, filename, __dirname);
          return this.exports;
        } finally {
          this.loaded = true;
        }
      }
      // exposed for external use, Node.js does the opposite
      static _load = (request, parent, _isMain) => {
        const require = Module$1.createRequire((parent == null ? void 0 : parent.filename) ?? request);
        return require(request);
      };
      static wrap = (script) => {
        return Module$1.wrapper[0] + script + Module$1.wrapper[1];
      };
      static wrapper = new primitives.Array(
        "(function (exports, require, module, __filename, __dirname) { ",
        "\n});"
      );
      static builtinModules = Module.builtinModules;
      static findSourceMap = Module.findSourceMap;
      static SourceMap = Module.SourceMap;
      static syncBuiltinESMExports = Module.syncBuiltinESMExports;
      static _cache = executor.moduleCache;
      static _extensions = executor.extensions;
      static createRequire = (filename) => {
        return executor.createRequire(filename);
      };
      static runMain = () => {
        throw new primitives.Error('[vitest] "runMain" is not implemented.');
      };
      // @ts-expect-error not typed
      static _resolveFilename = Module._resolveFilename;
      // @ts-expect-error not typed
      static _findPath = Module._findPath;
      // @ts-expect-error not typed
      static _initPaths = Module._initPaths;
      // @ts-expect-error not typed
      static _preloadModules = Module._preloadModules;
      // @ts-expect-error not typed
      static _resolveLookupPaths = Module._resolveLookupPaths;
      // @ts-expect-error not typed
      static globalPaths = Module.globalPaths;
      // eslint-disable-next-line @typescript-eslint/prefer-ts-expect-error
      // @ts-ignore not typed in lower versions
      static isBuiltin = Module.isBuiltin;
      static Module = Module$1;
    };
    this.extensions[".js"] = this.requireJs;
    this.extensions[".json"] = this.requireJson;
  }
  requireJs = (m, filename) => {
    const content = this.fs.readFile(filename);
    m._compile(content, filename);
  };
  requireJson = (m, filename) => {
    const code = this.fs.readFile(filename);
    m.exports = JSON.parse(code);
  };
  createRequire = (filename) => {
    const _require2 = createRequire(filename);
    const require = (id) => {
      const resolved = _require2.resolve(id);
      const ext = extname(resolved);
      if (ext === ".node" || isNodeBuiltin(resolved))
        return this.requireCoreModule(resolved);
      const module = new this.Module(resolved);
      return this.loadCommonJSModule(module, resolved);
    };
    require.resolve = _require2.resolve;
    Object.defineProperty(require, "extensions", {
      get: () => this.extensions,
      set: () => {
      },
      configurable: true
    });
    require.main = void 0;
    require.cache = this.publicRequireCache;
    return require;
  };
  createProxyCache() {
    return new Proxy(/* @__PURE__ */ Object.create(null), {
      defineProperty: () => true,
      deleteProperty: () => true,
      set: () => true,
      get: (_, key) => this.requireCache.get(key),
      has: (_, key) => this.requireCache.has(key),
      ownKeys: () => Array.from(this.requireCache.keys()),
      getOwnPropertyDescriptor() {
        return {
          configurable: true,
          enumerable: true
        };
      }
    });
  }
  // very naive implementation for Node.js require
  loadCommonJSModule(module, filename) {
    const cached = this.requireCache.get(filename);
    if (cached)
      return cached.exports;
    const extension = this.findLongestRegisteredExtension(filename);
    const loader = this.extensions[extension] || this.extensions[".js"];
    loader(module, filename);
    return module.exports;
  }
  findLongestRegisteredExtension(filename) {
    const name = basename(filename);
    let currentExtension;
    let index;
    let startIndex = 0;
    while ((index = name.indexOf(".", startIndex)) !== -1) {
      startIndex = index + 1;
      if (index === 0)
        continue;
      currentExtension = name.slice(index);
      if (this.extensions[currentExtension])
        return currentExtension;
    }
    return ".js";
  }
  require(identifier) {
    const ext = extname(identifier);
    if (ext === ".node" || isNodeBuiltin(identifier))
      return this.requireCoreModule(identifier);
    const module = new this.Module(identifier);
    return this.loadCommonJSModule(module, identifier);
  }
  requireCoreModule(identifier) {
    const normalized = identifier.replace(/^node:/, "");
    if (this.builtinCache[normalized])
      return this.builtinCache[normalized].exports;
    const moduleExports = _require(identifier);
    if (identifier === "node:module" || identifier === "module") {
      const module = new this.Module("/module.js");
      module.exports = this.Module;
      this.builtinCache[normalized] = module;
      return module.exports;
    }
    this.builtinCache[normalized] = _require.cache[normalized];
    return moduleExports;
  }
}

function interopCommonJsModule(interopDefault, mod) {
  if (isPrimitive(mod) || Array.isArray(mod) || mod instanceof Promise) {
    return {
      keys: [],
      moduleExports: {},
      defaultExport: mod
    };
  }
  if (interopDefault !== false && "__esModule" in mod && !isPrimitive(mod.default)) {
    const defaultKets = Object.keys(mod.default);
    const moduleKeys = Object.keys(mod);
    const allKeys = /* @__PURE__ */ new Set([...defaultKets, ...moduleKeys]);
    allKeys.delete("default");
    return {
      keys: Array.from(allKeys),
      moduleExports: new Proxy(mod, {
        get(mod2, prop) {
          var _a;
          return mod2[prop] ?? ((_a = mod2.default) == null ? void 0 : _a[prop]);
        }
      }),
      defaultExport: mod
    };
  }
  return {
    keys: Object.keys(mod).filter((key) => key !== "default"),
    moduleExports: mod,
    defaultExport: mod
  };
}
const SyntheticModule$1 = vm.SyntheticModule;
const SourceTextModule = vm.SourceTextModule;

const dataURIRegex = /^data:(?<mime>text\/javascript|application\/json|application\/wasm)(?:;(?<encoding>charset=utf-8|base64))?,(?<code>.*)$/;
class EsmExecutor {
  constructor(executor, options) {
    this.executor = executor;
    this.context = options.context;
  }
  moduleCache = /* @__PURE__ */ new Map();
  esmLinkMap = /* @__PURE__ */ new WeakMap();
  context;
  async evaluateModule(m) {
    if (m.status === "unlinked") {
      this.esmLinkMap.set(
        m,
        m.link(
          (identifier, referencer) => this.executor.resolveModule(identifier, referencer.identifier)
        )
      );
    }
    await this.esmLinkMap.get(m);
    if (m.status === "linked")
      await m.evaluate();
    return m;
  }
  async createEsModule(fileUrl, code) {
    const cached = this.moduleCache.get(fileUrl);
    if (cached)
      return cached;
    if (fileUrl.endsWith(".json")) {
      const m2 = new SyntheticModule$1(
        ["default"],
        () => {
          const result = JSON.parse(code);
          m2.setExport("default", result);
        }
      );
      this.moduleCache.set(fileUrl, m2);
      return m2;
    }
    const m = new SourceTextModule(
      code,
      {
        identifier: fileUrl,
        context: this.context,
        importModuleDynamically: this.executor.importModuleDynamically,
        initializeImportMeta: (meta, mod) => {
          meta.url = mod.identifier;
          meta.resolve = (specifier, importer) => {
            return this.executor.resolve(specifier, importer ?? mod.identifier);
          };
        }
      }
    );
    this.moduleCache.set(fileUrl, m);
    return m;
  }
  async loadWebAssemblyModule(source, identifier) {
    const cached = this.moduleCache.get(identifier);
    if (cached)
      return cached;
    const wasmModule = await WebAssembly.compile(source);
    const exports = WebAssembly.Module.exports(wasmModule);
    const imports = WebAssembly.Module.imports(wasmModule);
    const moduleLookup = {};
    for (const { module } of imports) {
      if (moduleLookup[module] === void 0) {
        const resolvedModule = await this.executor.resolveModule(
          module,
          identifier
        );
        moduleLookup[module] = await this.evaluateModule(resolvedModule);
      }
    }
    const syntheticModule = new SyntheticModule$1(
      exports.map(({ name }) => name),
      () => {
        const importsObject = {};
        for (const { module, name } of imports) {
          if (!importsObject[module])
            importsObject[module] = {};
          importsObject[module][name] = moduleLookup[module].namespace[name];
        }
        const wasmInstance = new WebAssembly.Instance(
          wasmModule,
          importsObject
        );
        for (const { name } of exports)
          syntheticModule.setExport(name, wasmInstance.exports[name]);
      },
      { context: this.context, identifier }
    );
    return syntheticModule;
  }
  cacheModule(identifier, module) {
    this.moduleCache.set(identifier, module);
  }
  resolveCachedModule(identifier) {
    return this.moduleCache.get(identifier);
  }
  async createDataModule(identifier) {
    const cached = this.moduleCache.get(identifier);
    if (cached)
      return cached;
    const match = identifier.match(dataURIRegex);
    if (!match || !match.groups)
      throw new Error("Invalid data URI");
    const mime = match.groups.mime;
    const encoding = match.groups.encoding;
    if (mime === "application/wasm") {
      if (!encoding)
        throw new Error("Missing data URI encoding");
      if (encoding !== "base64")
        throw new Error(`Invalid data URI encoding: ${encoding}`);
      const module = await this.loadWebAssemblyModule(
        Buffer.from(match.groups.code, "base64"),
        identifier
      );
      this.moduleCache.set(identifier, module);
      return module;
    }
    let code = match.groups.code;
    if (!encoding || encoding === "charset=utf-8")
      code = decodeURIComponent(code);
    else if (encoding === "base64")
      code = Buffer.from(code, "base64").toString();
    else
      throw new Error(`Invalid data URI encoding: ${encoding}`);
    if (mime === "application/json") {
      const module = new SyntheticModule$1(
        ["default"],
        () => {
          const obj = JSON.parse(code);
          module.setExport("default", obj);
        },
        { context: this.context, identifier }
      );
      this.moduleCache.set(identifier, module);
      return module;
    }
    return this.createEsModule(identifier, code);
  }
}

const CLIENT_ID = "/@vite/client";
const CLIENT_FILE = pathToFileURL(CLIENT_ID).href;
class ViteExecutor {
  constructor(options) {
    this.options = options;
    this.esm = options.esmExecutor;
  }
  esm;
  resolve = (identifier, parent) => {
    if (identifier === CLIENT_ID) {
      if (this.workerState.environment.transformMode === "web")
        return identifier;
      const packageName = this.getPackageName(parent);
      throw new Error(
        `[vitest] Vitest cannot handle ${CLIENT_ID} imported in ${parent} when running in SSR environment. Add "${packageName}" to "ssr.noExternal" if you are using Vite SSR, or to "server.deps.inline" if you are using Vite Node.`
      );
    }
  };
  get workerState() {
    return this.options.context.__vitest_worker__;
  }
  getPackageName(modulePath) {
    const path = normalize(modulePath);
    let name = path.split("/node_modules/").pop() || "";
    if (name == null ? void 0 : name.startsWith("@"))
      name = name.split("/").slice(0, 2).join("/");
    else
      name = name.split("/")[0];
    return name;
  }
  async createViteModule(fileUrl) {
    if (fileUrl === CLIENT_FILE)
      return this.createViteClientModule();
    const cached = this.esm.resolveCachedModule(fileUrl);
    if (cached)
      return cached;
    const result = await this.options.transform(fileUrl, "web");
    if (!result.code)
      throw new Error(`[vitest] Failed to transform ${fileUrl}. Does the file exist?`);
    return this.esm.createEsModule(fileUrl, result.code);
  }
  createViteClientModule() {
    const identifier = CLIENT_ID;
    const cached = this.esm.resolveCachedModule(identifier);
    if (cached)
      return cached;
    const stub = this.options.viteClientModule;
    const moduleKeys = Object.keys(stub);
    const module = new SyntheticModule$1(
      moduleKeys,
      () => {
        moduleKeys.forEach((key) => {
          module.setExport(key, stub[key]);
        });
      },
      { context: this.options.context, identifier }
    );
    this.esm.cacheModule(identifier, module);
    return module;
  }
  canResolve = (fileUrl) => {
    var _a;
    const transformMode = this.workerState.environment.transformMode;
    if (transformMode !== "web")
      return false;
    if (fileUrl === CLIENT_FILE)
      return true;
    const config = ((_a = this.workerState.config.deps) == null ? void 0 : _a.web) || {};
    const [modulePath] = fileUrl.split("?");
    if (config.transformCss && CSS_LANGS_RE.test(modulePath))
      return true;
    if (config.transformAssets && KNOWN_ASSET_RE.test(modulePath))
      return true;
    if (toArray(config.transformGlobPattern).some((pattern) => pattern.test(modulePath)))
      return true;
    return false;
  };
}

const SyntheticModule = vm.SyntheticModule;
const nativeResolve = import.meta.resolve;
class ExternalModulesExecutor {
  constructor(options) {
    this.options = options;
    this.context = options.context;
    this.fs = options.fileMap;
    this.esm = new EsmExecutor(this, {
      context: this.context
    });
    this.cjs = new CommonjsExecutor({
      context: this.context,
      importModuleDynamically: this.importModuleDynamically,
      fileMap: options.fileMap
    });
    this.vite = new ViteExecutor({
      esmExecutor: this.esm,
      context: this.context,
      transform: options.transform,
      viteClientModule: options.requestStubs["/@vite/client"]
    });
    this.resolvers = [this.vite.resolve];
  }
  cjs;
  esm;
  vite;
  context;
  fs;
  resolvers = [];
  // dynamic import can be used in both ESM and CJS, so we have it in the executor
  importModuleDynamically = async (specifier, referencer) => {
    const module = await this.resolveModule(specifier, referencer.identifier);
    return this.esm.evaluateModule(module);
  };
  resolveModule = async (specifier, referencer) => {
    const identifier = await this.resolve(specifier, referencer);
    return await this.createModule(identifier);
  };
  async resolve(specifier, parent) {
    for (const resolver of this.resolvers) {
      const id = resolver(specifier, parent);
      if (id)
        return id;
    }
    return nativeResolve(specifier, parent);
  }
  findNearestPackageData(basedir) {
    var _a;
    const originalBasedir = basedir;
    const packageCache = this.options.packageCache;
    while (basedir) {
      const cached = getCachedData(packageCache, basedir, originalBasedir);
      if (cached)
        return cached;
      const pkgPath = join(basedir, "package.json");
      try {
        if ((_a = statSync(pkgPath, { throwIfNoEntry: false })) == null ? void 0 : _a.isFile()) {
          const pkgData = JSON.parse(this.fs.readFile(pkgPath));
          if (packageCache)
            setCacheData(packageCache, pkgData, basedir, originalBasedir);
          return pkgData;
        }
      } catch {
      }
      const nextBasedir = dirname$1(basedir);
      if (nextBasedir === basedir)
        break;
      basedir = nextBasedir;
    }
    return {};
  }
  wrapCoreSynteticModule(identifier, exports) {
    const moduleKeys = Object.keys(exports);
    const m = new SyntheticModule(
      [...moduleKeys, "default"],
      () => {
        for (const key of moduleKeys)
          m.setExport(key, exports[key]);
        m.setExport("default", exports);
      },
      {
        context: this.context,
        identifier
      }
    );
    return m;
  }
  wrapCommonJsSynteticModule(identifier, exports) {
    const { keys, moduleExports, defaultExport } = interopCommonJsModule(this.options.interopDefault, exports);
    const m = new SyntheticModule(
      [...keys, "default"],
      () => {
        for (const key of keys)
          m.setExport(key, moduleExports[key]);
        m.setExport("default", defaultExport);
      },
      {
        context: this.context,
        identifier
      }
    );
    return m;
  }
  getModuleInformation(identifier) {
    if (identifier.startsWith("data:"))
      return { type: "data", url: identifier, path: identifier };
    const extension = extname(identifier);
    if (extension === ".node" || isNodeBuiltin(identifier))
      return { type: "builtin", url: identifier, path: identifier };
    const isFileUrl = identifier.startsWith("file://");
    const pathUrl = isFileUrl ? fileURLToPath(identifier.split("?")[0]) : identifier;
    const fileUrl = isFileUrl ? identifier : pathToFileURL(pathUrl).toString();
    let type;
    if (this.vite.canResolve(fileUrl)) {
      type = "vite";
    } else if (extension === ".mjs") {
      type = "module";
    } else if (extension === ".cjs") {
      type = "commonjs";
    } else {
      const pkgData = this.findNearestPackageData(normalize(pathUrl));
      type = pkgData.type === "module" ? "module" : "commonjs";
    }
    return { type, path: pathUrl, url: fileUrl };
  }
  async createModule(identifier) {
    const { type, url, path } = this.getModuleInformation(identifier);
    switch (type) {
      case "data":
        return this.esm.createDataModule(identifier);
      case "builtin": {
        const exports = this.require(identifier);
        return this.wrapCoreSynteticModule(identifier, exports);
      }
      case "vite":
        return await this.vite.createViteModule(url);
      case "module":
        return await this.esm.createEsModule(url, this.fs.readFile(path));
      case "commonjs": {
        const exports = this.require(path);
        return this.wrapCommonJsSynteticModule(identifier, exports);
      }
      default: {
        const _deadend = type;
        return _deadend;
      }
    }
  }
  async import(identifier) {
    const module = await this.createModule(identifier);
    await this.esm.evaluateModule(module);
    return module.namespace;
  }
  require(identifier) {
    return this.cjs.require(identifier);
  }
  createRequire(identifier) {
    return this.cjs.createRequire(identifier);
  }
}

class FileMap {
  fsCache = /* @__PURE__ */ new Map();
  fsBufferCache = /* @__PURE__ */ new Map();
  readFile(path) {
    const cached = this.fsCache.get(path);
    if (cached)
      return cached;
    const source = readFileSync(path, "utf-8");
    this.fsCache.set(path, source);
    return source;
  }
  readBuffer(path) {
    const cached = this.fsBufferCache.get(path);
    if (cached)
      return cached;
    const buffer = readFileSync(path);
    this.fsBufferCache.set(path, buffer);
    return buffer;
  }
}

const entryUrl = pathToFileURL(resolve(distDir, "entry.js")).href;
async function createVitestExecutor(options) {
  const runner = new VitestExecutor(options);
  await runner.executeId("/@vite/env");
  await runner.mocker.initializeSpyModule();
  return runner;
}
let _viteNode;
const packageCache = /* @__PURE__ */ new Map();
const moduleCache = new ModuleCacheMap();
const mockMap = /* @__PURE__ */ new Map();
const fileMap = new FileMap();
async function startViteNode(options) {
  if (_viteNode)
    return _viteNode;
  const executor = await startVitestExecutor(options);
  const { run } = await import(entryUrl);
  _viteNode = { run, executor };
  return _viteNode;
}
async function startVitestExecutor(options) {
  const state = () => getWorkerState() || options.state;
  const rpc = () => state().rpc;
  const processExit = process.exit;
  process.exit = (code = process.exitCode || 0) => {
    const error = new Error(`process.exit called with "${code}"`);
    rpc().onWorkerExit(error, code);
    return processExit(code);
  };
  function catchError(err, type) {
    var _a;
    const worker = state();
    const error = processError(err);
    if (!isPrimitive(error)) {
      error.VITEST_TEST_NAME = (_a = worker.current) == null ? void 0 : _a.name;
      if (worker.filepath)
        error.VITEST_TEST_PATH = relative(state().config.root, worker.filepath);
      error.VITEST_AFTER_ENV_TEARDOWN = worker.environmentTeardownRun;
    }
    rpc().onUnhandledError(error, type);
  }
  process.on("uncaughtException", (e) => catchError(e, "Uncaught Exception"));
  process.on("unhandledRejection", (e) => catchError(e, "Unhandled Rejection"));
  const getTransformMode = () => {
    return state().environment.transformMode ?? "ssr";
  };
  return await createVitestExecutor({
    fetchModule(id) {
      return rpc().fetch(id, getTransformMode());
    },
    resolveId(id, importer) {
      return rpc().resolveId(id, importer, getTransformMode());
    },
    transform(id) {
      return rpc().transform(id, "web");
    },
    packageCache,
    moduleCache,
    mockMap,
    get interopDefault() {
      return state().config.deps.interopDefault;
    },
    get moduleDirectories() {
      return state().config.deps.moduleDirectories;
    },
    get root() {
      return state().config.root;
    },
    get base() {
      return state().config.base;
    },
    ...options
  });
}
function updateStyle(id, css) {
  if (typeof document === "undefined")
    return;
  const element = document.querySelector(`[data-vite-dev-id="${id}"]`);
  if (element) {
    element.textContent = css;
    return;
  }
  const head = document.querySelector("head");
  const style = document.createElement("style");
  style.setAttribute("type", "text/css");
  style.setAttribute("data-vite-dev-id", id);
  style.textContent = css;
  head == null ? void 0 : head.appendChild(style);
}
function removeStyle(id) {
  if (typeof document === "undefined")
    return;
  const sheet = document.querySelector(`[data-vite-dev-id="${id}"]`);
  if (sheet)
    document.head.removeChild(sheet);
}
class VitestExecutor extends ViteNodeRunner {
  constructor(options) {
    super({
      ...options,
      // interop is done inside the external executor instead
      interopDefault: options.context ? false : options.interopDefault
    });
    this.options = options;
    this.mocker = new VitestMocker(this);
    if (!options.context) {
      Object.defineProperty(globalThis, "__vitest_mocker__", {
        value: this.mocker,
        writable: true,
        configurable: true
      });
      const clientStub = { ...DEFAULT_REQUEST_STUBS["@vite/client"], updateStyle, removeStyle };
      this.options.requestStubs = {
        "/@vite/client": clientStub,
        "@vite/client": clientStub
      };
      this.primitives = {
        Object,
        Reflect,
        Symbol
      };
    } else {
      const clientStub = vm.runInContext(
        `(defaultClient) => ({ ...defaultClient, updateStyle: ${updateStyle.toString()}, removeStyle: ${removeStyle.toString()} })`,
        options.context
      )(DEFAULT_REQUEST_STUBS["@vite/client"]);
      this.options.requestStubs = {
        "/@vite/client": clientStub,
        "@vite/client": clientStub
      };
      this.primitives = vm.runInContext("({ Object, Reflect, Symbol })", options.context);
      this.externalModules = new ExternalModulesExecutor({
        ...options,
        fileMap,
        context: options.context,
        packageCache: options.packageCache
      });
    }
  }
  mocker;
  externalModules;
  primitives;
  getContextPrimitives() {
    return this.primitives;
  }
  get state() {
    return getWorkerState() || this.options.state;
  }
  shouldResolveId(id, _importee) {
    var _a;
    if (isInternalRequest(id) || id.startsWith("data:"))
      return false;
    const transformMode = ((_a = this.state.environment) == null ? void 0 : _a.transformMode) ?? "ssr";
    return transformMode === "ssr" ? !isNodeBuiltin(id) : !id.startsWith("node:");
  }
  async originalResolveUrl(id, importer) {
    return super.resolveUrl(id, importer);
  }
  async resolveUrl(id, importer) {
    if (VitestMocker.pendingIds.length)
      await this.mocker.resolveMocks();
    if (importer && importer.startsWith("mock:"))
      importer = importer.slice(5);
    try {
      return await super.resolveUrl(id, importer);
    } catch (error) {
      if (error.code === "ERR_MODULE_NOT_FOUND") {
        const { id: id2 } = error[Symbol.for("vitest.error.not_found.data")];
        const path = this.mocker.normalizePath(id2);
        const mock = this.mocker.getDependencyMock(path);
        if (mock !== void 0)
          return [id2, id2];
      }
      throw error;
    }
  }
  async runModule(context, transformed) {
    const vmContext = this.options.context;
    if (!vmContext || !this.externalModules)
      return super.runModule(context, transformed);
    const codeDefinition = `'use strict';async (${Object.keys(context).join(",")})=>{{`;
    const code = `${codeDefinition}${transformed}
}}`;
    const options = {
      filename: context.__filename,
      lineOffset: 0,
      columnOffset: -codeDefinition.length
    };
    const fn = vm.runInContext(code, vmContext, {
      ...options,
      // if we encountered an import, it's not inlined
      importModuleDynamically: this.externalModules.importModuleDynamically
    });
    await fn(...Object.values(context));
  }
  async importExternalModule(path) {
    if (this.externalModules)
      return this.externalModules.import(path);
    return super.importExternalModule(path);
  }
  async dependencyRequest(id, fsPath, callstack) {
    const mocked = await this.mocker.requestWithMock(fsPath, callstack);
    if (typeof mocked === "string")
      return super.dependencyRequest(mocked, mocked, callstack);
    if (mocked && typeof mocked === "object")
      return mocked;
    return super.dependencyRequest(id, fsPath, callstack);
  }
  prepareContext(context) {
    if (this.state.filepath && normalize(this.state.filepath) === normalize(context.__filename)) {
      const globalNamespace = this.options.context || globalThis;
      Object.defineProperty(context.__vite_ssr_import_meta__, "vitest", { get: () => globalNamespace.__vitest_index__ });
    }
    if (this.options.context && this.externalModules)
      context.require = this.externalModules.createRequire(context.__filename);
    return context;
  }
}

export { VitestExecutor as V, mockMap as a, startVitestExecutor as b, moduleCache as m, startViteNode as s };
