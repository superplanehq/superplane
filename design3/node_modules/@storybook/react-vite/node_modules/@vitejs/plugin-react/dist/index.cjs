'use strict';

const path = require('node:path');
const babel = require('@babel/core');
const vite = require('vite');
const MagicString = require('magic-string');
const fs = require('node:fs');
const node_module = require('node:module');

function _interopNamespaceDefault(e) {
  const n = Object.create(null);
  if (e) {
    for (const k in e) {
      n[k] = e[k];
    }
  }
  n.default = e;
  return n;
}

const babel__namespace = /*#__PURE__*/_interopNamespaceDefault(babel);

const runtimePublicPath = "/@react-refresh";
const _require = node_module.createRequire((typeof document === 'undefined' ? new (require('u' + 'rl').URL)('file:' + __filename).href : (document.currentScript && document.currentScript.src || new URL('index.cjs', document.baseURI).href)));
const reactRefreshDir = path.dirname(
  _require.resolve("react-refresh/package.json")
);
const runtimeFilePath = path.join(
  reactRefreshDir,
  "cjs/react-refresh-runtime.development.js"
);
const runtimeCode = `
const exports = {}
${fs.readFileSync(runtimeFilePath, "utf-8")}
${fs.readFileSync(_require.resolve("./refreshUtils.js"), "utf-8")}
export default exports
`;
const preambleCode = `
import RefreshRuntime from "__BASE__${runtimePublicPath.slice(1)}"
RefreshRuntime.injectIntoGlobalHook(window)
window.$RefreshReg$ = () => {}
window.$RefreshSig$ = () => (type) => type
window.__vite_plugin_react_preamble_installed__ = true
`;
const header = `
import RefreshRuntime from "${runtimePublicPath}";

let prevRefreshReg;
let prevRefreshSig;

if (import.meta.hot) {
  if (!window.__vite_plugin_react_preamble_installed__) {
    throw new Error(
      "@vitejs/plugin-react can't detect preamble. Something is wrong. " +
      "See https://github.com/vitejs/vite-plugin-react/pull/11#discussion_r430879201"
    );
  }

  prevRefreshReg = window.$RefreshReg$;
  prevRefreshSig = window.$RefreshSig$;
  window.$RefreshReg$ = (type, id) => {
    RefreshRuntime.register(type, __SOURCE__ + " " + id)
  };
  window.$RefreshSig$ = RefreshRuntime.createSignatureFunctionForTransform;
}`.replace(/\n+/g, "");
const footer = `
if (import.meta.hot) {
  window.$RefreshReg$ = prevRefreshReg;
  window.$RefreshSig$ = prevRefreshSig;

  import(/* @vite-ignore */ import.meta.url).then((currentExports) => {
    RefreshRuntime.registerExportsForReactRefresh(__SOURCE__, currentExports);
    import.meta.hot.accept((nextExports) => {
      if (!nextExports) return;
      const invalidateMessage = RefreshRuntime.validateRefreshBoundaryAndEnqueueUpdate(currentExports, nextExports);
      if (invalidateMessage) import.meta.hot.invalidate(invalidateMessage);
    });
  });
}`;
function addRefreshWrapper(code, id) {
  return header.replace("__SOURCE__", JSON.stringify(id)) + code + footer.replace("__SOURCE__", JSON.stringify(id));
}

const prependReactImportCode = "import React from 'react'; ";
const refreshContentRE = /\$Refresh(?:Reg|Sig)\$\(/;
function viteReact(opts = {}) {
  let devBase = "/";
  let filter = vite.createFilter(opts.include, opts.exclude);
  let needHiresSourcemap = false;
  let isProduction = true;
  let projectRoot = process.cwd();
  let skipFastRefresh = opts.fastRefresh === false;
  let skipReactImport = false;
  let runPluginOverrides = (options, context) => false;
  let staticBabelOptions;
  const useAutomaticRuntime = opts.jsxRuntime !== "classic";
  const importReactRE = /(?:^|\n)import\s+(?:\*\s+as\s+)?React(?:,|\s+)/;
  const fileExtensionRE = /\.[^/\s?]+$/;
  const viteBabel = {
    name: "vite:react-babel",
    enforce: "pre",
    config(userConfig, { mode }) {
      const resolvedRoot = vite.normalizePath(
        userConfig.root ? path.resolve(userConfig.root) : process.cwd()
      );
      const envDir = userConfig.envDir ? vite.normalizePath(path.resolve(resolvedRoot, userConfig.envDir)) : resolvedRoot;
      vite.loadEnv(mode, envDir, vite.resolveEnvPrefix(userConfig));
      const isProduction2 = (process.env.NODE_ENV || process.env.VITE_USER_NODE_ENV || mode) === "production";
      if (opts.jsxRuntime === "classic") {
        return {
          esbuild: {
            logOverride: {
              "this-is-undefined-in-esm": "silent"
            },
            jsx: "transform",
            jsxImportSource: opts.jsxImportSource,
            jsxSideEffects: opts.jsxPure === false
          }
        };
      } else {
        return {
          esbuild: {
            jsxDev: !isProduction2,
            jsx: "automatic",
            jsxImportSource: opts.jsxImportSource,
            jsxSideEffects: opts.jsxPure === false
          }
        };
      }
    },
    configResolved(config) {
      devBase = config.base;
      projectRoot = config.root;
      filter = vite.createFilter(opts.include, opts.exclude, {
        resolve: projectRoot
      });
      needHiresSourcemap = config.command === "build" && !!config.build.sourcemap;
      isProduction = config.isProduction;
      skipFastRefresh || (skipFastRefresh = isProduction || config.command === "build");
      const jsxInject = config.esbuild && config.esbuild.jsxInject;
      if (jsxInject && importReactRE.test(jsxInject)) {
        skipReactImport = true;
        config.logger.warn(
          "[@vitejs/plugin-react] This plugin imports React for you automatically, so you can stop using `esbuild.jsxInject` for that purpose."
        );
      }
      config.plugins.forEach((plugin) => {
        const hasConflict = plugin.name === "react-refresh" || plugin !== viteReactJsx && plugin.name === "vite:react-jsx";
        if (hasConflict)
          return config.logger.warn(
            `[@vitejs/plugin-react] You should stop using "${plugin.name}" since this plugin conflicts with it.`
          );
      });
      runPluginOverrides = (babelOptions, context) => {
        const hooks = config.plugins.map((plugin) => plugin.api?.reactBabel).filter(Boolean);
        if (hooks.length > 0) {
          return (runPluginOverrides = (babelOptions2, context2) => {
            hooks.forEach((hook) => hook(babelOptions2, context2, config));
            return true;
          })(babelOptions, context);
        }
        runPluginOverrides = () => false;
        return false;
      };
    },
    async transform(code, id, options) {
      const ssr = options?.ssr === true;
      const [filepath, querystring = ""] = id.split("?");
      const [extension = ""] = querystring.match(fileExtensionRE) || filepath.match(fileExtensionRE) || [];
      if (/\.(?:mjs|[tj]sx?)$/.test(extension)) {
        const isJSX = extension.endsWith("x");
        const isNodeModules = id.includes("/node_modules/");
        const isProjectFile = !isNodeModules && (id[0] === "\0" || id.startsWith(projectRoot + "/"));
        let babelOptions = staticBabelOptions;
        if (typeof opts.babel === "function") {
          const rawOptions = opts.babel(id, { ssr });
          babelOptions = createBabelOptions(rawOptions);
          runPluginOverrides(babelOptions, { ssr, id });
        } else if (!babelOptions) {
          babelOptions = createBabelOptions(opts.babel);
          if (!runPluginOverrides(babelOptions, { ssr, id })) {
            staticBabelOptions = babelOptions;
          }
        }
        const plugins = isProjectFile ? [...babelOptions.plugins] : [];
        let useFastRefresh = false;
        if (!skipFastRefresh && !ssr && !isNodeModules) {
          const isReactModule = isJSX || importReactRE.test(code);
          if (isReactModule && filter(id)) {
            useFastRefresh = true;
            plugins.push([
              await loadPlugin("react-refresh/babel"),
              { skipEnvCheck: true }
            ]);
          }
        }
        let prependReactImport = false;
        if (!isProjectFile || isJSX) {
          if (!useAutomaticRuntime && isProjectFile) {
            if (!isProduction) {
              plugins.push(
                await loadPlugin("@babel/plugin-transform-react-jsx-self"),
                await loadPlugin("@babel/plugin-transform-react-jsx-source")
              );
            }
            if (!skipReactImport && !importReactRE.test(code)) {
              prependReactImport = true;
            }
          }
        }
        let inputMap;
        if (prependReactImport) {
          if (needHiresSourcemap) {
            const s = new MagicString(code);
            s.prepend(prependReactImportCode);
            code = s.toString();
            inputMap = s.generateMap({ hires: true, source: id });
          } else {
            code = prependReactImportCode + code;
          }
        }
        const shouldSkip = !plugins.length && !babelOptions.configFile && !(isProjectFile && babelOptions.babelrc);
        if (shouldSkip) {
          return {
            code,
            map: inputMap ?? null
          };
        }
        const parserPlugins = [
          ...babelOptions.parserOpts.plugins,
          "importMeta",
          // This plugin is applied before esbuild transforms the code,
          // so we need to enable some stage 3 syntax that is supported in
          // TypeScript and some environments already.
          "topLevelAwait",
          "classProperties",
          "classPrivateProperties",
          "classPrivateMethods"
        ];
        if (!extension.endsWith(".ts")) {
          parserPlugins.push("jsx");
        }
        if (/\.tsx?$/.test(extension)) {
          parserPlugins.push("typescript");
        }
        const result = await babel__namespace.transformAsync(code, {
          ...babelOptions,
          root: projectRoot,
          filename: id,
          sourceFileName: filepath,
          parserOpts: {
            ...babelOptions.parserOpts,
            sourceType: "module",
            allowAwaitOutsideFunction: true,
            plugins: parserPlugins
          },
          generatorOpts: {
            ...babelOptions.generatorOpts,
            decoratorsBeforeExport: true
          },
          plugins,
          sourceMaps: true,
          // Vite handles sourcemap flattening
          inputSourceMap: inputMap ?? false
        });
        if (result) {
          let code2 = result.code;
          if (useFastRefresh && refreshContentRE.test(code2)) {
            code2 = addRefreshWrapper(code2, id);
          }
          return {
            code: code2,
            map: result.map
          };
        }
      }
    }
  };
  const viteReactRefresh = {
    name: "vite:react-refresh",
    enforce: "pre",
    config: () => ({
      resolve: {
        dedupe: ["react", "react-dom"]
      }
    }),
    resolveId(id) {
      if (id === runtimePublicPath) {
        return id;
      }
    },
    load(id) {
      if (id === runtimePublicPath) {
        return runtimeCode;
      }
    },
    transformIndexHtml() {
      if (!skipFastRefresh)
        return [
          {
            tag: "script",
            attrs: { type: "module" },
            children: preambleCode.replace(`__BASE__`, devBase)
          }
        ];
    }
  };
  const reactJsxRuntimeId = "react/jsx-runtime";
  const reactJsxDevRuntimeId = "react/jsx-dev-runtime";
  const virtualReactJsxRuntimeId = "\0" + reactJsxRuntimeId;
  const virtualReactJsxDevRuntimeId = "\0" + reactJsxDevRuntimeId;
  const viteReactJsx = {
    name: "vite:react-jsx",
    enforce: "pre",
    config() {
      return {
        optimizeDeps: {
          // We can't add `react-dom` because the dependency is `react-dom/client`
          // for React 18 while it's `react-dom` for React 17. We'd need to detect
          // what React version the user has installed.
          include: [reactJsxRuntimeId, reactJsxDevRuntimeId, "react"]
        }
      };
    },
    resolveId(id, importer) {
      if (id === reactJsxRuntimeId && importer !== virtualReactJsxRuntimeId) {
        return virtualReactJsxRuntimeId;
      }
      if (id === reactJsxDevRuntimeId && importer !== virtualReactJsxDevRuntimeId) {
        return virtualReactJsxDevRuntimeId;
      }
    },
    load(id) {
      if (id === virtualReactJsxRuntimeId) {
        return [
          `import * as jsxRuntime from ${JSON.stringify(reactJsxRuntimeId)}`,
          `export const Fragment = jsxRuntime.Fragment`,
          `export const jsx = jsxRuntime.jsx`,
          `export const jsxs = jsxRuntime.jsxs`
        ].join("\n");
      }
      if (id === virtualReactJsxDevRuntimeId) {
        return [
          `import * as jsxRuntime from ${JSON.stringify(reactJsxDevRuntimeId)}`,
          `export const Fragment = jsxRuntime.Fragment`,
          `export const jsxDEV = jsxRuntime.jsxDEV`
        ].join("\n");
      }
    }
  };
  return [viteBabel, viteReactRefresh, useAutomaticRuntime && viteReactJsx];
}
viteReact.preambleCode = preambleCode;
function loadPlugin(path2) {
  return import(path2).then((module) => module.default || module);
}
function createBabelOptions(rawOptions) {
  var _a;
  const babelOptions = {
    babelrc: false,
    configFile: false,
    ...rawOptions
  };
  babelOptions.plugins || (babelOptions.plugins = []);
  babelOptions.presets || (babelOptions.presets = []);
  babelOptions.overrides || (babelOptions.overrides = []);
  babelOptions.parserOpts || (babelOptions.parserOpts = {});
  (_a = babelOptions.parserOpts).plugins || (_a.plugins = []);
  return babelOptions;
}

module.exports = viteReact;
module.exports.default = viteReact;
